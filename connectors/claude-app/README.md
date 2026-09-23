# Claude application (claude.ai, Claude Desktop)

Sources (checked 2026-09-23):

- https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp
- https://claude.com/docs/connectors/custom/remote-mcp
- https://claude.com/docs/connectors/building/authentication

## Requirements

- A remote MCP server over HTTPS, reachable from Anthropic's egress range (`160.79.104.0/21`).
- Plans: Free (one custom connector), Pro, Max, Team, Enterprise. On Team and Enterprise, an owner adds the connector under Organization settings > Connectors.
- Auth, as supported by Claude:
  - OAuth (CIMD, DCR, or your own client ID).
  - **Static request headers (beta, up to 4).**
  - None.

## Setup with this release

1. Deploy `trimble-mcp -transport http` behind TLS, and set `TRIMBLE_MCP_ALLOWED_HOSTS` to your hostname.
2. Create a caller token with `trimblectl token new`, and give it read-only scopes and a project list in the tokens file.
3. In Claude, go to Customize > Connectors > Add custom connector:
   - **URL:** `https://<host>/mcp`
   - **Request headers (beta):** `Authorization` = `Bearer <token>`
   - Auth settings cannot be edited after adding, so recreate the connector to rotate the token.
4. In the connector's tool settings:
   - Leave every tool on per-call approval, or "Always allow" only `trimble_get_capabilities`.
   - Block any tool you do not need.

All tools declare `readOnlyHint: true` and `destructiveHint: false`. There are no upload, folder, version, view, delete, webhook, or bulk-export tools. When mutation tools are added, each must require explicit confirmation plus a server-issued approval reference.

## OAuth (production)

Claude's OAuth flow requires the bridge to return 401 with `resource_metadata`. That metadata's `resource` must match the connector URL exactly, and the authorization server must support PKCE S256. The bridge already emits `WWW-Authenticate: Bearer resource_metadata=...` when `TRIMBLE_MCP_RESOURCE_METADATA_URL` is set, but token validation against an authorization server is not implemented. See `docs/security/remote-auth.md`.

## Revoke and disconnect

1. Disconnect or remove the connector in Claude settings.
2. Delete the token's hash from the tokens file and restart the server.
3. Review `trimble-mcp-audit.jsonl` for that subject, and run `trimblectl audit verify --file trimble-mcp-audit.jsonl`.
