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
//   obsidian-handler.exe --doctor
//   obsidian-handler.exe --repair

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
	advapi  = syscall.NewLazyDLL("advapi32.dll")

	procShellExecuteW   = shell32.NewProc("ShellExecuteW")
	procSHChangeNotify  = shell32.NewProc("SHChangeNotify")
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowTextW  = user32.NewProc("GetWindowTextW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procIsIconic        = user32.NewProc("IsIconic")
	procShowWindow      = user32.NewProc("ShowWindow")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
	procRegCreateKeyExW = advapi.NewProc("RegCreateKeyExW")
	procRegSetValueExW  = advapi.NewProc("RegSetValueExW")
)

const (
	swRestore         = 9
	shcneAssocChanged = 0x08000000
	shcnfIDList       = 0
)

const (
	regCommandPath  = `Software\Classes\Applications\Obsidian.exe\shell\open\command`
	regDefaultIcon  = `Software\Classes\Applications\Obsidian.exe\DefaultIcon`
	regApplication  = `Software\Classes\Applications\Obsidian.exe`
	regUserChoice   = `Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.md\UserChoice`
	regDefaultValue = ""
	regFriendlyName = "FriendlyAppName"
	expectedProgID  = `Applications\Obsidian.exe`
	productionExe   = "obsidian-handler.exe"
	friendlyAppName = "Obsidian"
)

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
	opts := parseArgs(os.Args[1:])
	cfg := loadHandlerConfig()

	if opts.doctor || opts.repair {
		report := diagnoseSelfHeal(cfg, opts.repair)
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		if !report.OK {
			os.Exit(1)
		}
		return
	}

	if opts.filePath == "" {
		writeLog("ERROR: 파일 경로 없음. 사용법: obsidian-handler.exe [--debug] <파일경로>")
		os.Exit(2)
	}
	filePath := filepath.Clean(opts.filePath)

	if !opts.debug {
		runSafeSelfHeal(cfg)
	}
	vaults := loadVaults()
	vault := findVault(filePath, vaults)

	if opts.debug {
		type debugInfo struct {
			FilePath string        `json:"filePath"`
			Vault    *vaultEntry   `json:"vault"`
			Config   handlerConfig `json:"config"`
			Vaults   []vaultEntry  `json:"allVaults"`
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

// safeEscape: URI 쿼리 파라미터 인코딩. url.QueryEscape의 공백→"+" 변환을
// "%20"으로 대체한다. obsidian:// URI는 decodeURIComponent로 디코딩되므로
// "+"를 공백으로 처리하지 않아 파일 못 찾는 버그가 발생했음.
func safeEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func buildURI(filePath, vaultPath string) string {
	rel, err := filepath.Rel(vaultPath, filePath)
	if err != nil {
		rel = filePath
	}
	rel = filepath.ToSlash(rel)

	// adv-uri 프로토콜: Advanced URI v1.44.0+ 이중 디코딩 버그 수정판
	// % 포함 파일명도 단일 인코딩으로 정상 동작
	vaultName := filepath.Base(vaultPath)
	return "obsidian://adv-uri?vault=" +
		safeEscape(vaultName) +
		"&filepath=" +
		safeEscape(rel) +
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
// Self-heal: .md 연결 레지스트리 drift 진단/복구
// ---------------------------------------------------------------------------

type cliOptions struct {
	debug    bool
	doctor   bool
	repair   bool
	filePath string
}

type registryState struct {
	Command          string `json:"command,omitempty"`
	DefaultIcon      string `json:"defaultIcon,omitempty"`
	FriendlyAppName  string `json:"friendlyAppName,omitempty"`
	UserChoiceProgID string `json:"userChoiceProgId,omitempty"`
}

type selfHealReport struct {
	OK          bool            `json:"ok"`
	Mode        string          `json:"mode"`
	HandlerExe  string          `json:"handlerExe"`
	ObsidianExe string          `json:"obsidianExe,omitempty"`
	Current     registryState   `json:"current"`
	Expected    registryState   `json:"expected"`
	Checks      []selfHealCheck `json:"checks"`
}

type selfHealCheck struct {
	Name      string `json:"name"`
	Subkey    string `json:"subkey"`
	ValueName string `json:"valueName"`
	Current   string `json:"current,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Needed    bool   `json:"needed"`
	Repaired  bool   `json:"repaired"`
	Error     string `json:"error,omitempty"`
}

func runSafeSelfHeal(cfg handlerConfig) {
	exe, err := os.Executable()
	if err != nil {
		writeLog("self-heal skip: executable path unavailable")
		return
	}
	if !isProductionHandlerExe(exe) {
		writeLog("self-heal skip: not production handler exe " + exe)
		return
	}

	report := diagnoseSelfHeal(cfg, true)
	for _, check := range report.Checks {
		if check.Needed && check.Error != "" {
			writeLog(fmt.Sprintf("self-heal failed name=%s error=%s", check.Name, check.Error))
		} else if check.Repaired {
			writeLog("self-heal repaired " + check.Name)
		}
	}
}

func diagnoseSelfHeal(cfg handlerConfig, repair bool) selfHealReport {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	exe = filepath.Clean(exe)
	handlerExe := expectedHandlerExePath(exe)
	obsidianExe := findObsidianExePath(cfg)
	current := readCurrentRegistryState()
	expected := expectedRegistryState(handlerExe, obsidianExe)
	checks := buildSelfHealChecks(current, expected)

	changed := false
	if repair {
		for i := range checks {
			if !checks[i].Needed {
				continue
			}
			if err := writeRegistryString(checks[i].Subkey, checks[i].ValueName, checks[i].Expected); err != nil {
				checks[i].Error = err.Error()
				continue
			}
			checks[i].Repaired = true
			changed = true
		}
		if changed {
			notifyAssociationChanged()
			current = readCurrentRegistryState()
			refreshSelfHealChecks(checks, current)
		}
	}

	ok := true
	for _, c := range checks {
		if c.Needed || c.Error != "" {
			ok = false
			break
		}
	}
	if current.UserChoiceProgID != "" && current.UserChoiceProgID != expectedProgID {
		ok = false
	}

	mode := "doctor"
	if repair {
		mode = "repair"
	}
	return selfHealReport{
		OK:          ok,
		Mode:        mode,
		HandlerExe:  handlerExe,
		ObsidianExe: obsidianExe,
		Current:     current,
		Expected:    expected,
		Checks:      checks,
	}
}

func readCurrentRegistryState() registryState {
	return registryState{
		Command:          readRegistryStringOrEmpty(regCommandPath, regDefaultValue),
		DefaultIcon:      readRegistryStringOrEmpty(regDefaultIcon, regDefaultValue),
		FriendlyAppName:  readRegistryStringOrEmpty(regApplication, regFriendlyName),
		UserChoiceProgID: readRegistryStringOrEmpty(regUserChoice, "ProgId"),
	}
}

func expectedRegistryState(handlerExe, obsidianExe string) registryState {
	state := registryState{
		Command:          quoteWindowsPath(handlerExe) + ` "%1"`,
		FriendlyAppName:  friendlyAppName,
		UserChoiceProgID: expectedProgID,
	}
	if obsidianExe != "" {
		state.DefaultIcon = quoteWindowsPath(obsidianExe) + ",0"
	}
	return state
}

func buildSelfHealChecks(current, expected registryState) []selfHealCheck {
	checks := []selfHealCheck{
		newSelfHealCheck("handler-command", regCommandPath, regDefaultValue, current.Command, expected.Command),
		newSelfHealCheck("friendly-app-name", regApplication, regFriendlyName, current.FriendlyAppName, expected.FriendlyAppName),
	}
	if expected.DefaultIcon != "" {
		checks = append(checks, newSelfHealCheck("default-icon", regDefaultIcon, regDefaultValue, current.DefaultIcon, expected.DefaultIcon))
	}
	return checks
}

func newSelfHealCheck(name, subkey, valueName, current, expected string) selfHealCheck {
	return selfHealCheck{
		Name:      name,
		Subkey:    subkey,
		ValueName: valueName,
		Current:   current,
		Expected:  expected,
		Needed:    current != expected,
	}
}

func refreshSelfHealChecks(checks []selfHealCheck, current registryState) {
	for i := range checks {
		switch checks[i].Name {
		case "handler-command":
			checks[i].Current = current.Command
		case "friendly-app-name":
			checks[i].Current = current.FriendlyAppName
		case "default-icon":
			checks[i].Current = current.DefaultIcon
		}
		checks[i].Needed = checks[i].Current != checks[i].Expected
	}
}

func findObsidianExePath(cfg handlerConfig) string {
	if cfg.ObsidianExePath != "" && fileExists(cfg.ObsidianExePath) {
		return filepath.Clean(cfg.ObsidianExePath)
	}

	candidates := []string{
		`C:\Program Files\Obsidian\Obsidian.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Programs\Obsidian\Obsidian.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Obsidian\Obsidian.exe`),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Obsidian\Obsidian.exe`),
	}
	for _, c := range candidates {
		if fileExists(c) {
			return filepath.Clean(c)
		}
	}
	return ""
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func isProductionHandlerExe(exe string) bool {
	return strings.EqualFold(filepath.Base(exe), productionExe)
}

func expectedHandlerExePath(runningExe string) string {
	if runningExe == "" {
		return ""
	}
	productionPath := filepath.Join(filepath.Dir(runningExe), productionExe)
	if fileExists(productionPath) {
		return productionPath
	}
	return runningExe
}

func quoteWindowsPath(path string) string {
	return `"` + path + `"`
}

func readRegistryStringOrEmpty(subkey, valueName string) string {
	value, err := readRegistryString(subkey, valueName)
	if err != nil {
		return ""
	}
	return value
}

func readRegistryString(subkey, valueName string) (string, error) {
	subkeyPtr, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return "", err
	}
	valuePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return "", err
	}

	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, subkeyPtr, 0, syscall.KEY_QUERY_VALUE, &key); err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(key) //nolint:errcheck

	var valueType uint32
	var byteLen uint32
	if err := syscall.RegQueryValueEx(key, valuePtr, nil, &valueType, nil, &byteLen); err != nil {
		return "", err
	}
	if valueType != syscall.REG_SZ && valueType != syscall.REG_EXPAND_SZ {
		return "", fmt.Errorf("unsupported registry value type %d", valueType)
	}
	if byteLen == 0 {
		return "", nil
	}

	buf := make([]uint16, (byteLen+1)/2)
	if err := syscall.RegQueryValueEx(key, valuePtr, nil, &valueType, (*byte)(unsafe.Pointer(&buf[0])), &byteLen); err != nil {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}

