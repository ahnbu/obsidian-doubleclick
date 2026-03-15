// obsidian-md-handler: Windows .md 파일 더블클릭 핸들러
// 옵시디언 볼트 내 파일은 Advanced URI로 옵시디언에서 열고,
// 볼트 밖 파일은 fallback 앱(Typora → VS Code → 메모장)으로 열림.
//
// 빌드 (배포용, 콘솔 창 없음):
//   go build -ldflags "-H=windowsgui" -o obsidian-handler.exe .
//
// 빌드 (디버그용, 콘솔 창 있음):
//   go build -o obsidian-handler-debug.exe .
//
// 사용:
//   obsidian-handler.exe [--debug] <파일경로>

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Win32 API
// ---------------------------------------------------------------------------

var (
	shell32 = syscall.NewLazyDLL("shell32.dll")
	user32  = syscall.NewLazyDLL("user32.dll")

	procShellExecuteW   = shell32.NewProc("ShellExecuteW")
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowTextW  = user32.NewProc("GetWindowTextW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procIsIconic        = user32.NewProc("IsIconic")
	procShowWindow      = user32.NewProc("ShowWindow")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
)

const swRestore = 9

// ---------------------------------------------------------------------------
// 설정 구조체
// ---------------------------------------------------------------------------

// obsidianConfig: %APPDATA%\Obsidian\obsidian.json 구조
type obsidianConfig struct {
	Vaults map[string]vaultEntry `json:"vaults"`
}

type vaultEntry struct {
	Path string `json:"path"`
}

// handlerConfig: 핸들러 설치 폴더의 obsidian-handler.config.json (선택)
type handlerConfig struct {
	FallbackCommand string `json:"fallbackCommand,omitempty"`
	ObsidianExePath string `json:"obsidianExePath,omitempty"`
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	debug, filePath := parseArgs(os.Args[1:])

	if filePath == "" {
		writeLog("ERROR: 파일 경로 없음. 사용법: obsidian-handler.exe [--debug] <파일경로>")
		os.Exit(2)
	}
	filePath = filepath.Clean(filePath)

	vaults := loadVaults()
	cfg := loadHandlerConfig()
	vault := findVault(filePath, vaults)

	if debug {
		type debugInfo struct {
			FilePath string       `json:"filePath"`
			Vault    *vaultEntry  `json:"vault"`
			Config   handlerConfig `json:"config"`
			Vaults   []vaultEntry `json:"allVaults"`
		}
		info := debugInfo{
			FilePath: filePath,
			Vault:    vault,
			Config:   cfg,
			Vaults:   vaults,
		}
		data, _ := json.MarshalIndent(info, "", "  ")
		writeLog("DEBUG " + string(data))
		// 디버그 빌드에선 stdout도 출력 (windowsgui 빌드에선 무시됨)
		fmt.Println(string(data))
		return
	}

	switch {
	case vault != nil:
		uri := buildURI(filePath, vault.Path)
		writeLog("LAUNCH obsidian-uri " + uri)
		shellOpen(uri)
		activateObsidian()

	case cfg.FallbackCommand != "":
		writeLog("LAUNCH fallback-config " + cfg.FallbackCommand)
		shellOpenWith(cfg.FallbackCommand, filePath)

	default:
		apps := detectFallbackApps()
		if len(apps) > 0 {
			writeLog("LAUNCH fallback-detected " + apps[0])
			shellOpenWith(apps[0], filePath)
		} else {
			writeLog("WARN no fallback found — 파일을 열 수 없음: " + filePath)
		}
	}
}

// ---------------------------------------------------------------------------
// URI 빌드
// ---------------------------------------------------------------------------

func buildURI(filePath, vaultPath string) string {
	rel, err := filepath.Rel(vaultPath, filePath)
	if err != nil {
		rel = filePath
	}
	rel = filepath.ToSlash(rel)
	vaultName := filepath.Base(vaultPath)
	return "obsidian://advanced-uri?vault=" +
		url.QueryEscape(vaultName) +
		"&filepath=" +
		url.QueryEscape(rel) +
		"&openmode=true"
}

// ---------------------------------------------------------------------------
// 볼트 탐색
// ---------------------------------------------------------------------------

func findVault(filePath string, vaults []vaultEntry) *vaultEntry {
	fileLower := strings.ToLower(filepath.Clean(filePath))
	var best *vaultEntry
	bestLen := 0

	for i, v := range vaults {
		vLower := strings.ToLower(filepath.Clean(v.Path))
		sep := string(filepath.Separator)
		if fileLower == vLower || strings.HasPrefix(fileLower, vLower+sep) {
			if len(vLower) > bestLen {
				bestLen = len(vLower)
				best = &vaults[i]
			}
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// 설정 로드
// ---------------------------------------------------------------------------

func loadVaults() []vaultEntry {
	appdata := os.Getenv("APPDATA")
	configPath := filepath.Join(appdata, "Obsidian", "obsidian.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		writeLog("WARN obsidian.json 읽기 실패: " + configPath)
		return nil
	}

	var cfg obsidianConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		writeLog("WARN obsidian.json 파싱 실패")
		return nil
	}

	result := make([]vaultEntry, 0, len(cfg.Vaults))
	for _, v := range cfg.Vaults {
		if v.Path != "" {
			result = append(result, v)
		}
	}
	return result
}

func loadHandlerConfig() handlerConfig {
	exe, err := os.Executable()
	if err != nil {
		return handlerConfig{}
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "obsidian-handler.config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return handlerConfig{}
	}
	var cfg handlerConfig
	json.Unmarshal(data, &cfg) //nolint:errcheck
	return cfg
}

// ---------------------------------------------------------------------------
// Win32: ShellExecute
// ---------------------------------------------------------------------------

func shellOpen(uri string) {
	verbPtr, _ := syscall.UTF16PtrFromString("open")
	targetPtr, _ := syscall.UTF16PtrFromString(uri)

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		0, 0,
		1, // SW_NORMAL
	)
	if ret <= 32 {
		writeLog(fmt.Sprintf("WARN ShellExecuteW returned %d for %s", ret, uri))
	}
}

func shellOpenWith(app, filePath string) {
	verbPtr, _ := syscall.UTF16PtrFromString("open")
	appPtr, _ := syscall.UTF16PtrFromString(app)
	filePtr, _ := syscall.UTF16PtrFromString(filePath)

	procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(appPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		0,
		1,
	)
}

// ---------------------------------------------------------------------------
// Win32: 창 활성화
// ---------------------------------------------------------------------------

func findObsidianHwnd() uintptr {
	var found uintptr

	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		buf := make([]uint16, 256)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
		title := syscall.UTF16ToString(buf)
		if strings.Contains(title, "Obsidian") {
			found = hwnd
			return 0 // 열거 중단
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

func activateObsidian() {
	time.Sleep(1 * time.Second)

	for i := 0; i < 3; i++ {
		hwnd := findObsidianHwnd()
		if hwnd != 0 {
			iconic, _, _ := procIsIconic.Call(hwnd)
			if iconic != 0 {
				procShowWindow.Call(hwnd, swRestore)
			}
			procSetForeground.Call(hwnd)
			writeLog(fmt.Sprintf("activate-window success attempt=%d", i+1))
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	writeLog("activate-window: Obsidian 창을 찾지 못함 (3회 시도)")
}

// ---------------------------------------------------------------------------
// Fallback 앱 감지
// ---------------------------------------------------------------------------

func detectFallbackApps() []string {
	localAppData := os.Getenv("LOCALAPPDATA")
	winDir := os.Getenv("WINDIR")
	if winDir == "" {
		winDir = `C:\Windows`
	}

	candidates := []string{
		`C:\Program Files\Typora\Typora.exe`,
		filepath.Join(localAppData, `Programs\Microsoft VS Code\Code.exe`),
		`C:\Program Files\Microsoft VS Code\Code.exe`,
		filepath.Join(winDir, `System32\notepad.exe`),
	}

	var found []string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			found = append(found, c)
		}
	}
	return found
}

// ---------------------------------------------------------------------------
// 로그
// ---------------------------------------------------------------------------

func writeLog(msg string) {
	tempDir := os.Getenv("TEMP")
	if tempDir == "" {
		return
	}
	logPath := filepath.Join(tempDir, "obsidian-md-handler.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	fmt.Fprintf(f, "%s %s\n", ts, msg)
}

// ---------------------------------------------------------------------------
// 인자 파싱
// ---------------------------------------------------------------------------

func parseArgs(args []string) (debug bool, filePath string) {
	for _, a := range args {
		if a == "--debug" {
			debug = true
		} else if !strings.HasPrefix(a, "--") {
			filePath = a
		}
	}
	return
}
