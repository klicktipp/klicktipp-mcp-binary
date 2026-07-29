package api

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
)

func TestLoginReportsIncompleteLoadedEnvFile(t *testing.T) {
	cfg := config.Config{
		AuthMode:     "session",
		Username:     "test-user",
		EnvFilePath:  filepath.Join(t.TempDir(), ".env"),
		EnvFileFound: true,
	}
	client := NewClient(cfg)

	_, err := client.login(context.Background())
	if err == nil {
		t.Fatal("login succeeded without KT_PASSWORD")
	}
	if !strings.Contains(err.Error(), "was loaded, but KT_PASSWORD is required for session auth") {
		t.Fatalf("login returned an unexpected error: %q", err)
	}
}
