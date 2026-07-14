package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
)

func Log(cfg config.Config, entry map[string]any) {
	if !cfg.AuditLogs {
		return
	}

	payload := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range entry {
		payload[key] = value
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "{\"type\":\"audit_error\",\"message\":%q}\n", err.Error())
		return
	}

	fmt.Fprintln(os.Stderr, string(encoded))
}
