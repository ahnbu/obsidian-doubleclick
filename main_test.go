package main

import (
	"os"
	"testing"
)

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
		`D:\tools\obsidian-handler.exe`,
		`C:\Program Files\Obsidian\Obsidian.exe`,
	)

	if got.Command != `"D:\tools\obsidian-handler.exe" "%1"` {
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
	expected := expectedRegistryState(`D:\tools\obsidian-handler.exe`, "")

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
	productionPath := dir + `\obsidian-handler.exe`
	if err := os.WriteFile(productionPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	got := expectedHandlerExePath(dir + `\obsidian-handler-debug.exe`)

	if got != productionPath {
		t.Fatalf("expected %q, got %q", productionPath, got)
	}
}
