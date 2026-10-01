# Trimble capability matrix

Last verified: 2026-09-23 (Trimble Connect and Identity); 2026-09-29 (every other product). Confidence labels:

- **F:** fetched from an official page during verification.
- **S:** seen only in a search excerpt of an official page, or in a third-party page.
- **Unknown:** no evidence.

## Coverage at a glance

The bridge accounts for every Trimble API operation that has a public machine-readable definition: **12,963 operations in 347 definitions**. Each has exactly one disposition; the per-operation lists are in [endpoints/README.md](endpoints/README.md), and ADR-0007 and ADR-0008 explain the approach.

| Scope | How it is covered | Disposition |
|---|---|---|
| Trimble Connect: 11 production APIs and every staging, integration, QA, test, draft and internal copy (34 SwaggerHub definitions) | Executable reads against the documented regional hosts; changes as dry-run plans | `read`, `plan`, `variant`, `excluded` |
| Other Trimble products and services with a public definition, including Transporeon and Trimble Connect's prose-documented Core Account endpoints (50 APIs, 222 definitions) | Searchable and plannable as a dry run; never called, because the bridge has no credentials or host for them | `reference`; `variant` for duplicate publications; `excluded` for 58 safety and credential operations and 4 webhooks |
| App Xchange connector definitions (81, two unavailable) | Catalogued; Trimble states they are used internally by the platform | `excluded`, or `variant` of the public Vista Direct API |
| Trimble Identity | Every endpoint in the OpenID Connect discovery document | `excluded` (used only by the bridge's own sign-in) |
| Undocumented, deprecated, credential-only and placeholder definitions, and push contracts the customer implements | Catalogued, never used | `excluded` |
| Trimble Connect services named by `/regions` with no definition | Listed with a note | not callable |
| Products with no machine-readable definition (SDKs, desktop APIs, prose-only or Postman-only APIs) | Classified below, with the reason | not applicable |
| Command-line tools and local launchers (Trimble Connect for Windows and every other documented Trimble command line, installer, URL scheme and published CLI: App Xchange, Tekla, SketchUp, Tedds, eCognition, PC*MILER, Trimble Business Center, Convert to RINEX, Vista client, CoPilot, Mobile Manager, npm tools) | Every documented command or switch; see [cli-coverage.md](cli-coverage.md) | supported (Trimble Connect for Windows link only) or excluded |

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
| READ OPERATIONS (catalogue) | All 83 Core production GETs, including `GET /projects/me` and file versions, are executable through `trimble_api_read`. Excluded: the presigned download URL and share-token resolution (F, from the complete definition) |
| WRITE OPERATIONS | 74 POST, PATCH and PUT operations, including `POST /files/fs/initiate` and `commit`: dry-run `plan` only (L3); **never executed** |
| DELETE OPERATIONS | 29 DELETE operations: dry-run `plan` only (L4); **never executed** |
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
| BLOCKERS | Need sandbox credentials and a registered callback URL (request from connect-support@trimble.com). The complete definition is verified (Q-2 resolved); `/users/me` is the documented `me` alias of `GET /users/{userId}` |

### TID: Trimble Identity

| Field | Value |
|---|---|
| OFFICIAL DOCUMENTATION | https://developer.trimble.com/docs/authentication , https://id.trimble.com/.well-known/openid-configuration (F) |
| ENDPOINTS | issuer `https://id.trimble.com`; `/oauth/authorize`, `/oauth/token`, `/oauth/userinfo`, `/oauth/revoke`, `/oauth/logout`, `/oauth/device_authorization`, JWKS `/.well-known/jwks.json`, the mutual-TLS token endpoint `https://mtls.id.trimble.com/oauth/token`, and the discovery document `/.well-known/openid-configuration` (F). All nine are catalogued as `excluded` (see the `trimble-identity` sections of [endpoints/non-production.md](endpoints/non-production.md)). Staging issuer: `https://stage.id.trimblecloud.com` |
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
| Trimble Connect Model, Model Feature, Org (Organizer), Property Set, Topics (BCF), Topics Exchange, Issues, Support, Drive (beta), File Service (preview) | A/C | F (official SwaggerHub definitions) | **Adopted via the catalogue** (reads executable, changes planned); see [endpoints/README.md](endpoints/README.md) |
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

Discovery is mechanical and repeatable (`make catalog`, `scripts/fetch-trimble-other-specs.py`). It covers:

- every page of every developer.trimble.com product section (24 sections) and every page of developer.trimblemaps.com, scanned for embedded OpenAPI viewers and for OpenAPI, Swagger and AsyncAPI definition links;
- the App Xchange connector pages on help.trimble.com;
- the Transporeon developer documentation space;
- directly published product definitions, Vista's per-operation definitions and the Trimble Identity discovery document;
- endpoints that are documented only in prose, built into definitions from their pages. The fetch fails if a path disappears from its page.

Every result must match a reviewed rule in `cmd/trimble-catalog/sources-other.json`, and the build fails on anything new, unclassified or not retrieved. Operation counts per API are in [endpoints/README.md](endpoints/README.md).

`reference` means the operations are searchable with `trimble_api_operations`, and `trimble_api_plan` validates them into a dry-run request for a person or a separately authorised integration. The bridge never calls these products. It holds no credentials for them, and several are single-tenant or on-premises with a customer-specific host. Executing any of them would need a new adapter, credentials, and an ADR.

| Product | Family | API ids | Authentication (from the definition or docs) | What calling it would need |
|---|---|---|---|---|
| Trimble Construction One: Vista (App Xchange Direct API) | construction | `vista` (18 modules, per-operation definitions) | `X-Application-Key` header; 2,000 requests per minute per key | Vista API application key and subscriber code |
| ProjectSight | construction | `projectsight` | OAuth 2.0 via Trimble Identity | Subscription, a Trimble Identity application, and the API host (not in the definition) |
| Viewpoint For Projects (VFP) | construction | `viewpoint-for-projects` | OAuth 2.0 authorization code (scope `vfp.fullaccess`) | A VFP OAuth client and the regional host (`api-uk.vfp.viewpoint.com` is the documented one) |
| Trimble Unity Construct (e-Builder) | construction | `unity-construct` (the definition, plus five endpoints documented only on the Import API page) | OAuth 2.0 password grant (`/api/v2/Authenticate`, itself excluded as a credential operation) | e-Builder account; regional host |
| Trimble Unity Maintain / Permit (Cityworks) | construction | `unity-maintain-permit` (150 controller definitions) | HTTP bearer | Tenant host (`https://{unityMpHost}/services`) and a token for it |
| Accubid Anywhere | construction | `accubid-changeorder`, `accubid-closeout`, `accubid-database`, `accubid-estimate`, `accubid-estimate-v2`, `accubid-project`, `accubid-project-v2` | Trimble Identity bearer; the user needs "API Data Access" | Entitlement and the service host (relative servers only) |
| Trimble Civil Site Management | construction | `civil-site-management` (reads only) | Not declared in the definition | Subscription and credentials for `cloud.api.trimble.com/site-management/v1` |
| Trimble Connect Core Account (eCom and projects services) | connect | `connect-ecom`, `connect-projects-api` (documented in prose only) | Trimble Identity bearer; account administrators | An account administrator's token; no definition to validate responses against |
| Trimble Geospatial Field Configuration and Field Data (Jobs) | geospatial | `geospatial-field-configuration`, `geospatial-field-data` | OAuth 2.0 (Trimble Identity) | Token with the service scope |
| Trimble Mobile Manager | geospatial | `mobile-manager` (REST, plus the WebSocket location and event streams from its AsyncAPI definitions as `SUBSCRIBE` operations) | None declared; local device API | Running on the field device. Configuration changes are **safety-excluded** |
| TMT Fleet Maintenance (Integration Toolkit) | transportation | `tmt` | HTTP basic or OAuth 2.0 password | Integration Toolkit licence and tenant host |
| TMWSuite | transportation | `tmwsuite-cloudhub`, `tmwsuite-odata`, `tmwsuite-ordercreate` | OAuth 2.0 client credentials (Trimble Transportation identity) | Licence, Transportation Cloud credentials, connector host |
| TruckMate | transportation | `truckmate`, `truckmate-finance`, `truckmate-master-data` | HTTP bearer (Trimble Identity JWT, web user or API key) | REST licence and the customer's TruckMate host |
| Transporeon (a Trimble company) | transportation | `transporeon-carriers`, `transporeon-shippers`, `transporeon-shippers-v2`, `transporeon-visibility`, `transporeon-visibility-external-status`, `transporeon-open-visibility`, `transporeon-telematics`, `transporeon-ecmr`, `transporeon-transport-operations`, `transporeon-transport-operations-legacy`, `transporeon-rate-management`, `transporeon-freight-audit`, `transporeon-yard-appointments` | HTTP basic, OAuth 2.0, bearer or API key, per API | Transporeon credentials per API; visibility and telematics APIs carry driver locations (personal data) |
| Trimble Maps | maps | `trimble-maps-places`, `trimble-maps-account-manager`, `trimble-maps-fleet`, `trimble-maps-dwell-time`, `trimble-maps-single-search`, `trimble-maps-multi-vehicle-routing`, `trimble-maps-routing-profile`, `trimble-maps-geofence-notifications`, `trimble-maps-road-speeds`, `trimble-maps-routereporter` | Trimble Maps API key, or tokens from each API's authenticate operation (excluded) | A Trimble Maps API key with the product enabled. Fleet and Routing Profile changes to in-cab CoPilot navigation are **safety-excluded** |
| PC*MILER Web Services (Route Reports) | maps | `pcmiler-route-reports` | API key | A PC*MILER Web Services key. The rest of PC*MILER Web Services publishes only a WCF help page (below) |
| PTx Trimble FarmENGAGE Data API | agriculture | `ptx-farmengage` | Not declared; see the developer guide | PTx Trimble API credentials. Sending prescriptions, work orders or resource files to in-cab devices is **safety-excluded** |

Plans for transportation and agriculture operations carry a warning that a qualified person must review and perform them. Every product's sign-in, token and secret-storing operations are excluded, and plans refuse any credential field.

### Catalogued as `excluded` (definition retrieved, operations never used)

| Definition | Why it is excluded |
|---|---|
| App Xchange connector definitions (18 connector pages on help.trimble.com: Spectrum, B2W, ProjectSight, Unity Construct, Unity Maintain, Trimble Connect, Vista and third-party connectors) | Trimble states these OADs "are used internally by the platform and not directly by end users". Vista connector operations that the public Vista Direct API also documents are `variant`s of it. Two definitions returned HTTP 500 and are recorded as unavailable |
| Trimble Maps RouteReporter service host's own Swagger (`routereporterservice.trimblemaps.com`) | The documented RouteReporter definition on developer.trimblemaps.com is catalogued instead; the operations only in this file are undocumented |
| Trimble Maps Content API (beta) | Deprecated by Trimble and replaced by the Places API |
| Trimble Maps Appian Identity API | Credential endpoints (authenticate, refresh, whoami) |
| Transporeon push-notification and outgoing contracts | Contracts the customer implements and Transporeon calls |
| Tekla support API `tsupport-v3-dev` (SwaggerHub org "Tekla") | A development support-ticket API not linked from Tekla documentation |
| SwaggerHub `trimble-analytics/trimble-identity` | An empty stub (0 paths); Identity is covered from the discovery document |
| App Xchange "Example API" (served from a personal GitHub Pages site) | A placeholder, not a Trimble API |

### Without a public machine-readable definition

These products were checked on 2026-09-29 and 2026-10-01. None publishes an OpenAPI, Swagger or AsyncAPI definition that could be catalogued, so there are no operations to list; each is classified here instead.

| Product | What Trimble publishes | Class | Decision |
|---|---|---|---|
| Trimble Identity reference docs | Prose and a Postman collection; the endpoints are in the discovery document (catalogued) | A | Covered through the discovery document |
| PC*MILER Web Services (other than Route Reports) | A WCF HTML help page (https://pcmiler.alk.com/apis/rest/v1.0/Service.svc/help), not OpenAPI | A (API key) | Not catalogued: no machine-readable definition |
| Trimble Maps REST APIs without an embedded definition | Prose documentation on developer.trimblemaps.com | A (API key) | Not catalogued; every page is rescanned on each refresh |
| CoPilot | SDK and CPIK libraries; a `copilot://` URL launch interface (excluded, see [cli-coverage.md](cli-coverage.md)) | D | Not applicable |
| Trimble Maps SDKs (JavaScript, mobile) | SDK references | D | Not applicable |
| Spectrum | Help-site web-service docs; its App Xchange connector definitions are internal (above) | C | Not catalogued as an end-user API |
| Jobpac Connect | Prose help | C | Not catalogued: prose only |
| TAP Store | A Postman collection only | C | Not catalogued: no OpenAPI |
| Trimble Access (Survey Core plug-in API) | Native SDK | D | Not applicable |
| Precision, Catalyst, Trimble Precision SDK | Native SDKs | D | Not applicable |
| Trimble eCognition | Command-line engines (see [cli-coverage.md](cli-coverage.md)) and an SDK automation API documented in the SDK, with no OpenAPI definition | D | Not applicable |
| SketchUp | Ruby API and C SDK; no REST API. Start-up switches are in [cli-coverage.md](cli-coverage.md) | D | Not applicable |
| Tekla Structures, Structural Designer, Tedds, PowerFab | .NET/COM "Open API" SDKs (the name does not mean the OpenAPI format); command lines in [cli-coverage.md](cli-coverage.md) | D | Not applicable |
| Tekla Environment Service | The `teklaenv` CLI's API (`https://cloud.api.trimble.com/tekla/environments/v1`); no published definition | C | Not catalogued; the CLI is in [cli-coverage.md](cli-coverage.md) |
| App Xchange platform | SDK and the `xchange` CLI | D | CLI in [cli-coverage.md](cli-coverage.md) |
| Trimble Business Center | Desktop software; deployment command lines in [cli-coverage.md](cli-coverage.md) | D | Not applicable |
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
