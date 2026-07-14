package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/api"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/audit"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
	kerrors "github.com/klicktipp/klicktipp-binary-mcp/internal/errors"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const jsonSchemaDraft07 = "https://json-schema.org/draft/2020-12/schema"

type toolLevel string

const (
	levelRead        toolLevel = "read"
	levelWrite       toolLevel = "write"
	levelDestructive toolLevel = "destructive"
)

type toolSpec struct {
	name        string
	description string
	schema      map[string]any
	level       toolLevel
	run         func(ctx context.Context, args map[string]any) (any, error)
}

func Register(server *mcp.Server, client *api.Client, cfg config.Config, env map[string]string) {
	_ = env
	for _, tool := range build(client, cfg) {
		registerTool(server, cfg, tool)
	}
}

func build(client *api.Client, cfg config.Config) []toolSpec {
	tools := []toolSpec{
		readTool("list_tags", "List KlickTipp tags as id/name pairs.", objectSchema(), func(ctx context.Context, args map[string]any) (any, error) {
			tags, err := client.ListTags(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(tags))
			for id, name := range tags {
				items = append(items, map[string]any{"id": id, "name": name})
			}
			sort.Slice(items, func(i int, j int) bool {
				return fmt.Sprint(items[i]["id"]) < fmt.Sprint(items[j]["id"])
			})
			return map[string]any{"count": len(items), "items": items}, nil
		}),
		readTool("get_tag", "Get one KlickTipp tag by tag ID.", objectSchema(propStringOrIntegerID("tagid")), func(ctx context.Context, args map[string]any) (any, error) {
			return client.GetTag(ctx, args["tagid"])
		}),
		readTool("list_fields", "List KlickTipp data fields as id/name pairs.", objectSchema(), func(ctx context.Context, args map[string]any) (any, error) {
			fields, err := client.ListFields(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(fields))
			for id, name := range fields {
				items = append(items, map[string]any{"id": id, "name": name})
			}
			sort.Slice(items, func(i int, j int) bool {
				return fmt.Sprint(items[i]["id"]) < fmt.Sprint(items[j]["id"])
			})
			return map[string]any{"count": len(items), "items": items}, nil
		}),
		readTool("get_field", "Get one KlickTipp data field by field ID. Accepts IDs returned by list_fields, for example fieldFirstName or field203826, and also raw API IDs such as FirstName or 203826.", objectSchema(propString("fieldid")), func(ctx context.Context, args map[string]any) (any, error) {
			fieldID, err := requireString(args, "fieldid")
			if err != nil {
				return nil, err
			}
			return client.GetField(ctx, fieldID)
		}),
		readTool("list_opt_in_processes", "List KlickTipp opt-in processes as id/name pairs.", objectSchema(), func(ctx context.Context, args map[string]any) (any, error) {
			lists, err := client.ListOptInProcesses(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(lists))
			for id, name := range lists {
				items = append(items, map[string]any{"id": id, "name": name})
			}
			sort.Slice(items, func(i int, j int) bool {
				return fmt.Sprint(items[i]["id"]) < fmt.Sprint(items[j]["id"])
			})
			return map[string]any{"count": len(items), "items": items}, nil
		}),
		readTool("get_opt_in_process", "Get one KlickTipp opt-in process by list ID.", objectSchema(propStringOrIntegerID("listid")), func(ctx context.Context, args map[string]any) (any, error) {
			return client.GetOptInProcess(ctx, args["listid"])
		}),
		readTool("get_opt_in_process_redirect", "Get the redirect URL for a specific KlickTipp opt-in process and email address.", objectSchema(propStringOrIntegerID("listid"), propEmail("email")), func(ctx context.Context, args map[string]any) (any, error) {
			email, err := requireString(args, "email")
			if err != nil {
				return nil, err
			}
			return client.GetOptInProcessRedirect(ctx, map[string]any{"listid": args["listid"], "email": email})
		}),
		readTool("list_contacts", "List KlickTipp contact IDs with optional status and bounce status filters. Allowed status values: subscribed, pending, unsubscribed. Allowed bounceStatus values: nobounce, softbounce, hardbounce, spambounce.", objectSchema(optionalEnumArray("status", "subscribed", "pending", "unsubscribed"), optionalEnumArray("bounceStatus", "nobounce", "softbounce", "hardbounce", "spambounce")), func(ctx context.Context, args map[string]any) (any, error) {
			return client.ListContacts(ctx, pick(args, "status", "bounceStatus"))
		}),
		readTool("get_contact", "Get one KlickTipp contact by contact ID.", objectSchema(propStringOrIntegerID("subscriberid")), func(ctx context.Context, args map[string]any) (any, error) {
			return client.GetContact(ctx, args["subscriberid"])
		}),
		readTool("search_contact", "Search for a KlickTipp contact ID by email address.", objectSchema(propEmail("email")), func(ctx context.Context, args map[string]any) (any, error) {
			email, err := requireString(args, "email")
			if err != nil {
				return nil, err
			}
			return client.SearchContact(ctx, map[string]any{"email": email})
		}),
		readTool("search_tagged_contacts", "List tagged KlickTipp contacts for one tag, with optional status and bounce status filters. Allowed status values: subscribed, pending, unsubscribed. Allowed bounceStatus values: nobounce, softbounce, hardbounce, spambounce.", objectSchema(propStringOrIntegerID("tagid"), optionalEnumArray("status", "subscribed", "pending", "unsubscribed"), optionalEnumArray("bounceStatus", "nobounce", "softbounce", "hardbounce", "spambounce")), func(ctx context.Context, args map[string]any) (any, error) {
			return client.SearchTaggedContacts(ctx, pick(args, "tagid", "status", "bounceStatus"))
		}),
	}

	if cfg.WritesAllowed() {
		tools = append(tools,
			writeTool("create_tag", "Create a new KlickTipp tag.", objectSchema(propString("name"), optionalPropString("text"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.CreateTag(ctx, pick(args, "name", "text"))
			}),
			writeTool("update_tag", "Update an existing KlickTipp tag by tag ID.", objectSchema(propStringOrIntegerID("tagid"), optionalPropString("name"), optionalPropString("text"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.UpdateTag(ctx, args["tagid"], pick(args, "name", "text"))
			}),
			writeTool("create_or_update_contact", "Create a new KlickTipp contact or update an existing one by email or SMS number.", objectSchema(optionalPropEmail("email"), optionalPropString("smsnumber"), optionalPropIntegerID("listid"), optionalPropStringOrIntegerID("tagid"), optionalObject("fields"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				if isBlank(args["email"]) && isBlank(args["smsnumber"]) {
					return nil, fmt.Errorf("either email or smsnumber is required")
				}
				return client.CreateOrUpdateContact(ctx, pick(args, "email", "smsnumber", "listid", "tagid", "fields"))
			}),
			writeTool("update_contact", "Update an existing KlickTipp contact by contact ID.", objectSchema(propStringOrIntegerID("subscriberid"), optionalPropEmail("newemail"), optionalPropString("newsmsnumber"), optionalObject("fields"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.UpdateContact(ctx, args["subscriberid"], pick(args, "newemail", "newsmsnumber", "fields"))
			}),
			writeTool("tag_contact", "Assign one or more existing KlickTipp tags to a contact by email address.", objectSchema(propEmail("email"), requiredStringOrIntegerArray("tagids"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.TagContact(ctx, pick(args, "email", "tagids"))
			}),
			writeTool("untag_contact", "Remove one existing KlickTipp tag from a contact by email address.", objectSchema(propEmail("email"), propStringOrIntegerID("tagid"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.UntagContact(ctx, pick(args, "email", "tagid"))
			}),
		)
	}

	if cfg.DestructiveAllowed() {
		tools = append(tools,
			destructiveTool("delete_tag", "Delete an existing KlickTipp tag by tag ID.", objectSchema(propStringOrIntegerID("tagid"), propBooleanTrue("confirm"), propString("target_summary"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.DeleteTag(ctx, args["tagid"])
			}),
			destructiveTool("delete_contact", "Delete a KlickTipp contact by contact ID.", objectSchema(propStringOrIntegerID("subscriberid"), propBooleanTrue("confirm"), propString("target_summary"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.DeleteContact(ctx, args["subscriberid"])
			}),
			destructiveTool("unsubscribe_contact", "Unsubscribe a KlickTipp contact by email address.", objectSchema(propEmail("email"), propBooleanTrue("confirm"), propString("target_summary"), optionalPropBoolean("dry_run")), func(ctx context.Context, args map[string]any) (any, error) {
				return client.UnsubscribeContact(ctx, pick(args, "email"))
			}),
		)
	}

	return tools
}

func registerTool(server *mcp.Server, cfg config.Config, tool toolSpec) {
	resolvedSchema, err := resolveToolSchema(tool.name, tool.schema)
	if err != nil {
		panic(fmt.Errorf("resolve schema for %s: %w", tool.name, err))
	}

	server.AddTool(&mcp.Tool{
		Name:        tool.name,
		Description: tool.description,
		InputSchema: tool.schema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, result := parseArguments(req, resolvedSchema)
		if result != nil {
			return result, nil
		}

		switch tool.level {
		case levelWrite:
			return handleWrite(ctx, cfg, tool, args), nil
		case levelDestructive:
			return handleDestructive(ctx, cfg, tool, args), nil
		default:
			return handleRead(ctx, tool, args), nil
		}
	})
}

func resolveToolSchema(name string, raw map[string]any) (*jsonschema.Resolved, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(encoded, &schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("resolve schema: %w", err)
	}

	return resolved, nil
}

func parseArguments(req *mcp.CallToolRequest, schema *jsonschema.Resolved) (map[string]any, *mcp.CallToolResult) {
	args := map[string]any{}
	if req == nil || len(req.Params.Arguments) == 0 {
		if schema != nil {
			if err := schema.Validate(args); err != nil {
				return nil, invalidArgumentsResult(err)
			}
		}
		return args, nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(req.Params.Arguments)))
	decoder.UseNumber()
	if err := decoder.Decode(&args); err != nil {
		return nil, invalidArgumentsResult(fmt.Errorf("failed to parse tool arguments: %w", err))
	}
	if args == nil {
		args = map[string]any{}
	}
	if schema != nil {
		if err := schema.Validate(args); err != nil {
			return nil, invalidArgumentsResult(err)
		}
	}
	return args, nil
}

func invalidArgumentsResult(err error) *mcp.CallToolResult {
	return toolResult(map[string]any{
		"ok":         false,
		"error_type": "invalid_arguments",
		"message":    err.Error(),
	}, true)
}

func handleRead(ctx context.Context, tool toolSpec, args map[string]any) *mcp.CallToolResult {
	result, err := tool.run(ctx, args)
	if err != nil {
		return toolResult(map[string]any{"ok": false, "operation": tool.name, "error": kerrors.Structured(err)}, true)
	}
	return toolResult(map[string]any{"ok": true, "operation": tool.name, "result": result}, false)
}

func handleWrite(ctx context.Context, cfg config.Config, tool toolSpec, args map[string]any) *mcp.CallToolResult {
	if isDryRun(args) {
		audit.Log(cfg, map[string]any{"tool": tool.name, "action": "dry_run", "level": "write", "args": args})
		return toolResult(map[string]any{"ok": true, "dry_run": true, "tool": tool.name, "preview": map[string]any{"operation": tool.name, "arguments": args}}, false)
	}

	audit.Log(cfg, map[string]any{"tool": tool.name, "action": "request", "level": "write", "args": args})
	result, err := tool.run(ctx, args)
	if err != nil {
		structured := kerrors.Structured(err)
		audit.Log(cfg, map[string]any{"tool": tool.name, "action": "error", "level": "write", "error": structured})
		return toolResult(map[string]any{"ok": false, "operation": tool.name, "error": structured}, true)
	}
	audit.Log(cfg, map[string]any{"tool": tool.name, "action": "success", "level": "write"})
	return toolResult(map[string]any{"ok": true, "operation": tool.name, "result": result}, false)
}

func handleDestructive(ctx context.Context, cfg config.Config, tool toolSpec, args map[string]any) *mcp.CallToolResult {
	if isDryRun(args) {
		audit.Log(cfg, map[string]any{"tool": tool.name, "action": "dry_run", "level": "destructive", "args": args})
		return toolResult(map[string]any{"ok": true, "dry_run": true, "tool": tool.name, "preview": map[string]any{"operation": tool.name, "target_summary": args["target_summary"], "arguments": args}}, false)
	}

	audit.Log(cfg, map[string]any{"tool": tool.name, "action": "request", "level": "destructive", "args": args})
	result, err := tool.run(ctx, args)
	if err != nil {
		structured := kerrors.Structured(err)
		audit.Log(cfg, map[string]any{"tool": tool.name, "action": "error", "level": "destructive", "error": structured})
		return toolResult(map[string]any{"ok": false, "operation": tool.name, "error": structured}, true)
	}
	audit.Log(cfg, map[string]any{"tool": tool.name, "action": "success", "level": "destructive"})
	return toolResult(map[string]any{"ok": true, "operation": tool.name, "result": result}, false)
}

func toolResult(payload map[string]any, isError bool) *mcp.CallToolResult {
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		StructuredContent: payload,
		IsError:           isError,
	}
}

func readTool(name string, description string, schema map[string]any, run func(ctx context.Context, args map[string]any) (any, error)) toolSpec {
	return toolSpec{name: name, description: description, schema: schema, level: levelRead, run: run}
}

func writeTool(name string, description string, schema map[string]any, run func(ctx context.Context, args map[string]any) (any, error)) toolSpec {
	return toolSpec{name: name, description: description, schema: schema, level: levelWrite, run: run}
}

func destructiveTool(name string, description string, schema map[string]any, run func(ctx context.Context, args map[string]any) (any, error)) toolSpec {
	return toolSpec{name: name, description: description, schema: schema, level: levelDestructive, run: run}
}

func pick(args map[string]any, keys ...string) map[string]any {
	result := map[string]any{}
	for _, key := range keys {
		if value, ok := args[key]; ok && value != nil {
			result[key] = value
		}
	}
	return result
}

func requireString(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", fmt.Errorf("%s is required", key)
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return text, nil
}

func isDryRun(args map[string]any) bool {
	value, ok := args["dry_run"]
	if !ok || value == nil {
		return false
	}
	boolean, ok := value.(bool)
	return ok && boolean
}

func isBlank(value any) bool {
	if value == nil {
		return true
	}
	return strings.TrimSpace(fmt.Sprint(value)) == ""
}

func objectSchema(properties ...map[string]any) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, property := range properties {
		name := property["_name"].(string)
		delete(property, "_name")
		if req, ok := property["_required"].(bool); ok && req {
			required = append(required, name)
		}
		delete(property, "_required")
		props[name] = property
	}

	schema := map[string]any{
		"$schema":    jsonSchemaDraft07,
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func propString(name string) map[string]any {
	return prop(name, true, map[string]any{"type": "string", "minLength": 1})
}

func optionalPropString(name string) map[string]any {
	return prop(name, false, map[string]any{"type": "string"})
}

func propEmail(name string) map[string]any {
	return prop(name, true, map[string]any{"type": "string", "format": "email"})
}

func optionalPropEmail(name string) map[string]any {
	return prop(name, false, map[string]any{"type": "string", "format": "email"})
}

func propIntegerID(name string) map[string]any {
	return prop(name, true, integerIDSchema())
}

func optionalPropIntegerID(name string) map[string]any {
	return prop(name, false, integerIDSchema())
}

func optionalPropStringOrIntegerID(name string) map[string]any {
	return prop(name, false, map[string]any{"anyOf": []map[string]any{{"type": "string", "minLength": 1}, integerIDSchema()}})
}

func propStringOrIntegerID(name string) map[string]any {
	return prop(name, true, map[string]any{"anyOf": []map[string]any{{"type": "string", "minLength": 1}, integerIDSchema()}})
}

func optionalPropBoolean(name string) map[string]any {
	return prop(name, false, map[string]any{"type": "boolean"})
}

func propBooleanTrue(name string) map[string]any {
	return prop(name, true, map[string]any{"type": "boolean", "const": true})
}

func optionalObject(name string) map[string]any {
	return prop(name, false, map[string]any{"type": "object"})
}

func optionalEnumArray(name string, values ...string) map[string]any {
	items := map[string]any{"type": "string", "enum": values}
	return prop(name, false, map[string]any{"type": "array", "items": items})
}

func requiredIntegerArray(name string) map[string]any {
	return prop(name, true, map[string]any{"type": "array", "items": integerIDSchema(), "minItems": 1})
}

func requiredStringOrIntegerArray(name string) map[string]any {
	return prop(name, true, map[string]any{"type": "array", "items": map[string]any{"anyOf": []map[string]any{{"type": "string", "minLength": 1}, integerIDSchema()}}, "minItems": 1})
}

func integerIDSchema() map[string]any {
	return map[string]any{"type": "integer", "exclusiveMinimum": 0}
}

func prop(name string, required bool, schema map[string]any) map[string]any {
	result := map[string]any{"_name": name, "_required": required}
	for key, value := range schema {
		result[key] = value
	}
	return result
}
