package auth

import (
	"path/filepath"
	"testing"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
)

func TestPartnerHeadersAllowsCompleteEnvironmentOnlyConfiguration(t *testing.T) {
	cfg := config.Config{
		Username:     "test-user",
		DeveloperKey: "00",
		CustomerKey:  "test-customer",
		EnvFilePath:  filepath.Join(t.TempDir(), ".env"),
		EnvFileFound: false,
	}

	headers, err := PartnerHeaders(cfg)
	if err != nil {
		t.Fatalf("PartnerHeaders returned an error for complete environment-only configuration: %v", err)
	}
	if headers["X-Un"] != cfg.Username {
		t.Fatalf("PartnerHeaders set X-Un to %q, want %q", headers["X-Un"], cfg.Username)
	}
	if headers["X-Ci"] == "" {
		t.Fatal("PartnerHeaders returned an empty X-Ci header")
	}
}

func TestPartnerHeadersReportsMissingEnvFile(t *testing.T) {
	cfg := config.Config{
		EnvFilePath:  filepath.Join(t.TempDir(), ".env"),
		EnvFileFound: false,
	}

	_, err := PartnerHeaders(cfg)
	if err == nil {
		t.Fatal("PartnerHeaders succeeded with incomplete configuration")
	}
	if got := err.Error(); got == "KT_USERNAME is required for partner auth" {
		t.Fatalf("PartnerHeaders returned the ambiguous legacy error: %q", got)
	}
}
