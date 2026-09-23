# Trimble capability matrix

Last verified: 2026-09-23. Confidence labels:

- **F:** fetched from an official page during verification.
- **S:** seen only in a search excerpt of an official page, or in a third-party page.
- **Unknown:** no evidence.

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
| OPENAPI AVAILABLE | Yes: https://api.swaggerhub.com/apis/Trimble-Connect/tcps/2.0 (F; the retrieved copy was truncated after `/folders/fs/...`) |
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
| READ OPERATIONS (used) | `GET /2.1/projects`, `GET /2.1/folders/{folderId}/items`, `GET /2.0/files/{fileId}` (F) |
| READ OPERATIONS (not used) | `GET /projects/{id}` and `GET /users/me`: shown in guides but not verifiable in the truncated spec. File versions and download URL: verified but deferred |
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
| CONFIDENCE | Medium. The used endpoints are confirmed (F); spec truncation left gaps |
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
| Trimble Connect Model / Topics / Organizer / Property Set APIs | A/C | F (listed on portal) | Deferred |
| Trimble Connect Workspace API | D (browser JS) | S | Not applicable to a server |
| Trimble Maps / PC*MILER REST | A (API key) | S | Candidate for routing and geocoding; needs verification |
| Trimble Agriculture (Ag Developer Network) | B | S | Deferred |
| Viewpoint Vista / Spectrum / Trimble Construction One | C (App Xchange add-on) | S | Deferred |
| ProjectSight | C (OpenAPI v1, Trimble Identity) | S | Candidate |
| e-Builder / Trimble Unity Construct | C | S | Deferred |
| Cityworks / Unity Maintain | C (per-deployment) | S | Deferred |
| Trimble Unity (webhooks via Action Manager) | C | S | Deferred |
| Tekla Structures Open API | D (.NET desktop) | S | Not applicable |
| SketchUp Ruby / C SDK | D | S | Not applicable |
| Trimble Transportation / PeopleNet fleet | B; telematics units reportedly being divested | S | Deferred; re-verify ownership |
| GNSS / RTX corrections | G for this bridge (device correction streams) | S | Not implemented |
| Machinery or vehicle control of any kind | G (prohibited) | n/a | **Never** |

## Workflow matrix

| Workflow | Product | API | Entitlement | Read | Write | Personal data | Safety impact | Approval | Status |
|---|---|---|---|---|---|---|---|---|---|
| Discover capabilities | all | bridge | none | config | none | none | none | L0 | Done |
| List projects | Connect | `GET /2.1/projects` | Connect licence | names, IDs | none | project names | low | L0 | Done |
| Browse folders | Connect | `GET /2.1/folders/{id}/items` | Connect licence | names, IDs | none | file names | low | L0 | Done |
| File metadata | Connect | `GET /2.0/files/{id}` | Connect licence | metadata | none | none | low | L0 | Done |
| Download file | Connect | `GET /files/fs/{id}/downloadurl` | Connect licence | content | none | document content | medium | L1 | Not started |
| Upload new version | Connect | `/files/fs/initiate` + `commit` | Connect licence | none | file | none | medium | L3/L4 | Not started |
| Route or geocode | Trimble Maps | Unknown | API key | n/a | n/a | locations | medium | L1 | Not verified |
| Vehicle position | fleet | Unknown | partner | n/a | n/a | driver location | high | L4 | Not verified |
