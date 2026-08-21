package main

import (
	"path/filepath"
	"testing"
)

func TestEnvPathUsesExecutableDirectory(t *testing.T) {
	executableDir := t.TempDir()
	executable := filepath.Join(executableDir, "klicktipp-mcp")

	otherDir := t.TempDir()
	t.Chdir(otherDir)

	got := envPath(executable)
	want := filepath.Join(executableDir, ".env")
	if got != want {
		t.Fatalf("envPath(%q) = %q, want %q", executable, got, want)
	}
}

func TestDefaultVersionIsSet(t *testing.T) {
	if version == "" {
		t.Fatal("expected version to be non-empty")
	}
}