func writeRegistryString(subkey, valueName, value string) error {
	subkeyPtr, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return err
	}
	valuePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return err
	}
	valueUTF16, err := syscall.UTF16FromString(value)
	if err != nil {
		return err
	}

	var key syscall.Handle
	ret, _, _ := procRegCreateKeyExW.Call(
		uintptr(syscall.HKEY_CURRENT_USER),
		uintptr(unsafe.Pointer(subkeyPtr)),
		0,
		0,
		0,
		uintptr(syscall.KEY_SET_VALUE),
		0,
		uintptr(unsafe.Pointer(&key)),
		0,
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	defer syscall.RegCloseKey(key) //nolint:errcheck

	byteLen := uintptr(len(valueUTF16) * 2)
	ret, _, _ = procRegSetValueExW.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valuePtr)),
		0,
		uintptr(syscall.REG_SZ),
		uintptr(unsafe.Pointer(&valueUTF16[0])),
		byteLen,
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func notifyAssociationChanged() {
	procSHChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0)
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
	escapedPath := escapeShellArgument(filePath)
	filePtr, _ := syscall.UTF16PtrFromString(escapedPath)

	writeLog(fmt.Sprintf("LAUNCH fallback-args app=%s args=%s", app, escapedPath))

	procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(appPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		0,
		1,
	)
}

func escapeShellArgument(arg string) string {
	return syscall.EscapeArg(arg)
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

func parseArgs(args []string) cliOptions {
	var opts cliOptions
	for _, a := range args {
		switch a {
		case "--debug":
			opts.debug = true
		case "--doctor":
			opts.doctor = true
		case "--repair":
			opts.repair = true
		default:
			if !strings.HasPrefix(a, "--") {
				opts.filePath = a
			}
		}
	}
	return opts
}
