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

Default rule: use the published release archive, not a source checkout.

1. Detect the user's operating system and architecture.
2. Identify the matching published release asset.
3. Download that release archive.
4. Tell the user to extract the archive locally, or extract it if local policy allows.
5. Create `.env` from `.env.example`.
6. Stop before any credential entry and instruct the user to fill in secret values locally in their own editor.
7. Register the extracted executable in the user's MCP client config so it is invokable in a new session.
8. Tell the user to restart the MCP client, open a new session, and run read-only checks before enabling write or destructive actions.

Source setup is allowed only if at least one of the following is true:

1. The user explicitly asked for a source or development setup.
2. No compatible release asset exists.
3. The user is actively developing this repository.

If none of those conditions is true, do not clone the repository and do not build from source.

## Installation Path Guardrails

1. Do not replace the release-install path with a source build for convenience or speed.
2. If release asset discovery fails because of sandbox or network restrictions, request permission to inspect the release metadata instead of switching to `git clone`.
3. If release asset names are unclear, stop and report that uncertainty to the user before taking a different installation path.
4. Treat source installation as a fallback that requires explicit justification in the final response.

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
