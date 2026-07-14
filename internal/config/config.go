package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BaseURL      string
	Timeout      time.Duration
	AuthMode     string
	Username     string
	Password     string
	DeveloperKey string
	CustomerKey  string
	ToolMode     string
	EnableWrites bool
	EnableDanger bool
	AuditLogs    bool
}

func LoadEnv(path string) (map[string]string, error) {
	env := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return env, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := env[key]; !exists {
			env[key] = value
		}
	}

	return env, scanner.Err()
}

func FromEnv(env map[string]string) (Config, error) {
	timeoutMS := 30000
	if raw := strings.TrimSpace(env["KT_TIMEOUT_MS"]); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("KT_TIMEOUT_MS must be an integer: %w", err)
		}
		timeoutMS = parsed
	}

	toolMode := strings.ToLower(strings.TrimSpace(defaultString(env["KT_TOOL_MODE"], "readonly")))
	if toolMode != "readonly" && toolMode != "full" {
		return Config{}, fmt.Errorf("KT_TOOL_MODE must be either 'readonly' or 'full'")
	}

	authMode := strings.ToLower(strings.TrimSpace(defaultString(env["KT_AUTH_MODE"], "partner")))
	if authMode != "partner" && authMode != "session" {
		return Config{}, fmt.Errorf("KT_AUTH_MODE must be either 'partner' or 'session'")
	}

	cfg := Config{
		BaseURL:      strings.TrimRight(defaultString(env["KT_BASE_URL"], "https://api.klicktipp.com"), "/"),
		Timeout:      time.Duration(timeoutMS) * time.Millisecond,
		AuthMode:     authMode,
		Username:     strings.TrimSpace(env["KT_USERNAME"]),
		Password:     env["KT_PASSWORD"],
		DeveloperKey: env["KT_DEVELOPER_KEY"],
		CustomerKey:  env["KT_CUSTOMER_KEY"],
		ToolMode:     toolMode,
		EnableWrites: parseBool(env["KT_ENABLE_WRITES"], false),
		EnableDanger: parseBool(env["KT_ENABLE_DESTRUCTIVE"], false),
		AuditLogs:    parseBool(env["KT_AUDIT_LOGS"], true),
	}

	return cfg, nil
}

func (c Config) WritesAllowed() bool {
	return c.ToolMode == "full" && c.EnableWrites
}

func (c Config) DestructiveAllowed() bool {
	return c.ToolMode == "full" && c.EnableWrites && c.EnableDanger
}

func parseBool(value string, defaultValue bool) bool {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
