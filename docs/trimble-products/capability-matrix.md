# Trimble capability matrix

Last verified: 2026-09-23 (Trimble Connect and Identity); 2026-09-29 (every other product). Confidence labels:

- **F:** fetched from an official page during verification.
- **S:** seen only in a search excerpt of an official page, or in a third-party page.
- **Unknown:** no evidence.

## Coverage at a glance

The bridge accounts for every Trimble API operation that has a public machine-readable definition: **7,016 operations in 230 definitions**. Each has exactly one disposition; the per-operation lists are in [endpoints/README.md](endpoints/README.md), and ADR-0007 and ADR-0008 explain the approach.

| Scope | How it is covered | Disposition |
|---|---|---|
| Trimble Connect: 11 production APIs and every staging, integration, QA, test, draft and internal copy (34 SwaggerHub definitions) | Executable reads against the documented regional hosts; changes as dry-run plans | `read`, `plan`, `variant`, `excluded` |
| 14 other Trimble products and services with a public definition (24 APIs, 190 definitions) | Searchable and plannable as a dry run; never called, because the bridge has no credentials or host for them | `reference` (with 6 safety exclusions) |
| Trimble Identity | Every endpoint in the OpenID Connect discovery document | `excluded` (used only by the bridge's own sign-in) |
| Publicly reachable definitions that Trimble does not link, and placeholders | Catalogued, never used | `excluded` |
| Trimble Connect services named by `/regions` with no definition | Listed with a note | not callable |
| Products with no machine-readable definition (SDKs, desktop APIs, prose-only or Postman-only APIs) | Classified below, with the reason | not applicable |
| Command-line tools (Trimble Connect for Windows, its installer, the App Xchange `xchange` CLI, `teklaenv`, Tekla Structures and SketchUp start-up switches) | Every documented command or switch; see [cli-coverage.md](cli-coverage.md) | supported or excluded |

## Adopted API families

### TC-CORE: Trimble Connect REST API (Core)

| Field | Value |
|---|---|
| API FAMILY ID | TC-CORE |
| PRODUCT | Trimble Connect |
| BUSINESS DOMAIN | Construction project collaboration: projects, folders, files, BIM coordination |
| OFFICIAL NAME | "Trimble Connect API" (OpenAPI `info.title`), called the Core API on the portal (F) |
| OFFICIAL DOCUMENTATION | https://developer.trimble.com/docs/connect , https://developer.trimble.com/docs/connect/reference (F) |
| DEVELOPER PORTAL | https://developer.trimble.com/ ; app registration in the Trimble Developer Console, https://console.developer.trimble.com/ (S) |
| BASE URL | Regional: `https://{app,app21,app22,app31,app32}.connect.trimble.com/tc/api` for us, eu, eu-gb, ap, ap-au (F, from `/regions`). Staging: `app.stage`, `app21.stage`, `app31.stage` (F, from OpenAPI `servers`) |
| API VERSION | 2.0, with 2.1 paths for newer collection endpoints (F) |
| OPENAPI AVAILABLE | Yes: https://api.swaggerhub.com/apis/Trimble-Connect/tcps/2.0 (F; complete, 113 paths, re-checked 2026-09-23). Also https://developer.trimble.com/docs/connect/reference/openapi/core |
| SDK AVAILABLE | Workspace API (browser JS) and a Windows .NET API; no Go SDK (F) |
| SUPPORTED LANGUAGES | Any HTTP client; the bridge uses a hand-written Go adapter |
| PUBLIC ACCESS | Documentation is public. Credentials are not self-service (F) |
| PARTNER ACCESS | Commercial integrations go through the Trimble Marketplace Partner programme, which grants sandbox and API credentials (F) |
| ENTERPRISE ACCESS | In-house use needs a paid Connect licence and a corporate-domain Trimble ID; credentials come via a request form. Personal subscribers are not eligible (F) |
| SANDBOX | Staging API hosts exist (F). The staging identity host is referenced but could not be resolved (Unknown) |
| AUTHENTICATION | Trimble Identity OAuth 2.0 authorization code, with PKCE recommended. **Client credentials not supported by Connect.** Bearer header. Refresh at least every 9 days (F) |
| AUTHORIZATION | Per Trimble user and project membership, enforced upstream. The bridge adds scope, tenant, and project grants |
| ACCOUNT MODEL | Trimble ID user accounts (F) |
| TENANT MODEL | The bridge binds one tenant to one Trimble Identity session and one region |
| ORGANIZATION MODEL | Core Account API exists (F); not used |
| PROJECT MODEL | Projects live in one region; resources are independent per region (F) |
| RESOURCE TYPES | Projects, folders, files, versions, users, activities, and more (F) |
| READ OPERATIONS (used) | `GET /2.1/projects`, `GET /2.0/projects/{projectId}`, `GET /2.1/folders/{folderId}/items?fields=size`, `GET /2.0/files/{fileId}` (F) |
| READ OPERATIONS (not used) | `GET /projects/me`, file versions, and the download URL: documented, deferred |
| WRITE OPERATIONS | `POST /files/fs/initiate`, `POST /files/fs/commit`, package upload (F). **Not implemented** |
| DELETE OPERATIONS | Unknown (not reviewed). Not implemented |
| BULK OPERATIONS | batch-api host listed in `/regions` (F). Not used |
| ASYNC OPERATIONS | Upload processing (`tc-enable-process`), upload status (F). Not used |
| WEBHOOKS | None documented for the REST API. `/activities` could support polling (F) |
| PAGINATION | v2.1: `pageSize` plus `skipToken`, next page via `links.next.href` (F). v2.0: `Range: items=a-b` with 206 and `Content-Range` (F) |
| FILTERING / SORTING | `fields`, `objectTypes` (FOLDER/FILE), `sortBy`/`sort` (F) |
| RATE LIMITS | Not published. 429 is documented as retryable (F). The bridge limits itself to 5 req/s by default |
| IDEMPOTENCY | No idempotency keys. `If-Match`/ETag for concurrency on writes and downloads (F) |
| REQUEST LIMITS | Names up to 255 characters; tags up to 40 (F) |
| FILE LIMITS | Unknown |
| DATA FORMATS | JSON. Download formats: TRB, PDF, THUMBNAIL, GEOJSON, USD, USDZ (F) |
| COORDINATE SYSTEMS | Not applicable to the implemented operations |
| UNITS | File `size` is assumed to be bytes; that is the conventional meaning, but the unit is not stated in the retrieved spec (assumption A-3) |
| TIME ZONES | Timestamp strings. The bridge parses RFC 3339 and normalises to UTC; unparseable values are omitted, not guessed |
| ERROR MODEL | `{"error": "...", "code": "..."}`. Retry guidance per status: 400/401/403/404/409/422 do not retry; 429/500/503 retry; 502/504 retry only if idempotent (F) |
| SERVICE LEVEL | Unknown |
| REGIONAL RESTRICTIONS | Region-bound data (F) |
| PRICING OR COMMERCIAL TERMS | Paid licence or partner agreement (F); pricing Unknown |
| DATA RETENTION | Unknown |
| LICENSE | Connect Terms: https://www.trimble.com/en/products/trimble-connect/terms-and-conditions ; SDK Internal Use License: https://developer.trimble.com/docs/connect/terms/SDK-IU-license-agreement (F) |
| LAST VERIFIED | 2026-09-23 |
| CONFIDENCE | Medium-high. Every endpoint and field used is in the complete OpenAPI definition (F); not yet exercised against a sandbox |
| ADOPTION DECISION | **Adopt: first read-only adapter (class A/B/C: public docs, gated credentials)** |
| BLOCKERS | Need sandbox credentials and a registered callback URL (request from connect-support@trimble.com). Re-verify the full spec for `/projects/{id}` and `/users/me` |

### TID: Trimble Identity

| Field | Value |
|---|---|
| OFFICIAL DOCUMENTATION | https://developer.trimble.com/docs/authentication , https://id.trimble.com/.well-known/openid-configuration (F) |
| ENDPOINTS | issuer `https://id.trimble.com`; `/oauth/authorize`, `/oauth/token`, `/oauth/userinfo`, `/oauth/revoke`, `/oauth/logout`, JWKS `/.well-known/jwks.json` (F) |
| GRANTS | authorization_code, refresh_token, client_credentials, token-exchange, device_code, and others (F). Connect rejects client_credentials (F) |
| PKCE | S256 (F) |
| CLIENT AUTH | client_secret_basic, client_secret_post, tls_client_auth (F) |
| SCOPES | `openid` plus the application scope from the Developer Console (F) |
| REVOCATION | Basic client auth; `token` and `token_type_hint` (F) |
| ADOPTION DECISION | Adopt for TC-CORE |

### TC-WIN-CLI: Trimble Connect for Windows command line

| Field | Value |
|---|---|
| OFFICIAL DOCUMENTATION | https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line (F) |
| INTERFACE | URL scheme `"trimbleconnect:/projects/[project-id]?show=[view parameter],[panel parameter]"`, one slash after the colon. It can be started from a command line or a browser (F) |
| VALUES | Views: projects, data, 3D. Panels: clashes, models, objects, ToDos, views. Case-insensitive (F) |
| EXAMPLE | `trimbleconnect:/projects/rQR1yhTGj9I?show=3D,ToDos` opens the ToDos tab "instead of the models tab" (F) |
| PLATFORM | Windows, where Trimble Connect for Windows registers the protocol |
| DATA ACCESS | None; it only navigates the desktop UI |
| UNDOCUMENTED | Single-value `show`, omitted `show`, behaviour when the app is not installed or not signed in, supported versions, and whether the project ID equals the REST API ID |
| OTHER COMMAND LINES | Enterprise extraction `TrimbleConnectSetup-VersionNumber-x64.exe /ad:\preq` (F; changes the system, excluded). Connect Sync: no documented CLI (F). No official Connect CLI or PowerShell module found |
| RELATED | Windows .NET API (C#, in-process; docs only in the local CHM) (F). Web viewer path `https://web.connect.trimble.com/projects/:projectId/viewer/3d` (F, Workspace API docs); not used |
| ADOPTION DECISION | **Adopt (provisional)**: link builder plus opt-in local launch (ADR-0006) |

### Identity update (2026-09-23)

Trimble Identity requires **Serial PKCE**: "a new (`code_verifier`, `code_challenge`) pair must be provided on each token or token refresh request" (F: https://developer.trimble.com/docs/authentication/guides/authorization-code-pkce/). Implemented in `internal/identity`.

## Product-to-capability classification

Classes:

- **A:** public API
- **B:** partner API
- **C:** enterprise or authorized API
- **D:** SDK only
- **E:** export/import
- **F:** manual
- **G:** unsupported

| Product | Class | Evidence | Decision |
|---|---|---|---|
| Trimble Connect Core REST | A/C (public docs, licensed credentials) | F | **Adopted (read-only)** |
| Trimble Connect Model, Model Feature, Org (Account/Organizer), Property Set, Topics (BCF), Topics Exchange, Issues, Support, Drive (beta), File Service (preview) | A/C | F (official SwaggerHub definitions) | **Adopted via the catalogue** (reads executable, changes planned); see [endpoints/README.md](endpoints/README.md) |
| Trimble Connect internal API (`tcps.internal`) | G | F | Excluded: Trimble-internal, not offered to integrators |
| Trimble Connect regional services without a definition (`batch-api`, `objects-sync-api`, `projects-api`, `user-api`, `wopi-api`) | G (no definition) | F (`/regions`) | Listed in [endpoints/README.md](endpoints/README.md); not callable |
| Trimble Connect Workspace API | D (browser JS, npm `trimble-connect-workspace-api`) | F | Not applicable to a server |
| Trimble Connect Windows .NET API | D (in-process C#) | F | Not applicable to a server |
| Trimble Connect for Windows command line (`trimbleconnect:`) | D/F (documented local launcher) | F | **Adopted (link builder plus opt-in local launch)** |
| Connect for Windows installer and MSI command lines | G for this bridge (host changes) | F | Never |
| Trimble Connect Sync | F (GUI only; no documented CLI) | F | Not implemented |
| Trimble Identity | A (OpenID Connect provider) | F | Used by the bridge's own sign-in; every endpoint catalogued and `excluded` for agents |
| All other products with a public definition | see the next section | F | **Catalogued as `reference`** |

## Other Trimble products

### With a public machine-readable definition (catalogued as `reference`)

Discovery method: every developer.trimble.com product section's sitemap was crawled for pages that embed an OpenAPI viewer (24 sections), plus the definitions that products publish directly and Vista's per-operation definitions. The discovery and classification are repeatable: `make catalog` reruns them and fails on anything new or unclassified (`cmd/trimble-catalog/sources-other.json`).

`reference` means that the operations are searchable with `trimble_api_operations`, and `trimble_api_plan` validates them into a dry-run request for a person or a separately authorised integration. The bridge never calls these products. It holds no credentials for them, and several are single-tenant or on-premises with a customer-specific host. Executing any of them would need a new adapter, credentials, and an ADR.

| Product | Family | API ids (endpoint pages) | Definitions | Operations | Authentication (from the definition or docs) | What calling it would need |
|---|---|---|---|---|---|---|
| Trimble Construction One: Vista (App Xchange Direct API) | construction | [`vista`](endpoints/vista.md) | 18 modules (per-operation fragments from https://direct-api.xchange.trimble.com) | 1,970 (17 are the same shared endpoint, marked `variant`) | `X-Application-Key` header; 2,000 requests per minute per key | Vista API application key and subscriber code |
| ProjectSight | construction | [`projectsight`](endpoints/projectsight.md) | 1 | 537 | OAuth 2.0 via Trimble Identity | ProjectSight subscription, a Trimble Identity application, and the API host (not in the definition) |
| Trimble Unity Construct (e-Builder) | construction | [`unity-construct`](endpoints/unity-construct.md) | 1 | 184 | OAuth 2.0 password grant (`/api/v2/Authenticate`) | e-Builder account; regional host |
| Trimble Unity Maintain / Permit (Cityworks) | construction | [`unity-maintain-permit`](endpoints/unity-maintain-permit.md) | 150 controller definitions | 1,223 | HTTP bearer | Tenant host (`https://{unityMpHost}/services`) and a token for it |
| Accubid Anywhere | construction | [`accubid-changeorder`](endpoints/accubid-changeorder.md), [`accubid-closeout`](endpoints/accubid-closeout.md), [`accubid-database`](endpoints/accubid-database.md), [`accubid-estimate`](endpoints/accubid-estimate.md), [`accubid-estimate-v2`](endpoints/accubid-estimate-v2.md), [`accubid-project`](endpoints/accubid-project.md), [`accubid-project-v2`](endpoints/accubid-project-v2.md) | 7 | 32 | Trimble Identity bearer; the user needs "API Data Access" | Entitlement and the service host (definitions give relative servers only) |
| Trimble Civil Site Management | construction | [`civil-site-management`](endpoints/civil-site-management.md) | 1 | 19 (reads only) | Not declared in the definition | Subscription and credentials for `cloud.api.trimble.com/site-management/v1` |
| Trimble Geospatial Field Configuration | geospatial | [`geospatial-field-configuration`](endpoints/geospatial-field-configuration.md) | 1 | 12 | OAuth 2.0 (Trimble Identity) | Token with the service scope |
| Trimble Geospatial Field Data (Jobs) | geospatial | [`geospatial-field-data`](endpoints/geospatial-field-data.md) | 1 | 29 | OAuth 2.0 (Trimble Identity) | Token with the service scope |
| Trimble Mobile Manager | geospatial | [`mobile-manager`](endpoints/mobile-manager.md) | 1 | 11 (4 configuration changes **safety-excluded**) | None declared; local device API | Running on the field device |
| TMT Fleet Maintenance (Integration Toolkit) | transportation | [`tmt`](endpoints/tmt.md) | 1 (Swagger 2.0) | 66 | HTTP basic or OAuth 2.0 password | Integration Toolkit licence and tenant host |
| TMWSuite | transportation | [`tmwsuite-cloudhub`](endpoints/tmwsuite-cloudhub.md), [`tmwsuite-odata`](endpoints/tmwsuite-odata.md), [`tmwsuite-ordercreate`](endpoints/tmwsuite-ordercreate.md) | 3 | 282 | OAuth 2.0 client credentials (Trimble Transportation identity) | Licence, Transportation Cloud credentials, connector host |
| TruckMate | transportation | [`truckmate`](endpoints/truckmate.md), [`truckmate-finance`](endpoints/truckmate-finance.md), [`truckmate-master-data`](endpoints/truckmate-master-data.md) | 3 | 596 | HTTP bearer (Trimble Identity JWT, web user or API key) | REST licence and the customer's TruckMate host |
| Trimble Maps Places API | maps | [`trimble-maps-places`](endpoints/trimble-maps-places.md) | 1 | 43 | API key | Trimble Maps API key |
| PTx Trimble FarmENGAGE Data API | agriculture | [`ptx-farmengage`](endpoints/ptx-farmengage.md) | 1 | 209 (2 that send data to in-cab vehicle devices **safety-excluded**) | Not declared; see the developer guide | PTx Trimble API credentials |

Plans for transportation and agriculture operations carry a warning that a qualified person must review and perform them.

### Publicly reachable but not documented (catalogued as `excluded`)

| Definition | Why it is excluded |
|---|---|
| Trimble Maps RouteReporter (`routereporterservice.trimblemaps.com`) | Not linked from any Trimble documentation. The bridge does not use undocumented endpoints |
| Trimble Maps Content API (beta) (`contentapi.trimblemaps.com`) | Same |
| Tekla support API `tsupport-v3-dev` (SwaggerHub org "Tekla") | A development support-ticket API not linked from Tekla documentation |
| SwaggerHub `trimble-analytics/trimble-identity` | An empty stub (0 paths); Identity is covered from the discovery document |
| App Xchange "Example API" (served from a personal GitHub Pages site) | A placeholder, not a Trimble API |

### Without a public machine-readable definition

These products were checked on 2026-09-29. None publishes an OpenAPI or Swagger definition that could be catalogued, so there are no operations to list; each is classified here instead.

| Product | What Trimble publishes | Class | Decision |
|---|---|---|---|
| Trimble Identity reference docs | Prose and a Postman collection; the endpoints are in the discovery document (catalogued) | A | Covered through the discovery document |
| PC*MILER Web Services | A WCF HTML help page (https://pcmiler.alk.com/apis/rest/v1.0/Service.svc/help), not OpenAPI | A (API key) | Not catalogued: no machine-readable definition. Candidate for a future adapter |
| Trimble Maps REST APIs other than Places | Prose documentation on developer.trimblemaps.com | A (API key) | Not catalogued, same reason |
| Trimble Maps Account Manager API | Its linked Swagger page returns HTTP 404 (checked 2026-09-29) | A | Not catalogued; re-check on the next refresh |
| CoPilot | SDK and CPIK libraries; the portal's example OpenAPI page has no definition | D | Not applicable |
| Trimble Maps SDKs (JavaScript, mobile) | SDK references | D | Not applicable |
| Spectrum | Help-site web-service docs (help.trimble.com) | C | Not catalogued: prose only |
| Jobpac Connect | Prose help | C | Not catalogued: prose only |
| TAP Store | A Postman collection only | C | Not catalogued: no OpenAPI |
| Trimble Access (Survey Core plug-in API) | Native SDK | D | Not applicable |
| Precision, Catalyst, Trimble Precision SDK | Native SDKs | D | Not applicable |
| SketchUp | Ruby API and C SDK; no REST API | D | Not applicable |
| Tekla Structures, Structural Designer, Tedds, PowerFab | .NET/COM "Open API" SDKs (the name does not mean the OpenAPI format) | D | Not applicable |
| Tekla Environment Service | The `teklaenv` CLI's API (`https://cloud.api.trimble.com/tekla/environments/v1`); no published definition | C | Not catalogued; the CLI is in [cli-coverage.md](cli-coverage.md) |
| App Xchange platform | SDK and the `xchange` CLI; only a placeholder definition | D | CLI in [cli-coverage.md](cli-coverage.md) |
| Trimble Unity webhooks (Action Manager) | Outbound callbacks, not an API the bridge calls | C | Deferred |
| Trimble Transportation / PeopleNet telematics | No public definition found; ownership reportedly changing | S | Deferred; re-verify ownership |
| GNSS / RTX corrections | Device correction streams | G for this bridge | Not implemented |
| Machinery or vehicle control of any kind | n/a | G (prohibited) | **Never** |

## Workflow matrix

| Workflow | Product | API | Entitlement | Read | Write | Personal data | Safety impact | Approval | Status |
|---|---|---|---|---|---|---|---|---|---|
| Discover capabilities | all | bridge | none | config | none | none | none | L0 | Done |
| List projects | Connect | `GET /2.1/projects` | Connect licence | names, IDs | none | project names | low | L0 | Done |
| Browse folders | Connect | `GET /2.1/folders/{id}/items` | Connect licence | names, IDs | none | file names | low | L0 | Done |
| File metadata | Connect | `GET /2.0/files/{id}` | Connect licence | metadata | none | none | low | L0 | Done |
| Download file | Connect | `GET /files/fs/{id}/downloadurl` | Connect licence | content | none | document content | medium | L1 | Not started |
| Upload new version | Connect | `/files/fs/initiate` + `commit` | Connect licence | none | file | none | medium | L3/L4 | Not started |
| Route or geocode | Trimble Maps | Places API catalogued as `reference`; PC*MILER has no definition | API key | n/a | n/a | locations | medium | L1 | Reference only |
| Find another product's operation and plan it | Vista, ProjectSight, Unity, TMWSuite, TruckMate and others | `trimble_api_operations`, `trimble_api_plan` | none (dry run) | definitions | none | none | low | L0 (plan); L1 to L4 to perform | Done (dry run only) |
| Vehicle position | fleet | Unknown | partner | n/a | n/a | driver location | high | L4 | Not verified |
