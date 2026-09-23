# Architecture overview

```
Agent client (Claude Code, Codex, Perplexity, Claude app, trimblectl)
      │  stdio (local operator principal)  or  HTTPS /mcp (bearer → principal)
      ▼
internal/mcp       JSON-RPC 2.0, legacy + modern revisions, Origin/Host/auth/session checks
      ▼
internal/gateway   tool registry · strict input decode · authz · rate limit · audit (fail-closed)
      │            output envelope · untrusted-text cleaning · capability gating
      ▼
internal/trimble   capability interfaces (ProjectReader, FileReader) · Descriptor
      ├── trimble/connect   Trimble Connect REST (read-only, verified endpoints)
      └── trimble/mock      simulated adapter for CI and evals
internal/identity  Trimble Identity OAuth (PKCE, refresh, revoke) · encrypted token store
internal/domain    typed IDs · pagination · CRS/units/positions · provenance · time
internal/authz     scopes · principal · tenant/project/product decisions
internal/audit     append-only, hash-chained JSONL
```

## Layer mapping

| Brief layer | Package |
|---|---|
| 1. Agent client | external (see `connectors/`) |
| 2. Skill or connector | `skills/canonical/trimble`, `.claude/skills`, `.agents/skills` |
| 3. MCP gateway | `internal/mcp`, `internal/gateway` |
| 4. Application services | `internal/gateway` tool handlers; add services when mutations arrive |
| 5. Canonical domain | `internal/domain` |
| 6. Trimble adapters | `internal/trimble/*` |
| 7. Policy and approval | `internal/authz` (approvals planned, ADR-0004) |
| 8. Persistence | `internal/identity` token store, `internal/audit`; no database yet |
| 9. Observability | structured JSON logs on stderr; metrics planned |

## Key properties

- **Tenant isolation:** the registry keys adapters by tenant, and a principal can only resolve adapters in its own tenant.
- **Tools follow capabilities:** a tool is listed only if the caller has its scope and some adapter declares the capability.
- **IDOR:** every folder or file call names a project, and adapters verify ownership from upstream data.
- **Fail closed:** unknown fields, invalid IDs, malformed upstream data, a missing CRS, and audit failure all produce errors, never partial guesses.
