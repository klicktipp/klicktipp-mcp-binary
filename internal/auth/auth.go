package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
)

func PartnerHeaders(cfg config.Config) (map[string]string, error) {
	username := strings.TrimSpace(cfg.Username)
	missing := make([]string, 0, 3)
	if username == "" {
		missing = append(missing, "KT_USERNAME")
	}
	if strings.TrimSpace(cfg.DeveloperKey) == "" {
		missing = append(missing, "KT_DEVELOPER_KEY")
	}
	if strings.TrimSpace(cfg.CustomerKey) == "" {
		missing = append(missing, "KT_CUSTOMER_KEY")
	}
	if len(missing) > 0 {
		return nil, cfg.MissingSettingsError("partner auth", missing...)
	}

	ci, err := BuildPartnerCipher(cfg.DeveloperKey, cfg.CustomerKey)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"X-Un": username,
		"X-Ci": ci,
	}, nil
}

func BuildPartnerCipher(developerKey string, customerKey string) (string, error) {
	normalizedDeveloperKey := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(developerKey), " ", ""))
	if normalizedDeveloperKey == "" {
		return "", fmt.Errorf("KT_DEVELOPER_KEY is required for partner auth")
	}
	if len(normalizedDeveloperKey)%2 != 0 {
		return "", fmt.Errorf("KT_DEVELOPER_KEY must be an even-length hexadecimal string")
	}

	keyBytes, err := hex.DecodeString(normalizedDeveloperKey)
	if err != nil {
		return "", fmt.Errorf("KT_DEVELOPER_KEY must be a hexadecimal string: %w", err)
	}

	normalizedCustomerKey := strings.TrimSpace(customerKey)
	if normalizedCustomerKey == "" {
		return "", fmt.Errorf("KT_CUSTOMER_KEY is required for partner auth")
	}

	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(normalizedCustomerKey))
	sum := mac.Sum(nil)

	payload := append(sum, []byte(normalizedCustomerKey)...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

func BuildGrantAccessURL(accountID string, redirectURL string) (string, error) {
	normalizedAccountID := strings.TrimSpace(accountID)
	normalizedRedirectURL := strings.TrimSpace(redirectURL)
	if normalizedAccountID == "" || normalizedRedirectURL == "" {
		return "", fmt.Errorf("both accountId and redirectUrl are required to build the grant URL")
	}

	return fmt.Sprintf(
		"https://app.klicktipp.com/grantapiaccess/%s?url=%s",
		url.PathEscape(normalizedAccountID),
		url.QueryEscape(normalizedRedirectURL),
	), nil
}
