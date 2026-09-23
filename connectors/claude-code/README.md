# Claude Code

Sources (checked 2026-09-23): https://code.claude.com/docs/en/mcp and https://code.claude.com/docs/en/skills

## Project-level install (recommended)

This repository already contains both pieces:

- `.mcp.json`: a project-scoped stdio server named `trimble`. By default it runs the **mock adapter only**, with no credentials. Claude Code asks you to approve project servers the first time; `claude mcp reset-project-choices` resets that choice.
- `.claude/skills/trimble/`: the skill, generated from `skills/canonical/trimble` by `scripts/sync-skills.sh`.

To add the server to another repository, use an installed binary rather than `go run`:

```sh
go install github.com/1a-li-lu-le-lo/trimblemcp/cmd/trimble-mcp@latest
claude mcp add --transport stdio --scope project trimble \
  --env TRIMBLE_MCP_ENABLE_MOCK=false --env TRIMBLE_CONNECT_ENABLED=true \
  --env TRIMBLE_CONNECT_ENV=stage --env TRIMBLE_CONNECT_REGION=us \
  --env TRIMBLE_TOKEN_STORE=/secure/path/tc-token --env TRIMBLE_TOKEN_KEY_FILE=/secure/path/tc-token.key \
  -- trimble-mcp -transport stdio
cp -R skills/canonical/trimble <other-repo>/.claude/skills/trimble   # omit evals/ if you like
```

`.mcp.json` supports `${VAR}` expansion, so paths can come from the environment instead of being committed.

User-level install (`--scope user`, `~/.claude/skills/`) is only justified for an operator who uses one tenant across many repositories.

## Authentication

- **stdio:** the server acts as the local operator with read-only scopes (`TRIMBLE_MCP_LOCAL_SCOPES`, default `trimble:capabilities:read trimble:projects:read trimble:files:read`). Write and delete scopes are rejected at startup.
- **Trimble Connect:** an operator runs `trimblectl auth login` once. Use sandbox (`stage`) credentials. Never paste tokens into the conversation.

## Remote (HTTP) alternative

```sh
claude mcp add --transport http --scope project trimble https://mcp.example.com/mcp \
  --header "Authorization: Bearer ${TRIMBLE_MCP_TOKEN}"
```

## Activation test

1. Ask: "List the projects in the mock Trimble product." The skill should call `trimble_get_capabilities`, then `trimble_list_projects` with `product: "mock"`, and report 7 simulated projects after following pagination.
2. Ask: "Explain trim in Go strings." The skill must not activate.
3. Run `go test ./internal/skilltest/` to check that the skill and server agree.

## What Claude Code may and must not do

It may build adapters, generate typed clients, write tests, inspect authorized project metadata, and troubleshoot integration code.

It must not receive production secrets in prompts, mutate production projects (no mutation tools exist), bypass authorization, operate machinery, or certify field data.

## Trimble Connect for Windows command line (local only)

To build `trimbleconnect:` links, set `TRIMBLE_CONNECT_DESKTOP_ENABLED=true`. To let the agent open the app on this Windows machine, also set:

- `TRIMBLE_CONNECT_DESKTOP_LAUNCH=true`
- `TRIMBLE_MCP_LOCAL_SCOPES=trimble:capabilities:read,trimble:projects:read,trimble:files:read,trimble:desktop:launch`

Launching is a dry run unless the agent passes `dry_run: false`. The client will also ask you to approve the tool, because it is not read-only.

## Removal

```sh
claude mcp remove trimble --scope project   # or delete the entry from .mcp.json
rm -rf .claude/skills/trimble
trimblectl auth logout                      # revokes the Trimble refresh token
```
