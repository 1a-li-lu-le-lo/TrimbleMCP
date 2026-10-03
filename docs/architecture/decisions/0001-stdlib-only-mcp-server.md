# ADR-0001: Standard-library-only MCP server

- **Status:** Accepted, 2026-09-23.
- **Context:** The build environment could not fetch Go modules; a module-proxy request was denied by the sandbox policy. Supply-chain risk is also a listed threat.
- **Decision:** Implement JSON-RPC 2.0, the stdio transport, and the Streamable HTTP transport with the Go standard library only. `go.mod` has no dependencies.
- **Consequences:**
  - The dependency surface is zero, and CI needs nothing but the Go toolchain.
  - Spec conformance is our responsibility and is covered by `internal/mcp` tests.
  - The official Go SDK (`github.com/modelcontextprotocol/go-sdk`) can replace `internal/mcp` later behind the same `mcp.Provider` interface.
