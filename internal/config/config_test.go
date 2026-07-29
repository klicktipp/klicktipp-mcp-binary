package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEnvWithSourceReportsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")

	_, found, err := LoadEnvWithSource(path)
	if err != nil {
		t.Fatalf("LoadEnvWithSource(%q) returned an error: %v", path, err)
	}
	if found {
		t.Fatalf("LoadEnvWithSource(%q) reported a file that does not exist", path)
	}
}

func TestMissingSettingsErrorReportsMissingEnvFile(t *testing.T) {
	cfg := Config{
		EnvFilePath:  filepath.Join("opt", "klicktipp-mcp", ".env"),
		EnvFileFound: false,
	}

	err := cfg.MissingSettingsError("partner auth", "KT_USERNAME")
	message := err.Error()
	if !strings.Contains(message, "was not found") {
		t.Fatalf("expected missing-file context, got %q", message)
	}
	if !strings.Contains(message, "KT_USERNAME is required for partner auth") {
		t.Fatalf("expected missing-setting context, got %q", message)
	}
}

func TestMissingSettingsErrorReportsIncompleteEnvFile(t *testing.T) {
	cfg := Config{
		EnvFilePath:  filepath.Join("opt", "klicktipp-mcp", ".env"),
		EnvFileFound: true,
	}

	err := cfg.MissingSettingsError("session auth", "KT_PASSWORD")
	message := err.Error()
	if !strings.Contains(message, "was loaded, but") {
		t.Fatalf("expected loaded-file context, got %q", message)
	}
	if !strings.Contains(message, "KT_PASSWORD is required for session auth") {
		t.Fatalf("expected missing-setting context, got %q", message)
	}
}
