# Trimble MCP Bridge

A Go MCP server and cross-agent skill that give AI agents narrow, audited, **read-only** access to verified Trimble product APIs.

The first adapter is the **Trimble Connect** REST API: projects, folder items, and file metadata. A simulated `mock` adapter lets everything run without credentials.

> "Trimble API" is not one API. Each product is researched, verified, and adapted separately. See [docs/trimble-products/capability-matrix.md](docs/trimble-products/capability-matrix.md).

## Quick start (no credentials)

```sh
make build test smoke
./bin/trimblectl capabilities
./bin/trimblectl projects list --product mock --page-size 3
```

- **Claude Code:** open this repository. `.mcp.json` registers the `trimble` server (mock only), and `.claude/skills/trimble` provides the skill.
- **Codex:** see `connectors/codex/`.
- **Perplexity:** see `connectors/perplexity/`.
- **Claude app:** see `connectors/claude-app/`.

## What is in the box

| Path | Contents |
|---|---|
| `cmd/trimble-mcp` | MCP server: stdio, and Streamable HTTP with TLS, bearer auth, and Origin/Host checks |
| `cmd/trimblectl` | Operator CLI: data commands, `auth login`/`logout` (PKCE), `audit verify`, `token new`, `diagnostics` |
| `internal/mcp` | Standard-library JSON-RPC and MCP implementation (legacy 2025-xx plus modern 2026-07-28) |
| `internal/gateway` | Tools, authorization, audit, rate limits, output envelope, untrusted-text handling |
| `internal/trimble/connect` | Trimble Connect adapter (verified endpoints only) |
| `internal/identity` | Trimble Identity OAuth (PKCE, refresh, revoke) and encrypted token store |
| `internal/domain` | Typed IDs, pagination, CRS, units, positions, provenance |
| `skills/canonical/trimble` | Canonical `SKILL.md`, references, templates, and evals |
| `.claude/skills/trimble`, `.agents/skills/trimble` | Claude Code and Codex installs (synced by `scripts/sync-skills.sh`) |
| `connectors/` | Client setup: Claude Code, Codex, Perplexity, Claude app |
| `docs/` | Capability matrix, requirements traceability, ADRs, threat model, privacy, operations |

## Tools

- `trimble_get_capabilities`
- `trimble_list_projects`
- `trimble_get_project` (listed only where the upstream endpoint is verified)
- `trimble_list_folder_items`
- `trimble_get_file_metadata`
- `trimble_build_desktop_link`: the [Trimble Connect for Windows command line](https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line), `trimbleconnect:/projects/<id>?show=<view>,<panel>`.
- `trimble_open_in_desktop`: opens that link locally. Windows and the local operator only, opt-in, dry run by default.

There are no data-mutation, download, shell, HTTP-proxy, or machinery tools. See [docs/mcp/tools.md](docs/mcp/tools.md).

## Safety properties

- **Tenant isolation:** isolated by construction.
- **Project binding:** every file call is checked against the authorized project.
- **Input:** strict schemas; unknown fields are rejected.
- **Secrets:** never logged or returned. Refresh tokens are encrypted at rest; HTTP tokens are stored only as hashes.
- **Audit:** append-only and hash-chained. Calls fail closed if the audit record cannot be written.
- **Untrusted text:** upstream names are cleaned and labelled as untrusted.
- **Geospatial:** CRS and axis order are explicit, and the US survey foot and international foot are kept distinct.

## Status

This is a **pilot-grade first vertical slice**, not production-ready. Production blockers are listed in [docs/requirements/open-questions.md](docs/requirements/open-questions.md) and in the traceability matrix. The main ones: sandbox credentials, the full Connect OpenAPI re-verification, and an OAuth 2.1 resource server for remote clients.
