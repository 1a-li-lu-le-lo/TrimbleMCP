# ADR-0003: Support legacy and modern MCP revisions

- **Status:** Accepted, 2026-09-23.
- **Context:** MCP 2026-07-28 removed `initialize`, sessions, and `ping` in favour of per-request `_meta` and `server/discover`. Deployed clients still use the 2025-xx handshake.
- **Decision:** Choose the mode per request. A request carrying `_meta["io.modelcontextprotocol/protocolVersion"]` is served statelessly. `initialize` starts a legacy session (over HTTP, bound to subject and tenant). Server-sent SSE streams and MRTR are not implemented; nothing needs them yet.
- **Consequences:** Two code paths share one dispatcher. Modern support is marked Partial until it is verified against a real modern client.
