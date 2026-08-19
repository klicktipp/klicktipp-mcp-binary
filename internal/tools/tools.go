package tools

import (
	"context"
	"encoding/json"
	"errors"
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

const smartTagDiscoveryNote = "Smart Tags do not currently have an independently discoverable catalog in this MCP server. Use get_contact to obtain known smart_tag_ids, then call get_tag or search_tagged_contacts with one of those IDs."

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
		readTool("list_tags", "List the manually discoverable KlickTipp tag catalog as id/name pairs. Smart Tags do not currently have a separate catalog endpoint here; use get_contact to obtain known smart_tag_ids, then use get_tag or search_tagged_contacts by ID.", objectSchema(), func(ctx context.Context, args map[string]any) (any, error) {
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
			return map[string]any{
				"count": len(items),
				"items": items,
				"discovery": map[string]any{
					"manual_tag_catalog_supported": true,
					"smart_tag_catalog_supported":  false,
					"note":                         smartTagDiscoveryNote,
				},
			}, nil
		}),
		readTool("get_tag", "Get one KlickTipp tag by known tag ID. This supports manual tag IDs and known Smart Tag IDs. Smart Tags are read-only context here and do not currently have a separate catalog endpoint.", objectSchema(propStringOrIntegerID("tagid")), func(ctx context.Context, args map[string]any) (any, error) {
			result, err := client.GetTag(ctx, args["tagid"])
			if err != nil {
				return nil, annotateKnownTagLookupError(args["tagid"], err)
			}
			manualTagIDs, err := client.ListTags(ctx)
			if err != nil {
				return nil, err
			}
			return enrichTagLookupResult(result, args["tagid"], manualTagIDs), nil
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
		readTool("get_contact", "Get one KlickTipp contact by contact ID. The response includes normalized tag_context data with stable manual_tag_ids and smart_tag_ids for follow-up read operations.", objectSchema(propStringOrIntegerID("subscriberid")), func(ctx context.Context, args map[string]any) (any, error) {
			result, err := client.GetContact(ctx, args["subscriberid"])
			if err != nil {
				return nil, err
			}
			return enrichContactResult(result), nil
		}),
		readTool("search_contact", "Search for a KlickTipp contact ID by email address.", objectSchema(propEmail("email")), func(ctx context.Context, args map[string]any) (any, error) {
			email, err := requireString(args, "email")
			if err != nil {
				return nil, err
			}
			return client.SearchContact(ctx, map[string]any{"email": email})
		}),
		readTool("search_tagged_contacts", "List KlickTipp contacts for one known tag ID, with optional status and bounce status filters. This supports manual tag IDs and known Smart Tag IDs. Smart Tags do not currently have a separate discovery/catalog endpoint. Allowed status values: subscribed, pending, unsubscribed. Allowed bounceStatus values: nobounce, softbounce, hardbounce, spambounce.", objectSchema(propStringOrIntegerID("tagid"), optionalEnumArray("status", "subscribed", "pending", "unsubscribed"), optionalEnumArray("bounceStatus", "nobounce", "softbounce", "hardbounce", "spambounce")), func(ctx context.Context, args map[string]any) (any, error) {
			result, err := client.SearchTaggedContacts(ctx, pick(args, "tagid", "status", "bounceStatus"))
			if err != nil {
				return nil, annotateKnownTagLookupError(args["tagid"], err)
			}
			return result, nil
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

func enrichTagLookupResult(result any, requestedTagID any, manualTagIDs map[string]string) any {
	tag, ok := result.(map[string]any)
	if !ok {
		return result
	}

	enriched := cloneMap(tag)
	enriched["lookup_context"] = map[string]any{
		"known_id_lookup_supported":   true,
		"smart_tag_catalog_supported": false,
		"note":                        smartTagDiscoveryNote,
	}

	if management, ok := inferTagManagement(tag, requestedTagID, manualTagIDs); ok {
		enriched["management"] = management
	}
	return enriched
}

func enrichContactResult(result any) any {
	contact, ok := result.(map[string]any)
	if !ok {
		return result
	}

	enriched := cloneMap(contact)
	manualEntries, manualIDs := normalizeTagAssignments(contact["manual_tags"])
	smartEntries, smartIDs := normalizeTagAssignments(contact["smart_tags"])
	allIDs := append([]string{}, manualIDs...)
	allIDs = append(allIDs, smartIDs...)
	sort.Strings(allIDs)

	enriched["tag_context"] = map[string]any{
		"manual_tag_ids":              manualIDs,
		"smart_tag_ids":               smartIDs,
		"all_tag_ids":                 dedupeSortedStrings(allIDs),
		"manual_tags":                 manualEntries,
		"smart_tags":                  smartEntries,
		"smart_tag_catalog_supported": false,
		"note":                        smartTagDiscoveryNote,
	}
	return enriched
}

func inferTagManagement(tag map[string]any, requestedTagID any, manualTagIDs map[string]string) (map[string]any, bool) {
	if value, ok := firstMapValue(tag, "read_only", "readonly", "system_managed", "systemManaged", "is_smart_tag", "isSmartTag", "smart_tag", "smartTag"); ok {
		if boolean, ok := toBool(value); ok {
			if boolean {
				return map[string]any{
					"mode":      "system_managed",
					"read_only": true,
					"note":      "This tag was identified as system-managed/read-only from the API response.",
				}, true
			}
			return map[string]any{
				"mode":      "manual",
				"read_only": false,
				"note":      "This tag was identified as a manual tag from the API response.",
			}, true
		}
	}

	if value, ok := firstMapValue(tag, "type", "tag_type", "tagType", "kind"); ok {
		normalized := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		if normalized != "" {
			switch normalized {
			case "smart", "smart_tag", "smarttag", "system", "system_managed":
				return map[string]any{
					"mode":      "system_managed",
					"read_only": true,
					"note":      "This tag was identified as system-managed/read-only from the API response.",
				}, true
			case "manual":
				return map[string]any{
					"mode":      "manual",
					"read_only": false,
					"note":      "This tag was identified as a manual tag from the API response.",
				}, true
			}
		}
	}

	if isManualTagID(requestedTagID, tag, manualTagIDs) {
		return map[string]any{
			"mode":      "manual",
			"read_only": false,
			"note":      "This tag was identified as manual because its ID appears in the manual tag catalog returned by list_tags.",
		}, true
	}

	return nil, false
}

func normalizeTagAssignments(raw any) ([]map[string]any, []string) {
	assignments, ok := raw.(map[string]any)
	if !ok || len(assignments) == 0 {
		return []map[string]any{}, []string{}
	}

	ids := make([]string, 0, len(assignments))
	for id := range assignments {
		ids = append(ids, fmt.Sprint(id))
	}
	sort.Strings(ids)

	entries := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		entry := map[string]any{"id": id}
		if assignedAt, ok := assignments[id]; ok && assignedAt != nil {
			entry["assigned_at"] = assignedAt
		}
		entries = append(entries, entry)
	}

	return entries, ids
}

func dedupeSortedStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	deduped := make([]string, 0, len(items))
	var previous string
	for index, item := range items {
		if index == 0 || item != previous {
			deduped = append(deduped, item)
			previous = item
		}
	}
	return deduped
}

func cloneMap(value map[string]any) map[string]any {
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func firstMapValue(value map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if item, ok := value[key]; ok {
			return item, true
		}
	}
	return nil, false
}

func toBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	}
	return false, false
}

func isManualTagID(requestedTagID any, tag map[string]any, manualTagIDs map[string]string) bool {
	if len(manualTagIDs) == 0 {
		return false
	}

	candidates := []string{strings.TrimSpace(fmt.Sprint(requestedTagID))}
	if value, ok := firstMapValue(tag, "tagid", "id"); ok {
		candidates = append(candidates, strings.TrimSpace(fmt.Sprint(value)))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := manualTagIDs[candidate]; ok {
			return true
		}
	}
	return false
}

func annotateKnownTagLookupError(tagID any, err error) error {
	var apiErr *kerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 404 {
		return err
	}

	return fmt.Errorf("tag %v was not found. Manual tags can be discovered with list_tags, but Smart Tags cannot currently be discovered independently; use get_contact to obtain a known smart_tag_id before retrying: %w", tagID, err)
}
