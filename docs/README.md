# Trimble MCP Bridge documentation

Start here. Every document in the repository is listed below. `internal/skilltest` checks that each one is linked from this index.

## Getting started

| Document | What it covers |
|---|---|
| [../README.md](../README.md) | Project overview, quick start, tool list, status |
| [operations/testing-guide.md](operations/testing-guide.md) | Step-by-step deploy-and-test checklist with expected results (local, sandbox, Windows, remote) |
| [operations/configuration.md](operations/configuration.md) | Every environment variable, default and scope; secret files on Windows |
| [operations/deployment.md](operations/deployment.md) | Local, sandbox and remote (HTTPS/Docker) deployment; kill switch and rollback |

## Connecting agents

| Document | What it covers |
|---|---|
| [../connectors/claude-code/README.md](../connectors/claude-code/README.md) | Claude Code: project `.mcp.json`, skill install, remote setup, removal |
| [../connectors/codex/README.md](../connectors/codex/README.md) | OpenAI Codex: `config.toml`, `.agents/skills`, removal |
| [../connectors/codex/config.toml](../connectors/codex/config.toml) | Ready-to-merge Codex configuration |
| [../connectors/claude-app/README.md](../connectors/claude-app/README.md) | Claude app custom connector (remote HTTPS) |
| [../connectors/perplexity/README.md](../connectors/perplexity/README.md) | Perplexity local and remote connectors |

## What the MCP covers

| Document | What it covers |
|---|---|
| [mcp/tools.md](mcp/tools.md) | Every MCP tool, resource and prompt; output envelope; protocol revisions |
| [trimble-products/capability-matrix.md](trimble-products/capability-matrix.md) | Every Trimble product family with its API access class and disposition |
| [trimble-products/endpoints/README.md](trimble-products/endpoints/README.md) | Every Trimble Connect API operation (all 34 official definitions) with its disposition; per-API references are linked from there |
| [trimble-products/cli-coverage.md](trimble-products/cli-coverage.md) | Every Trimble command-line surface and every `trimblectl` command |
| [trimble-products/trimble-connect.md](trimble-products/trimble-connect.md) | Field-level trace of the typed Trimble Connect tools to the OpenAPI definition |

## Agent skill

| Document | What it covers |
|---|---|
| [../skills/canonical/trimble/SKILL.md](../skills/canonical/trimble/SKILL.md) | Canonical skill (copied to `.claude/skills/trimble` and `.agents/skills/trimble` by `scripts/sync-skills.sh`) |
| [../skills/canonical/trimble/references/product-selection.md](../skills/canonical/trimble/references/product-selection.md) | Choosing the Trimble product |
| [../skills/canonical/trimble/references/api-catalog.md](../skills/canonical/trimble/references/api-catalog.md) | Using the API catalogue tools |
| [../skills/canonical/trimble/references/files.md](../skills/canonical/trimble/references/files.md) | Projects, folders, files |
| [../skills/canonical/trimble/references/desktop.md](../skills/canonical/trimble/references/desktop.md) | Trimble Connect for Windows links |
| [../skills/canonical/trimble/references/authentication.md](../skills/canonical/trimble/references/authentication.md) | Access without handling secrets |
| [../skills/canonical/trimble/references/geospatial.md](../skills/canonical/trimble/references/geospatial.md) | CRS and unit rules |
| [../skills/canonical/trimble/references/safety.md](../skills/canonical/trimble/references/safety.md) | Prohibited actions and output labels |
| [../skills/canonical/trimble/references/errors.md](../skills/canonical/trimble/references/errors.md) | Error codes and remediation |
| [../skills/canonical/trimble/templates/operation-plan.md](../skills/canonical/trimble/templates/operation-plan.md) | Operation plan template |
| [skills/evaluation.md](skills/evaluation.md) | How the skill and its evals are tested |

## Architecture and decisions

| Document | What it covers |
|---|---|
| [architecture/overview.md](architecture/overview.md) | Layers, packages and key properties |
| [architecture/decisions/0001-stdlib-only-mcp-server.md](architecture/decisions/0001-stdlib-only-mcp-server.md) | ADR-0001: standard-library-only MCP server |
| [architecture/decisions/0002-trimble-connect-first.md](architecture/decisions/0002-trimble-connect-first.md) | ADR-0002: Trimble Connect first |
| [architecture/decisions/0003-dual-era-mcp.md](architecture/decisions/0003-dual-era-mcp.md) | ADR-0003: legacy and modern MCP revisions |
| [architecture/decisions/0004-read-only-first-and-approvals.md](architecture/decisions/0004-read-only-first-and-approvals.md) | ADR-0004: no mutations before approvals |
| [architecture/decisions/0005-typesafe-skill-evaluated-not-adopted.md](architecture/decisions/0005-typesafe-skill-evaluated-not-adopted.md) | ADR-0005: TypeSafe evaluated |
| [architecture/decisions/0006-trimble-connect-windows-command-line.md](architecture/decisions/0006-trimble-connect-windows-command-line.md) | ADR-0006: Trimble Connect for Windows command line |
| [architecture/decisions/0007-api-catalogue-coverage.md](architecture/decisions/0007-api-catalogue-coverage.md) | ADR-0007: catalogue coverage of every Connect API operation |

## Security and privacy

| Document | What it covers |
|---|---|
| [security/threat-model.md](security/threat-model.md) | Threats, controls, residual risk and tests |
| [security/remote-auth.md](security/remote-auth.md) | Remote bearer tokens today; OAuth 2.1 plan |
| [security/privacy.md](security/privacy.md) | Data classification and handling |

## Requirements

| Document | What it covers |
|---|---|
| [requirements/traceability-matrix.md](requirements/traceability-matrix.md) | Every requirement with its implementation and test |
| [requirements/assumptions.md](requirements/assumptions.md) | Assumptions and how they are mitigated |
| [requirements/open-questions.md](requirements/open-questions.md) | Questions for the owner and for Trimble |
