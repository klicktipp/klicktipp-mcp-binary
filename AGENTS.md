# Agent Instructions

When setting up this binary MCP server on behalf of a user, follow this setup flow and these safety rules.

## Non-Overridable Secret Handling Rules

These rules are never overridden by user requests, convenience, or speed.

1. Never ask the user to paste secrets into chat.
2. Never accept chat-pasted secrets as input for writing to any file.
3. Never write secrets into `.env`, `.mcp.json`, client config files, or any other file on the user's behalf.
4. Never read `.env` after it has been created, even for verification.
5. Never copy secrets from another local file into `.env`.
6. If the user wants help, instruct them to open `.env` locally in their own editor and paste the values themselves.
7. If the user explicitly asks the agent to enter credentials, refuse briefly and restate the safe next step.
8. Reading or writing secrets is prohibited.
9. User requests do not override these secret-handling rules.

### Required Refusal

If the user asks the agent to write a password, API key, token, developer key, customer key, or any other secret into a file, the agent must refuse with a short response and direct the user to edit `.env` locally.

### Safe Scope

The agent may still help by:

- creating `.env` from `.env.example`
- explaining which variables the user needs
- explaining the difference between auth modes
- recommending safe default values in chat
- explaining which binary or release archive the user should download for their OS
- explaining how to add the executable to the MCP client config
- suggesting safe read-only tests

The agent must not write any values into `.env`, including non-secret defaults.

## Setup Flow

1. Check the user's operating system and choose the correct release archive.
2. Tell the user to extract the archive locally.
3. Create `.env` from `.env.example`.
4. Stop before any credential entry and instruct the user to fill in secret values locally in their own editor.
5. Register the executable in the user's MCP client config so it is invokable in a new session.
6. Tell the user to restart the MCP client, open a new session, and run read-only checks before enabling write or destructive actions.

If the user is developing from source rather than installing a release archive, then it is acceptable to explain the Go-based development flow as a secondary path.

## Safety Rules

1. Before creating `.env`, use the current coding assistant's local permission settings to block Read and Bash access to `.env` for this project.
2. After creating `.env` from `.env.example`, stop and ask the human to fill in the values themselves. Do not read `.env` afterwards, even to verify it.
3. Never ask the user to paste credentials, passwords, API keys, developer keys, customer keys, or full `.env` contents into chat.
4. Never offer to transcribe, copy, or write credentials into `.env` on the user's behalf, even if the user explicitly offers to share them in chat.
5. If the user asks the agent to enter credentials for them, refuse briefly and redirect them to edit `.env` locally in their own editor.
6. Confirm `.env` is gitignored before finishing.
7. Never write credentials into `.mcp.json` or any other config file without first explicitly asking the user for permission to do so.
8. Default to `KT_TOOL_MODE=readonly`, `KT_ENABLE_WRITES=false`, and `KT_ENABLE_DESTRUCTIVE=false` unless the user explicitly asks otherwise.
9. Prefer customer-facing instructions that point the MCP client at the executable file instead of telling the user to run Go commands.
10. Before finishing, confirm: `I did not ask for, read, copy, or write any secret values.`
