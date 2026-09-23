# OpenAI Codex

Sources (checked 2026-09-23):

- https://learn.chatgpt.com/docs/extend/mcp?surface=cli
- https://learn.chatgpt.com/docs/build-skills
- https://learn.chatgpt.com/docs/config-file/config-reference

## Install

1. Install the binaries: `go install github.com/1a-li-lu-le-lo/trimblemcp/cmd/...@latest`.
2. Add the server. Either merge `connectors/codex/config.toml` into `~/.codex/config.toml`, or run:
   `codex mcp add trimble --env TRIMBLE_MCP_ENABLE_MOCK=true -- trimble-mcp -transport stdio`
3. Add the skill. This repository ships it at `.agents/skills/trimble/`, which Codex discovers from `$REPO_ROOT/.agents/skills`. For other repositories, copy `skills/canonical/trimble` to `<repo>/.agents/skills/trimble` or to `$HOME/.agents/skills/trimble`. Invoke it explicitly with `$trimble`.

## Network and sandbox

- The stdio server needs outbound HTTPS only when Trimble Connect is enabled: `*.connect.trimble.com` and `id.trimble.com` (plus the staging hosts for sandbox).
- The mock adapter needs no network access.
- Use development or sandbox credentials only. Do not expose production Trimble credentials to coding environments.
- `enabled_tools` in the example config pins the read-only tool set.

## Test

`codex mcp list` should show `trimble`. Then ask: "Use $trimble to list projects in the mock product." Expect 7 simulated projects and one `audit_id` per call.

## Removal

Delete the `[mcp_servers.trimble]` block (or run `codex mcp remove trimble` if your version has it), delete `.agents/skills/trimble`, and run `trimblectl auth logout`.

## Unverified

The remote form `codex mcp add <name> --url <url> --bearer-token-env-var ENV` appears in third-party documentation, not on an OpenAI page. Prefer the `config.toml` keys `url` and `bearer_token_env_var`, which the official config reference documents.
