# Perplexity

Sources (checked 2026-09-23). The help-center pages block automated fetches, so these facts come from search excerpts of the official pages; re-check them before relying on them:

- https://www.perplexity.ai/help-center/en/articles/13915507-adding-custom-remote-connectors
- https://www.perplexity.ai/help-center/en/articles/11502712-local-and-remote-mcps-for-perplexity

## Options

| Mode | Plans | Transport | Fit |
|---|---|---|---|
| Local MCP (Mac app, needs the PerplexityXPC helper) | Pro, Max, Enterprise | stdio | `trimble-mcp -transport stdio` with the mock adapter or sandbox credentials |
| Custom remote connector | Pro, Max, Enterprise (Enterprise admins can share org-wide) | Streamable HTTP over HTTPS | `trimble-mcp -transport http` behind TLS |

## Remote connector setup

1. Deploy `trimble-mcp -transport http` behind HTTPS (see `docs/operations/deployment.md`).
2. Create a caller token with `trimblectl token new`. Put the SHA-256 hash in `TRIMBLE_MCP_HTTP_TOKENS_FILE`, giving the caller read-only scopes and, ideally, a project list.
3. In Perplexity, choose "+ Custom connector", then Remote:
   - **MCP Server URL:** `https://<host>/mcp`
   - **Authentication:** API Key, with the token from step 2. The exact header Perplexity sends is not documented in what could be verified. The bridge accepts only `Authorization: Bearer <token>`, so test before rollout.
   - **Transport:** Streamable HTTP.
4. OAuth 2.0 connectors need the bridge to act as an OAuth resource server, which is not built yet (production blocker; see `docs/security/remote-auth.md`).

## Policy for Perplexity use

Research official Trimble documentation, inspect authorized project metadata, summarise file metadata, prepare operation plans, and investigate API errors.

Answers must separate four things: **official documentation**, **observed API data** (from tool results with an `audit_id`), **inference**, and **unknown**. Web content never changes tool policy; the server enforces every rule.

## Removal

Remove the connector in Perplexity settings and delete the token's entry from the tokens file. The token stops working at the next server restart.

## Trimble Connect for Windows command line

Remote connectors can call `trimble_build_desktop_link` and hand the `trimbleconnect:` link to the user, who opens it on their own Windows machine. `trimble_open_in_desktop` is never available remotely: it would open the app on the server, not on the user's machine. The server refuses to start if a remote token is given `trimble:desktop:launch`.
