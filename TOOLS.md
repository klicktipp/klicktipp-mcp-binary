# KlickTipp Binary MCP Tool Reference

This file lists the MCP tools, expected inputs, and example outputs for the current binary server snapshot.

After you configure the executable in your MCP client, these are the tools you can call through that client.

The server exposes read-only, write, and destructive KlickTipp Management API tools, with the safer default setup being read-only until you explicitly enable write or destructive actions in `.env`.

Authentication endpoints are handled internally by the server and are not exposed as user-facing MCP tools.

## Read tools

### `list_tags`

Returns the manually discoverable tag catalog as `id` / `name` pairs.

Important:

- This is the manual tag catalog only.
- Smart Tags do not currently have an independent catalog endpoint in this MCP server.
- To work with a Smart Tag, first get a known Smart Tag ID from `get_contact`, then use `get_tag` or `search_tagged_contacts` with that ID.

Example result:

```json
{
  "ok": true,
  "operation": "list_tags",
  "result": {
    "count": 2,
    "items": [
      { "id": "21", "name": "Customers" },
      { "id": "22", "name": "Webinar" }
    ]
  }
}
```

### `list_fields`

Returns available data fields as `id` / `name` pairs.

Example result:

```json
{
  "ok": true,
  "operation": "list_fields",
  "result": {
    "count": 2,
    "items": [
      { "id": "fieldFirstName", "name": "First Name" },
      { "id": "field203826", "name": "Custom Field" }
    ]
  }
}
```

### `get_field`

Use the field ID returned by `list_fields`.

Examples:

- `fieldFirstName`
- `field203826`

The server also accepts raw API field IDs such as `FirstName` and `203826`.

### `list_opt_in_processes`

Returns opt-in processes as `id` / `name` pairs.

### `list_contacts`

Optional filters:

- `status`: `subscribed`, `pending`, `unsubscribed`
- `bounceStatus`: `nobounce`, `softbounce`, `hardbounce`, `spambounce`

Use the exact API values above. Labels such as `Subscribed` or `Not Bounced` are not valid.

Example input:

```json
{
  "status": ["subscribed"],
  "bounceStatus": ["nobounce"]
}
```

### `get_tag`

Gets one tag by known `tagid`.

This supports:

- manual tag IDs
- known Smart Tag IDs

The result includes lookup context. When the API response exposes enough metadata, the server also marks the tag as manual or `system_managed` / read-only.

If a tag lookup fails, the error message explains that manual tags can be discovered with `list_tags`, while Smart Tags currently require a known ID from contact context.

### `get_contact`

Gets one contact by `subscriberid`.

The response includes a normalized `tag_context` block with:

- `manual_tag_ids`
- `smart_tag_ids`
- `all_tag_ids`
- normalized `manual_tags` and `smart_tags` entries

This is the main read path for obtaining stable Smart Tag IDs for follow-up context.

### `search_tagged_contacts`

Required input:

- `tagid`

Optional filters:

- `status`: `subscribed`, `pending`, `unsubscribed`
- `bounceStatus`: `nobounce`, `softbounce`, `hardbounce`, `spambounce`

Use the exact API values above.

`tagid` may be:

- a manual tag ID
- a known Smart Tag ID

Smart Tags are supported here by known ID, but they are not independently discoverable through a separate catalog/list endpoint.

## Write tools

These tools require:

```dotenv
KT_TOOL_MODE=full
KT_ENABLE_WRITES=true
KT_ENABLE_DESTRUCTIVE=false
```

### `create_tag`

Example input:

```json
{
  "name": "New Tag"
}
```

Example result:

```json
{
  "ok": true,
  "operation": "create_tag",
  "result": {
    "id": 14790933
  }
}
```

### `update_tag`

Update an existing tag by `tagid`.

### `create_or_update_contact`

Requires at least one of:

- `email`
- `smsnumber`

### `update_contact`

Updates an existing contact by `subscriberid`.

### `tag_contact`

Assigns one or more tag IDs to a contact.

### `untag_contact`

Removes one tag ID from a contact.

## Destructive tools

These tools require:

```dotenv
KT_TOOL_MODE=full
KT_ENABLE_WRITES=true
KT_ENABLE_DESTRUCTIVE=true
```

Before any real destructive action, run the same tool first with `dry_run=true`.

Destructive tools also require:

- `confirm: true`
- `target_summary`

### `delete_tag`

Deletes a tag by `tagid`.

### `delete_contact`

Deletes a contact by `subscriberid`.

### `unsubscribe_contact`

Unsubscribes a contact by `email`.

## Dry-run

Write and destructive tools support `dry_run=true`.

Example:

```json
{
  "subscriberid": 456,
  "confirm": true,
  "target_summary": "Delete contact 456",
  "dry_run": true
}
```

This previews the action without changing data in KlickTipp.

## Tool usage notes

- Field keys must match KlickTipp field IDs exactly, for example `fieldFirstName`.
- Date and date-time custom fields should use Unix timestamps in seconds.
- `list_tags` is the manual tag catalog only; use `get_contact` to obtain known `smart_tag_ids`.
- Smart Tags are treated as read-only/system-managed context in this MCP server. No Smart Tag write operation is added or implied.
- Write and destructive tools return dry-run previews when `dry_run=true`.
- Destructive tools require `confirm: true` and `target_summary`.
