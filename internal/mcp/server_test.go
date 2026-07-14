package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

type testTool struct {
	definition ToolDefinition
}

func (t testTool) Definition() ToolDefinition { return t.definition }
func (t testTool) Call(arguments map[string]any) any {
	return map[string]any{"ok": true}
}

func TestInitializeAndToolsList(t *testing.T) {
	server := NewServer("klicktipp-mcp", "0.1.0-test", []Tool{testTool{definition: ToolDefinition{
		Name:        "list_tags",
		Description: "List tags",
		InputSchema: map[string]any{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object", "properties": map[string]any{}},
	}}})

	var input bytes.Buffer
	writeRequest(&input, 1, "initialize", map[string]any{"protocolVersion": "2025-06-18"})
	writeRequest(&input, 2, "tools/list", map[string]any{})

	var output bytes.Buffer
	if err := server.Serve(&input, &output); err != nil {
		t.Fatalf("serve failed: %v", err)
	}

	responses := parseResponses(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	initialize := responses[0]["result"].(map[string]any)
	if got := initialize["protocolVersion"]; got != "2025-06-18" {
		t.Fatalf("unexpected protocolVersion: %v", got)
	}
	capabilities := initialize["capabilities"].(map[string]any)
	tools := capabilities["tools"].(map[string]any)
	if got := tools["listChanged"]; got != true {
		t.Fatalf("expected listChanged true, got %v", got)
	}

	list := responses[1]["result"].(map[string]any)
	toolDefs := list["tools"].([]any)
	if len(toolDefs) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolDefs))
	}
	toolDef := toolDefs[0].(map[string]any)
	if got := toolDef["name"]; got != "list_tags" {
		t.Fatalf("unexpected tool name: %v", got)
	}
	inputSchema := toolDef["inputSchema"].(map[string]any)
	if got := inputSchema["$schema"]; got != "http://json-schema.org/draft-07/schema#" {
		t.Fatalf("unexpected input schema draft: %v", got)
	}
}

func writeRequest(buffer *bytes.Buffer, id int, method string, params map[string]any) {
	payload, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	fmt.Fprintf(buffer, "Content-Length: %d\r\n\r\n", len(payload))
	buffer.Write(payload)
}

func parseResponses(t *testing.T, raw []byte) []map[string]any {
	t.Helper()

	reader := bufio.NewReader(bytes.NewReader(raw))
	responses := []map[string]any{}
	for {
		payload, err := readMessage(reader)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			if len(payload) == 0 {
				break
			}
			t.Fatalf("failed to read response: %v", err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(payload, &parsed); err != nil {
			t.Fatalf("failed to parse response payload: %v", err)
		}
		responses = append(responses, parsed)
	}
	return responses
}
