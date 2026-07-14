# KlickTipp Binary MCP Tool Reference

This file lists the MCP tools, expected inputs, and example outputs for the current binary server snapshot.

After you configure the executable in your MCP client, these are the tools you can call through that client.

The server exposes read-only, write, and destructive KlickTipp Management API tools, with the safer default setup being read-only until you explicitly enable write or destructive actions in `.env`.

Authentication endpoints are handled internally by the server and are not exposed as user-facing MCP tools.

## Read tools

### `list_tags`

Returns available tags as `id` / `name` pairs.

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

### `search_tagged_contacts`

Required input:

- `tagid`

Optional filters:

- `status`: `subscribed`, `pending`, `unsubscribed`
- `bounceStatus`: `nobounce`, `softbounce`, `hardbounce`, `spambounce`

Use the exact API values above.

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
- Write and destructive tools return dry-run previews when `dry_run=true`.
- Destructive tools require `confirm: true` and `target_summary`.
