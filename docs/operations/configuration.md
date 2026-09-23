# Configuration

All configuration comes from environment variables. Secrets are passed only as paths to private (0600) files. Defaults are the safe choice.

| Variable | Default | Meaning |
|---|---|---|
| `TRIMBLE_MCP_TENANT` | `local` | Tenant this process serves |
| `TRIMBLE_MCP_AUDIT_LOG` | `trimble-mcp-audit.jsonl` | Append-only audit file (created 0600) |
| `TRIMBLE_MCP_LOCAL_SUBJECT` | `local-operator` | stdio and CLI principal |
| `TRIMBLE_MCP_LOCAL_SCOPES` | capabilities, projects, and files read | Comma-separated; write and delete are rejected |
| `TRIMBLE_MCP_LOCAL_PROJECTS` | (all) | Comma-separated project grant |
| `TRIMBLE_MCP_ENABLE_MOCK` | `true` | Simulated adapter |
| `TRIMBLE_CONNECT_ENABLED` | `false` | Trimble Connect adapter; `false` is the kill switch |
| `TRIMBLE_CONNECT_ENV` | `stage` | `stage` or `production` |
| `TRIMBLE_CONNECT_REGION` | `us` | `us`, `eu`, `eu-gb`, `ap`, `ap-au` (staging: `us`, `eu`, `ap`) |
| `TRIMBLE_IDENTITY_ISSUER` | `https://id.trimble.com` | Must be an allowed issuer |
| `TRIMBLE_CLIENT_ID` | — | Developer Console application |
| `TRIMBLE_CLIENT_SECRET_FILE` | — | Path to the client secret, if the app is confidential |
| `TRIMBLE_REDIRECT_URI` | `http://127.0.0.1:8765/callback` | Must be registered with Trimble |
| `TRIMBLE_SCOPE` | — | `openid <application scope>` |
| `TRIMBLE_TOKEN_STORE` | — | Encrypted token store path |
| `TRIMBLE_TOKEN_KEY_FILE` | — | 64-hex-character key file (`openssl rand -hex 32 > key && chmod 600 key`) |
| `TRIMBLE_ACCESS_TOKEN_FILE` | — | Sandbox only: a short-lived access token, never refreshed |
| `TRIMBLE_MCP_HTTP_ADDR` | `127.0.0.1:8787` | HTTP listen address |
| `TRIMBLE_MCP_HTTP_TOKENS_FILE` | — | Required for HTTP; see `docs/security/remote-auth.md` |
| `TRIMBLE_MCP_ALLOWED_ORIGINS` | (none) | Exact Origin allowlist; any other Origin gets 403 |
| `TRIMBLE_MCP_ALLOWED_HOSTS` | (any) | Host header allowlist |
| `TRIMBLE_MCP_RESOURCE_METADATA_URL` | — | Advertised in the 401 challenge |
| `TRIMBLE_MCP_TLS_CERT_FILE` / `_KEY_FILE` | — | TLS for HTTP |
| `TRIMBLE_MCP_ALLOW_PLAIN_HTTP` | `false` | Allow plain HTTP only behind a TLS-terminating proxy |
