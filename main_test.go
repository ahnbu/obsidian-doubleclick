package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- R2: URI 모드 판정 & URI 생성 -------------------------------------------

// writeCommunityPlugins: 임시 볼트에 .obsidian/community-plugins.json을 만든다.
// content가 빈 문자열이면 파일을 만들지 않는다(플러그인 설정 자체가 없는 볼트).
func writeCommunityPlugins(t *testing.T, content string) string {
	t.Helper()
	vault := t.TempDir()
	if content == "" {
		return vault
	}
	dir := filepath.Join(vault, ".obsidian")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "community-plugins.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return vault
}

func TestDetectAdvancedURI(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"활성 목록에 있음", `["dataview","obsidian-advanced-uri","show-hidden-files"]`, true},
		{"활성 목록에 없음", `["dataview","show-hidden-files"]`, false},
		{"파일 없음", "", false},
		{"JSON 깨짐", `["dataview",`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault := writeCommunityPlugins(t, tc.content)
			if got := detectAdvancedURI(vault); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestResolveURIModeConfigOverridesDetection(t *testing.T) {
	// 플러그인이 활성이지만 config가 official을 강제하는 경우
	vault := writeCommunityPlugins(t, `["obsidian-advanced-uri"]`)

	if got := resolveURIMode(handlerConfig{URIMode: uriModeOfficial}, vault); got != uriModeOfficial {
		t.Fatalf("config override ignored: got %q", got)
	}
	if got := resolveURIMode(handlerConfig{URIMode: uriModeAdvanced}, vault); got != uriModeAdvanced {
		t.Fatalf("config override ignored: got %q", got)
	}
}

func TestResolveURIModeAutoFollowsDetection(t *testing.T) {
	withPlugin := writeCommunityPlugins(t, `["obsidian-advanced-uri"]`)
	withoutPlugin := writeCommunityPlugins(t, `["dataview"]`)

	// 빈 값과 "auto" 모두 자동 판정으로 떨어져야 한다.
	for _, mode := range []string{"", "auto", "알수없는값"} {
		if got := resolveURIMode(handlerConfig{URIMode: mode}, withPlugin); got != uriModeAdvanced {
			t.Fatalf("uriMode=%q: expected %q, got %q", mode, uriModeAdvanced, got)
		}
		if got := resolveURIMode(handlerConfig{URIMode: mode}, withoutPlugin); got != uriModeOfficial {
			t.Fatalf("uriMode=%q: expected %q, got %q", mode, uriModeOfficial, got)
		}
	}
}

func TestBuildURIAdvancedMode(t *testing.T) {
	got := buildURI(`D:\vault\sub\note.md`, `D:\vault`, uriModeAdvanced)
	want := "obsidian://adv-uri?vault=vault&filepath=sub%2Fnote.md&openmode=true"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildURIOfficialMode(t *testing.T) {
	got := buildURI(`D:\vault\sub\note.md`, `D:\vault`, uriModeOfficial)
	want := "obsidian://open?vault=vault&file=sub%2Fnote.md&paneType=tab"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// 공백은 "+"가 아니라 "%20"으로 인코딩되어야 한다(2026-03-17 회귀 방지).
func TestBuildURIEncodesSpacesAndHangulInBothModes(t *testing.T) {
	adv := buildURI(`D:\vibe coding\한글 노트.md`, `D:\vibe coding`, uriModeAdvanced)
	official := buildURI(`D:\vibe coding\한글 노트.md`, `D:\vibe coding`, uriModeOfficial)

	for _, uri := range []string{adv, official} {
		if strings.Contains(uri, "+") {
			t.Fatalf("space must be %%20, not +: %q", uri)
		}
		if !strings.Contains(uri, "vibe%20coding") {
			t.Fatalf("vault name not encoded: %q", uri)
		}
		if !strings.Contains(uri, "%ED%95%9C%EA%B8%80%20%EB%85%B8%ED%8A%B8.md") {
			t.Fatalf("hangul filename not encoded: %q", uri)
		}
	}
}

// % 포함 파일명이 단일 인코딩되는지 (2026-03-18 회귀 방지)
func TestBuildURIEncodesPercentInFilename(t *testing.T) {
	got := buildURI(`D:\vault\50% 완료.md`, `D:\vault`, uriModeAdvanced)

	if !strings.Contains(got, "50%25%20%EC%99%84%EB%A3%8C.md") {
		t.Fatalf("percent not single-encoded: %q", got)
	}
}

// --- R1: 볼트별 창 특정 ------------------------------------------------------

func TestPickObsidianWindowPrefersMatchingVault(t *testing.T) {
	windows := []windowInfo{
		{hwnd: 0x11, title: "노트A - vibe-coding - Obsidian 1.13.7"},
		{hwnd: 0x22, title: "노트B - cowork - Obsidian 1.13.7"},
		{hwnd: 0x33, title: "노트C - .agents - Obsidian 1.13.7"},
	}

	got, match := pickObsidianWindow(windows, "cowork")

	if match != "exact" {
		t.Fatalf("expected exact match, got %q", match)
	}
	if got.hwnd != 0x22 {
		t.Fatalf("expected hwnd 0x22, got 0x%X", got.hwnd)
	}
}

// 볼트 이름이 다른 볼트 이름의 접두사여도 오매칭되지 않아야 한다.
func TestPickObsidianWindowDoesNotMatchVaultNamePrefix(t *testing.T) {
	windows := []windowInfo{
		{hwnd: 0x11, title: "노트A - vibe-coding - Obsidian 1.13.7"},
	}

	_, match := pickObsidianWindow(windows, "vibe")

	if match == "exact" {
		t.Fatal(`"vibe" must not match " - vibe-coding - Obsidian"`)
	}
}

func TestPickObsidianWindowFallsBackWhenVaultTitleAbsent(t *testing.T) {
	windows := []windowInfo{
		{hwnd: 0x44, title: "Obsidian"},
	}

	got, match := pickObsidianWindow(windows, "cowork")

	if match != "fallback" {
		t.Fatalf("expected fallback, got %q", match)
	}
	if got.hwnd != 0x44 {
		t.Fatalf("expected hwnd 0x44, got 0x%X", got.hwnd)
	}
}

func TestPickObsidianWindowReturnsNoneWhenEmpty(t *testing.T) {
	got, match := pickObsidianWindow(nil, "cowork")

	if match != "none" {
		t.Fatalf("expected none, got %q", match)
	}
	if got.hwnd != 0 {
		t.Fatalf("expected zero hwnd, got 0x%X", got.hwnd)
	}
}

// 볼트를 갓 열어 노트가 없는 창은 제목이 "<볼트명> - Obsidian <버전>"이다.
// 2026-08-27 실측에서 이 형태가 exact 매칭에 실패해 폴백으로 떨어졌다.
func TestPickObsidianWindowMatchesVaultWithNoNoteOpen(t *testing.T) {
	windows := []windowInfo{
		{hwnd: 0x11, title: "노트A - cowork - Obsidian 1.13.7"},
		{hwnd: 0x22, title: "vibe-coding - Obsidian 1.13.7"},
	}

	got, match := pickObsidianWindow(windows, "vibe-coding")

	if match != "exact" {
		t.Fatalf("expected exact match, got %q", match)
	}
	if got.hwnd != 0x22 {
		t.Fatalf("expected hwnd 0x22, got 0x%X", got.hwnd)
	}
}

// 노트 제목이 다른 볼트 이름과 같아도 오매칭되지 않아야 한다.
func TestMatchesVaultTitleRejectsNoteTitledLikeVault(t *testing.T) {
	// "vibe-coding"이라는 제목의 노트가 cowork 볼트에서 열린 창
	title := "vibe-coding - cowork - Obsidian 1.13.7"

	if matchesVaultTitle(title, "vibe-coding") {
		t.Fatalf("note titled like a vault must not match: %q", title)
	}
	if !matchesVaultTitle(title, "cowork") {
		t.Fatalf("actual vault must match: %q", title)
	}
}

func TestMatchesVaultTitleEmptyVaultName(t *testing.T) {
	if matchesVaultTitle("새 탭 - cowork - Obsidian 1.13.7", "") {
		t.Fatal("empty vault name must not match anything")
	}
}

// 콜드 스타트에서는 대기 예산이 눈에 띄게 길어야 한다.
// 고정 예산(약 2.5초)이 만료돼 창이 뜨기 전에 포기하던 것이 원래 증상이다.
func TestActivateBudgetIsLongerOnColdStart(t *testing.T) {
	warm := activateBudget(false)
	cold := activateBudget(true)

	if cold <= warm {
		t.Fatalf("cold budget (%v) must exceed warm budget (%v)", cold, warm)
	}
	if cold < 20*time.Second {
		t.Fatalf("cold budget %v is too short for an Electron cold start + vault indexing", cold)
	}
	if warm > 5*time.Second {
		t.Fatalf("warm budget %v is too long — Obsidian is already running", warm)
	}
}

func TestVaultTitleMarkerFormat(t *testing.T) {
	// 실측 창 제목: "새 탭 - cowork - Obsidian 1.13.7"
	// 제목 끝이 "Obsidian"이 아니라 버전이므로 부분 문자열이어야 한다.
	title := "새 탭 - cowork - Obsidian 1.13.7"

	if !strings.Contains(title, vaultTitleMarker("cowork")) {
		t.Fatalf("marker %q not found in %q", vaultTitleMarker("cowork"), title)
	}
}

func TestEscapeShellArgumentQuotesPathWithSpaces(t *testing.T) {
	got := escapeShellArgument(`D:\tmp\file with spaces.md`)
	want := `"D:\tmp\file with spaces.md"`

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestEscapeShellArgumentKeepsPathWithoutSpaces(t *testing.T) {
	got := escapeShellArgument(`D:\tmp\plain.md`)
	want := `D:\tmp\plain.md`

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestParseArgsRecognizesDoctorAndRepair(t *testing.T) {
	opts := parseArgs([]string{"--doctor", "--repair"})

	if !opts.doctor {
		t.Fatal("expected doctor flag")
	}
	if !opts.repair {
		t.Fatal("expected repair flag")
	}
	if opts.filePath != "" {
		t.Fatalf("expected no file path, got %q", opts.filePath)
	}
}

func TestExpectedRegistryState(t *testing.T) {
	got := expectedRegistryState(
		`D:\tools\obsidian-doubleclick.exe`,
		`C:\Program Files\Obsidian\Obsidian.exe`,
	)

	if got.Command != `"D:\tools\obsidian-doubleclick.exe" "%1"` {
		t.Fatalf("unexpected command: %q", got.Command)
	}
	if got.DefaultIcon != `"C:\Program Files\Obsidian\Obsidian.exe",0` {
		t.Fatalf("unexpected default icon: %q", got.DefaultIcon)
	}
	if got.FriendlyAppName != "Obsidian" {
		t.Fatalf("unexpected friendly name: %q", got.FriendlyAppName)
	}
}

func TestBuildSelfHealChecksSkipsIconWhenObsidianPathUnknown(t *testing.T) {
	current := registryState{}
	expected := expectedRegistryState(`D:\tools\obsidian-doubleclick.exe`, "")

	checks := buildSelfHealChecks(current, expected)

	for _, check := range checks {
		if check.Name == "default-icon" {
			t.Fatal("default icon should be skipped when Obsidian.exe path is unknown")
		}
	}
}

func TestExpectedHandlerExePathPrefersProductionSibling(t *testing.T) {
	t.Setenv("TEMP", t.TempDir())
	dir := t.TempDir()
	productionPath := dir + `\obsidian-doubleclick.exe`
	if err := os.WriteFile(productionPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	got := expectedHandlerExePath(dir + `\obsidian-doubleclick-debug.exe`)

	if got != productionPath {
		t.Fatalf("expected %q, got %q", productionPath, got)
	}
}
