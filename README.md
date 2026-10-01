# Trimble MCP Bridge

A Go MCP server and cross-agent skill that give AI agents narrow, audited, **read-only** access to verified Trimble product APIs.

The first adapter is the **Trimble Connect** REST API: projects, folder items, and file metadata. A simulated `mock` adapter lets everything run without credentials.

**Coverage:** every Trimble API operation with a public definition is accounted for: **13,928 operations in 381 definitions**.

- **Trimble Connect** (11 APIs): reads are executable, and changes are dry-run plans. Its prose-documented Core Account endpoints are catalogued too.
- **Other products** (60 reference APIs: Vista, ProjectSight, Viewpoint For Projects, Unity Construct, Unity Maintain/Permit, Accubid Anywhere, Civil Site Management, Geospatial field services, Mobile Manager, TMT, TMWSuite, TruckMate, Transporeon, Trimble Maps, PC*MILER, PTx FarmENGAGE, MEPcontent, Jobpac, Tekla PowerFab Go, Trimble Connect Status Sharing): documented and plannable, never called.
- **Copies:** staging, QA and draft copies, and duplicate publications, are `variant`s that point at the operation they copy.
- **Excluded, with a reason:** internal App Xchange connector definitions, push and webhook contracts, undocumented or deprecated definitions, credential operations, and safety exclusions.
- **Trimble Identity:** every endpoint is catalogued and excluded for agents.

Every Trimble command-line tool is accounted for as well. See [docs/trimble-products/capability-matrix.md](docs/trimble-products/capability-matrix.md) and [docs/trimble-products/cli-coverage.md](docs/trimble-products/cli-coverage.md).

> "Trimble API" is not one API. Each product is researched, verified, and adapted separately. See [docs/trimble-products/capability-matrix.md](docs/trimble-products/capability-matrix.md).

**Documentation:** start at [docs/README.md](docs/README.md), which indexes every document. To deploy and test, follow [docs/operations/testing-guide.md](docs/operations/testing-guide.md).

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
| `cmd/trimble-catalog`, `internal/catalog` | Generator and embedded catalogue of every published Trimble API operation (`make catalog`); `scripts/fetch-trimble-*` retrieve the definitions |
| `docs/` | Capability matrix, requirements traceability, ADRs, threat model, privacy, operations |

## Tools

- `trimble_get_capabilities`
- `trimble_list_projects`
- `trimble_get_project` (listed only where the upstream endpoint is verified)
- `trimble_list_folder_items`
- `trimble_get_file_metadata`
- `trimble_api_operations`, `trimble_api_read`, `trimble_api_plan`: every catalogued operation (13,928 in 381 definitions).
  - **Trimble Connect:** production reads are executable, and changes are dry-run planned.
  - **Other products:** operations are `reference`, which means searchable and dry-run planned, never called.
  - **Copies:** `variant`, pointing at the operation they duplicate.
  - **Everything else:** excluded with a reason, including safety exclusions for anything that could reach vehicles, machinery or field positioning, and every credential operation.

  See [docs/trimble-products/endpoints](docs/trimble-products/endpoints/README.md).
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

This is a **pilot-grade first vertical slice**, not production-ready. Production blockers are listed in [docs/requirements/open-questions.md](docs/requirements/open-questions.md) and in the traceability matrix. The main ones: sandbox credentials (with a registered callback URL), live verification of the Trimble Connect adapter against a sandbox, and an OAuth 2.1 resource server for remote clients.
