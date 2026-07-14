# KlickTipp Binary MCP

Local stdio MCP server for the KlickTipp Management API.

For the tool list, supported inputs, examples, and common response shapes, see [TOOLS.md](./TOOLS.md).

## Installation

The required setup flow is:

1. Download the release archive for your operating system.
2. Extract the archive.
3. Copy `.env.example` to `.env`.
4. Fill in credentials locally in your own editor.
5. Add the executable to your MCP client config.
6. Restart the MCP client.
7. Test read-only tools first.

## Authentication

This server supports two authentication modes.

This repository does not include any credentials.

**Recommendation:** for local, individual development setups, use Option 1, Session Authentication (username + password).
Reserve Option 2, Partner Authentication, for production-ready, partner, or customer-facing integrations.

### Option 1. Session Authentication: Username + password

Use:

- `KT_AUTH_MODE=session`
- `KT_USERNAME`
- `KT_PASSWORD`

This mode logs in to `/account/login` and reuses the returned session.

### Option 2. Partner Authentication: Developer key + customer key

Use:

- `KT_AUTH_MODE=partner`
- `KT_USERNAME`
- `KT_DEVELOPER_KEY`
- `KT_CUSTOMER_KEY`

This mode sends:

- `X-Un`
- `X-Ci`

`X-Ci` is derived from the developer key and customer key.

## Environment setup

Create the environment file:

```bash
cp .env.example .env
```

Open `.env` in your own editor and fill in the values. Do not paste credentials into a chat or AI assistant, and do not let an AI assistant write them into the file for you.

If an AI assistant is helping with setup, the safe stopping point is: create `.env`, then stop. The assistant should not ask you to paste secrets into chat, should not offer to transcribe them, and should not write them into `.env` on your behalf.

Session authentication example read tools only:

```dotenv
KT_BASE_URL=https://api.klicktipp.com
KT_TIMEOUT_MS=30000
KT_AUTH_MODE=session
KT_TOOL_MODE=readonly
KT_ENABLE_WRITES=false
KT_ENABLE_DESTRUCTIVE=false
KT_AUDIT_LOGS=true
KT_USERNAME=your-klicktipp-username
KT_PASSWORD=your-password
```

Partner authentication example read tools only:

```dotenv
KT_BASE_URL=https://api.klicktipp.com
KT_TIMEOUT_MS=30000
KT_AUTH_MODE=partner
KT_TOOL_MODE=readonly
KT_ENABLE_WRITES=false
KT_ENABLE_DESTRUCTIVE=false
KT_AUDIT_LOGS=true
KT_USERNAME=your-klicktipp-username
KT_DEVELOPER_KEY=your-developer-key-hex
KT_CUSTOMER_KEY=your-customer-key
```

To enable write tools:

```dotenv
KT_TOOL_MODE=full
KT_ENABLE_WRITES=true
KT_ENABLE_DESTRUCTIVE=false
```

To enable destructive tools as well:

```dotenv
KT_TOOL_MODE=full
KT_ENABLE_WRITES=true
KT_ENABLE_DESTRUCTIVE=true
```

## Client configuration

### Claude Desktop

Add the server to:

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

Example:

```json
{
  "mcpServers": {
    "klicktipp": {
      "command": "/absolute/path/to/extracted-folder/klicktipp-mcp",
      "args": []
    }
  }
}
```

### Codex

Add the server to:

- `~/.codex/config.toml`

Example:

```toml
[mcp_servers.klicktipp]
command = "/absolute/path/to/extracted-folder/klicktipp-mcp"
args = []
```

### Cursor

Add the server to one of:

- `~/.cursor/mcp.json`
- `.cursor/mcp.json`

Example:

```json
{
  "mcpServers": {
    "klicktipp": {
      "command": "/absolute/path/to/extracted-folder/klicktipp-mcp",
      "args": []
    }
  }
}
```

## Restart After Changes

After adding the server to your MCP client (Codex, Claude, Cursor):

1. Fully close the client.
2. Start it again.
3. Open a new chat.

If you change `.env` values later, for example feature flags such as `KT_TOOL_MODE`, `KT_ENABLE_WRITES`, or `KT_ENABLE_DESTRUCTIVE`, restart the MCP client again before testing.

## Testing recommendations

After configuring the server in your MCP client, try:

- `List my KlickTipp tags`
- `List my KlickTipp fields`
- `List my KlickTipp opt-in processes`

Then try a safe write operation with a test contact:

- `Create or update a KlickTipp contact with email testemail@example.com and fieldFirstName=Test`
- `Tag testemail@example.com with tag ID 12345`

Before any real destructive action, run the same destructive tool first with `dry_run=true` and verify the target carefully.

## AI Agent Setup

If an AI coding assistant is setting up this server on your behalf, it should follow the repository instructions for handling `.env`, credentials, and MCP client configuration.

See [AGENTS.md](./AGENTS.md) for the machine-readable setup instructions and safety rules.
