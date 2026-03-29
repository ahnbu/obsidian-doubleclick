package main

import "testing"

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
