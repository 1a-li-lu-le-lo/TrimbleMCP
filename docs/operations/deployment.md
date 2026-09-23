# Deployment and rollback

## Local (stdio)

```sh
go install ./cmd/...
trimblectl diagnostics
trimble-mcp -transport stdio      # normally launched by the MCP client
```

## Sandbox Trimble Connect

```sh
openssl rand -hex 32 > ~/.trimble/key && chmod 600 ~/.trimble/key
export TRIMBLE_CONNECT_ENABLED=true TRIMBLE_CONNECT_ENV=stage TRIMBLE_CONNECT_REGION=us
export TRIMBLE_CLIENT_ID=<id> TRIMBLE_SCOPE="openid <app-scope>"
export TRIMBLE_TOKEN_STORE=~/.trimble/tc-token TRIMBLE_TOKEN_KEY_FILE=~/.trimble/key
trimblectl auth login
trimblectl projects list --product trimble-connect
```

## Remote (HTTPS)

Build `deploy/docker/Dockerfile`, which produces a static binary on distroless nonroot. Then run it with:

- TLS configured, or `TRIMBLE_MCP_ALLOW_PLAIN_HTTP=true` behind a TLS proxy;
- `TRIMBLE_MCP_HTTP_TOKENS_FILE`, `TRIMBLE_MCP_ALLOWED_HOSTS` and `TRIMBLE_MCP_ALLOWED_ORIGINS` set;
- the audit log on a persistent volume.

Health check: `GET /healthz`. The process shuts down gracefully on SIGTERM, with a 20 s drain.

## Rollback

- **Kill switch:** set `TRIMBLE_CONNECT_ENABLED=false` and restart. The Connect tools disappear and the mock stays available.
- **Revoke access:**
  - Remote callers: delete tokens-file entries and restart.
  - Upstream: run `trimblectl auth logout`.
- **Version rollback:** redeploy the previous image tag. There are no data migrations; the audit log format is append-only and backwards compatible.
- **Evidence:** run `trimblectl audit verify --file <audit log>` before and after any incident action.
