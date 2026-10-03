# Remote authentication

## Current (pilot)

The HTTP transport accepts `Authorization: Bearer <token>`. Tokens are generated with `trimblectl token new`, and only their SHA-256 hashes are configured, in `TRIMBLE_MCP_HTTP_TOKENS_FILE`. That file is a JSON array, and must be mode 0600:

```json
[{"sha256": "<64 hex>", "subject": "alice@example.com", "client": "claude-app",
  "scopes": ["trimble:capabilities:read", "trimble:projects:read", "trimble:files:read"],
  "projects": ["<project id>"], "products": ["trimble-connect"]}]
```

Other rules:

- Tokens in query strings are never accepted.
- Write and delete scopes are rejected.
- Revoke a token by deleting its entry and restarting.

## Production (blocked: open question Q-5)

The MCP authorization spec makes a remote server an OAuth 2.1 resource server. The remaining work:

1. Publish RFC 9728 metadata at `/.well-known/oauth-protected-resource`, with `resource` equal to the public MCP URL and `authorization_servers` set to the chosen IdP.
2. Validate JWT access tokens: issuer, signature via JWKS, `exp` and `nbf`, and audience equal to this server (RFC 8707). Map token scopes to bridge scopes.
3. Return 401 with `WWW-Authenticate: Bearer resource_metadata="..."`, and 403 with `error="insufficient_scope"`.
4. Never pass client tokens through to Trimble. Trimble Identity tokens stay server-side, per tenant.

`TRIMBLE_MCP_RESOURCE_METADATA_URL` already makes the 401 challenge advertise the metadata URL.
