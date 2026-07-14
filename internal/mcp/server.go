package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Tool interface {
	Definition() ToolDefinition
	Call(arguments map[string]any) any
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type Server struct {
	name    string
	version string
	tools   map[string]Tool
	ordered []ToolDefinition
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func NewServer(name string, version string, tools []Tool) *Server {
	index := make(map[string]Tool, len(tools))
	ordered := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		def := tool.Definition()
		index[def.Name] = tool
		ordered = append(ordered, def)
	}

	return &Server{name: name, version: version, tools: index, ordered: ordered}
}

func (s *Server) Serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		payload, err := readMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var req request
		if err := json.Unmarshal(payload, &req); err != nil {
			if err := writeMessage(out, response{JSONRPC: "2.0", Error: rpcError(-32700, "parse error")}); err != nil {
				return err
			}
			continue
		}

		if req.Method == "notifications/initialized" || req.Method == "notifications/cancelled" {
			continue
		}

		result, rpcErr := s.handle(req)
		if req.ID == nil {
			continue
		}

		res := response{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			res.Error = rpcErr
		} else {
			res.Result = result
		}

		if err := writeMessage(out, res); err != nil {
			return err
		}
	}
}

func (s *Server) handle(req request) (any, map[string]any) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]any{
				"name":    s.name,
				"version": s.version,
			},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": s.ordered}, nil
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, rpcError(-32602, "invalid params")
		}
		tool, ok := s.tools[params.Name]
		if !ok {
			return nil, rpcError(-32601, fmt.Sprintf("unknown tool: %s", params.Name))
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		return tool.Call(params.Arguments), nil
	default:
		return nil, rpcError(-32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func rpcError(code int, message string) map[string]any {
	return map[string]any{
		"code":    code,
		"message": message,
	}
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, err
			}
			contentLength = parsed
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing content length")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeMessage(out io.Writer, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "Content-Length: %d\r\n\r\n", len(encoded))
	buffer.Write(encoded)

	_, err = out.Write(buffer.Bytes())
	return err
}
