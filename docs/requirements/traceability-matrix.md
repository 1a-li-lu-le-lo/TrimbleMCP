# Requirement traceability matrix

Status values: **Done** (implemented and tested), **Partial**, **Planned**, **Blocked**.

Evidence refers to test names; `go test ./...` runs them all.

| ID | Description | Implementation | Test / evidence | Status |
|---|---|---|---|---|
| TRM-FR-001 | Report configured products, versions, scopes, health | `gateway/tools.go` `getCapabilities` | `TestCapabilities` | Done |
| TRM-FR-002 | List projects one page at a time, with an explicit cursor | `gateway/tools.go` `listProjects` | `TestListProjectsPaginatesToCompletion` | Done |
| TRM-FR-003 | List folder items within an authorized project | `listFolderItems` | `TestFolderItemsEnforceProject`, `TestIDORAcrossProjects` | Done |
| TRM-FR-004 | Read file metadata within an authorized project | `getFileMetadata` | `TestFileMetadata` | Done |
| TRM-FR-005 | Get a single project (only where verified) | `getProject` (mock only) | `TestDescriptorDoesNotClaimGetProject` | Done |
| TRM-FR-006 | Upload, version, download, delete | none | n/a | Planned (needs approvals) |
| TRM-CLI-001 | Build documented Trimble Connect for Windows `trimbleconnect:` links | `trimble/desktop.BuildURI` | `TestBuildURIDocumentedExample`, `TestBuildURIVariants` | Done |
| TRM-CLI-002 | Only documented views and panels, always in the two-value form (default 3D,models); case-insensitive input, documented spelling out | `desktop.canonical` | `TestBuildURIVariants`, `TestBuildURIRejectsInjectionAndUndocumentedValues` | Done |
| TRM-CLI-003 | URI contains only safe characters (no injection into the protocol handler) | `desktop.projectID` regexp | `FuzzBuildURIOnlyEmitsSafeURIs` | Done |
| TRM-CLI-004 | Project ID verified via the Connect API; fabricated IDs rejected | `gateway.verifyProject` | `TestDesktopLinkBuiltAndVerified`, `TestDesktopLinkUnknownProjectRejected`, `TestDesktopLaunchUnverifiedRefused` | Done |
| TRM-CLI-005 | Launch is opt-in, Windows-only, local-operator-only, dry run by default, reason required, rate-limited, audited | `desktop.Launch`, `gateway.openInDesktop` | `TestLaunchGating`, `TestDesktopLaunchVisibility`, `TestDesktopLaunchDryRunThenLaunch`, `TestDesktopLaunchRefusals`, `TestDesktopLaunchRateLimited` | Done |
| TRM-CLI-006 | Launch via documented ShellExecuteW, never a shell | `desktop/open_windows.go` | `GOOS=windows go vet` and build; manual check on Windows pending | Partial |
| TRM-CLI-007 | Remote callers cannot hold `trimble:desktop:launch` | `cmd/trimble-mcp` tokens loading | code review | Done |
| TRM-CLI-008 | CLI parity (`trimblectl desktop link/open`) | `cmd/trimblectl` | `scripts/smoke.sh` | Done |
| TRM-API-001 | Use only documented Trimble Connect endpoints and fields | `trimble/connect` | `docs/trimble-products/trimble-connect.md`, contract tests | Done |
| TRM-API-002 | Regional base URLs from the documented host list only | `connect.BaseURL` | `TestBaseURLOnlyDocumentedHosts`, `TestOverrideRequiresExplicitTestFlag` | Done |
| TRM-API-003 | Retry per the Connect error table; never retry 401 | `connect.get`, `mapStatus` | `TestRetryOn429ThenSuccess`, `TestNoRetryOn401AndBodyStaysPrivate`, `TestRetriesExhaustedOn503` | Done |
| TRM-API-004 | Treat malformed responses as errors, never guess | `connect` | `TestMissingItemsIsMalformed`, `TestFolderItemWithoutProjectFailsClosed`, `TestOversizedBodyRejected` | Done |
| TRM-API-005 | Client-side rate limiting per adapter | `ratelimit`, `connect.waitLimiter` | `TestKeyedBuckets` | Done |
| TRM-API-006 | Re-verify full OpenAPI (`/projects/{id}`, `/users/me`) | n/a | open question Q-2 | Blocked |
| TRM-MCP-001 | Legacy MCP lifecycle (2025-03-26 to 2025-11-25) | `mcp/server.go` | `TestStdioLegacyLifecycle`, `TestLegacyVersionNegotiation` | Done |
| TRM-MCP-002 | Modern stateless MCP (2026-07-28), including `server/discover` | `mcp/server.go` | `TestStdioModernStateless`, `TestHTTPModernHeaders` | Partial (no SSE, no MRTR) |
| TRM-MCP-003 | Streamable HTTP: Origin, Host, auth, sessions, 405 for GET | `mcp/http.go` | `TestHTTPOriginAndMethod`, `TestHTTPLegacySessionBinding`, `TestHTTPAuthChallenge` | Done |
| TRM-MCP-004 | Tools listed only when scoped and supported | `gateway.visible` | `TestToolListFilteredByScope`, `TestToolListFilteredByCapability` | Done |
| TRM-MCP-005 | Strict input schemas; unknown fields rejected | `gateway.decode` | `TestStrictInputValidation` | Done |
| TRM-MCP-006 | Structured output envelope, with text mirror and `outputSchema` | `gateway.Envelope` | `invoke` helper checks text/structured parity | Done |
| TRM-MCP-007 | Resources and prompts enforce the same authorization | `gateway/resources.go` | `TestResourcesAndPrompts` | Done |
| TRM-MCP-008 | No generic HTTP, shell, SQL, filesystem, or mutation tools | `gateway/tools.go` | `TestNoMutationToolsExist` | Done |
| TRM-SKILL-001 | Canonical SKILL.md with required sections and limits | `skills/canonical/trimble` | `TestFrontmatterAndLimits` | Done |
| TRM-SKILL-002 | Claude Code and Codex adapters identical to canonical | `.claude/skills`, `.agents/skills` | `TestAdapterParity` | Done |
| TRM-SKILL-003 | Skill references only tools the server exposes | skill files | `TestReferencedToolsExist` | Done |
| TRM-SKILL-004 | Activation, safety, and geospatial eval sets | `evals/*.yaml` | `TestActivationEvals`, `TestSafetyAndGeoEvalsParse` | Partial (no model-graded run) |
| TRM-GEO-001 | CRS explicit, including axis order | `domain.CRS` | `TestCRSMustBeExplicit` | Done |
| TRM-GEO-002 | International and US survey feet distinct and exact | `domain.Distance` | `TestSurveyFootIsNotInternationalFoot` | Done |
| TRM-GEO-003 | Positions validated; height reference and units required | `domain.GeographicPosition` | `TestGeographicPosition` | Done |
| TRM-GEO-004 | No implicit coordinate transformation | no transform tool exists | `TestNoMutationToolsExist` (tool set) | Done |
| TRM-FILE-001 | Size in bytes; timestamps normalised to UTC | `connect.parseTime` | `TestListProjectsPaginationAndAuth`, `TestFileMetadata` | Done |
| TRM-FILE-002 | Streaming upload and download safety | none | n/a | Planned |
| TRM-SEC-001 | Cross-tenant isolation by construction | `gateway.Registry` | `TestCrossTenantIsolation`, `TestAuthorize` | Done |
| TRM-SEC-002 | IDOR: folder and file bound to the authorized project | adapters | `TestIDORAcrossProjects`, `TestFolderItemsEnforceProject` | Done |
| TRM-SEC-003 | IDs validated against path and query injection | `domain.validateID` | `TestIDValidation`, `FuzzIDNeverAllowsPathOrQueryBreakout` | Done |
| TRM-SEC-004 | Secrets never logged, printed, or returned | `identity.Secret`, `errs` | `TestSecretNeverPrints`, `TestTokenErrorDoesNotEchoBody`, `TestUpstreamFaultsMapToSafeErrors` | Done |
| TRM-SEC-005 | Refresh tokens encrypted at rest; private file permissions enforced | `identity.FileStore` | `TestFileStoreRoundTripAndPermissions` | Done |
| TRM-SEC-006 | PKCE S256 with state check | `identity`, `trimblectl auth login` | `TestPKCEChallengeRFC7636Vector`, `TestNewClientValidation` | Done |
| TRM-SEC-012 | Trimble Serial PKCE: new challenge on every token and refresh request; previous verifier presented | `identity.Exchange`, `identity.Refresh` | `TestSerialPKCEExchangeAndRefresh` | Done |
| TRM-SEC-007 | HTTP bearer tokens stored only as SHA-256; constant-time compare | `mcp.StaticTokenAuthenticator` | `TestHTTPAuthChallenge` | Done |
| TRM-SEC-008 | Prompt-injection text neutralised and labelled | `gateway.cleanUntrusted` | `TestPromptInjectionNamesAreNeutralisedAndLabelled` | Done |
| TRM-SEC-009 | Session IDs bound to subject and tenant | `mcp/http.go` `lookup` | `TestHTTPLegacySessionBinding` | Done |
| TRM-SEC-010 | Rate limits per caller (gateway and HTTP) | `ratelimit` | `TestGatewayRateLimit`, `TestHTTPRateLimit` | Done |
| TRM-SEC-011 | OAuth 2.1 resource server for remote clients | none | n/a | Blocked (needs an authorization server) |
| TRM-PRIV-001 | Project grant filtering reveals no restricted names | `listProjects` | `TestProjectGrantFiltersAndDenies` | Done |
| TRM-PRIV-002 | Audit stores hashes, not payloads | `audit` | `TestChainAndTamperDetection` | Done |
| TRM-OPS-001 | Append-only, hash-chained audit; tamper detection | `audit` | `TestChainAndTamperDetection` | Done |
| TRM-OPS-002 | Fail closed when audit cannot be written | `gateway.CallTool` | `TestAuditFailureFailsClosed`, `TestAuditRequired` | Done |
| TRM-OPS-003 | Per-product kill switch | `Registry.Disable` | `TestKillSwitch` | Done |
| TRM-OPS-004 | Metrics, traces, alerts | structured logs only | n/a | Planned |
| TRM-TEST-001 | CI without production credentials | mock adapter, httptest | `.github/workflows/ci.yml` | Done |
| TRM-TEST-002 | Fuzzing of IDs and MCP input | `domain`, `mcp` | `FuzzIDNeverAllowsPathOrQueryBreakout`, `FuzzStdioNeverPanicsAndEmitsOnlyJSON` | Done |
| TRM-DOC-001 | Capability matrix with sources | `docs/trimble-products/` | review | Done |
| TRM-DOC-002 | Connector setup for Claude Code, Codex, Perplexity, Claude app | `connectors/` | review | Done (unverified against live clients) |
