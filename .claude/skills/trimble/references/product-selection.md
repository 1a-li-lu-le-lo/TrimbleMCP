# Product selection

"Trimble API" is not one API. Trimble business units publish separate APIs with different portals, identity models, tenants, and contracts. Credentials and IDs from one product never apply to another.

## Supported in this release (an operator must enable each)

| Product ID | What it is | Tools | Status and switch |
|---|---|---|---|
| `trimble-connect` | Trimble Connect REST API (Core), construction project collaboration: projects, folders, files | `trimble_list_projects`, `trimble_get_project`, `trimble_list_folder_items`, `trimble_get_file_metadata` | Provisional (re-verify before production). Enable with `TRIMBLE_CONNECT_ENABLED=true` |
| `trimble-connect-desktop` | Trimble Connect for Windows links: `trimbleconnect:/projects/<id>?show=<view>,<panel>` | `trimble_build_desktop_link`, `trimble_open_in_desktop` (local Windows only, opt-in) | Provisional: link syntax verified, ID equivalence assumed. Enable with `TRIMBLE_CONNECT_DESKTOP_ENABLED=true` |
| `mock` | Simulated adapter with synthetic data for development and evaluation | `trimble_list_projects`, `trimble_get_project`, `trimble_list_folder_items`, `trimble_get_file_metadata` | Simulated. On by default; disable with `TRIMBLE_MCP_ENABLE_MOCK=false` |

Always confirm with `trimble_get_capabilities`; an operator may have disabled a product.

`trimble_get_project` is supported by `trimble-connect` and `mock`. Resolve a project name to its ID with `trimble_list_projects` first; never pass a guessed ID.

For anything beyond capabilities: a product is served by an adapter, and each tool is listed only when an enabled adapter supports it.

## Whole Trimble Connect API family (catalogue)

With `trimble-connect` enabled, `trimble_api_operations`, `trimble_api_read` and `trimble_api_plan` reach every operation of the 11 official Trimble Connect APIs (see the API catalogue reference listed in SKILL.md). The typed tools above cover the most common Core reads.

## Other products: documented, never called (`reference`)

The bridge has no adapter or credentials for these products. It can still find their documented operations and prepare a dry-run request plan with `trimble_api_plan`, without a `product`. It never calls them, so it cannot return their data. Say so, and offer the plan if the user wants one.

Find any of them with `trimble_api_operations` (filter with `family`, or call it without `api` to list every API id). By family:

| Family | Products (catalogue API ids) |
|---|---|
| connect | Trimble Connect ECom Service (`connect-ecom`: accounts, admins, entitlements, usage) and Core Account projects endpoint (`connect-projects-api`), Trimble Connect Status Sharing (`trimble-connect-status-sharing`) |
| construction | Vista (`vista`), ProjectSight (`projectsight`), Viewpoint For Projects (`viewpoint-for-projects`), Unity Construct / e-Builder (`unity-construct`), Unity Maintain / Permit / Cityworks (`unity-maintain-permit`), Accubid Anywhere (`accubid-*`), Civil Site Management (`civil-site-management`), MEPcontent (`mepcontent`), Jobpac Connect (`jobpac-connect`; almost every operation is excluded because the Jobpac token travels in the request), Tekla PowerFab Go (`tekla-powerfab-go`) |
| geospatial | Field Configuration (`geospatial-field-configuration`), Field Data / Jobs (`geospatial-field-data`), Mobile Manager (`mobile-manager`, and WebSocket streams `mobile-manager-ws-v1`, `mobile-manager-ws-v2`) |
| transportation | TMT Fleet Maintenance (`tmt`), TMWSuite (`tmwsuite-*`), TruckMate (`truckmate*`, including `truckmate-imaging`), Transporeon (`transporeon-*`: carriers, shippers, visibility, telematics, eCMR, transport operations, rate management, freight audit and freight audit self-service, freight procurement, yard appointments, market insights) |
| maps | Trimble Maps (`trimble-maps-*`: places, account manager, fleet, dwell time, single search, multi-vehicle routing, routing profile, geofence notifications, road speeds, RouteReporter) and PC*MILER Route Reports (`pcmiler-route-reports`) |
| agriculture | PTx FarmENGAGE (`ptx-farmengage`) |

Some of their operations are `excluded` with a `safety:` reason: anything that sends data to in-cab devices or changes in-cab navigation or field positioning, and every operation that signs in or carries a credential. Explain the reason and stop; do not suggest another way.

## Not available at all (say so and stop)

These have no public machine-readable API definition, or are out of scope (see the capability matrix):

- PC*MILER Web Services other than Route Reports (WCF help pages only), and Trimble Maps REST APIs without a published definition.
- Spectrum Data Exchange (SOAP and REST services on each customer's server), Trimble Field View (SOAP services on regional hosts), Appian DRTrack web services (SOAP on a per-customer host), Trimble Maps Trip Management and the PC*Miler Rail web services (prose only), Transporeon's and PC*MILER's SOAP services, TMWSuite SystemsLink, the TMT SOAP API and the TruckMate MCP Server, Master Builder and the Viewpoint Team connector, the Trimble Business Center Data Service (local OData), TAP Store (prose or Postman only), B2W Operational Suite (per-tenant Swagger only), Trimble FSM / GeoManager.
- Tekla Structures and SketchUp (desktop SDKs, not server APIs); the desktop Tekla PowerFab XML command API; CoPilot and other SDK-only products.
- The SketchUp Connector, a Trimble-hosted MCP service: the user connects it to their client directly; this bridge does not proxy it.
- APIs advertised without public documentation (POSPac Cloud, LiDAR QC Cloud, 3D Warehouse, Forestry WSX and DataVuze, TMT Road Call).
- GNSS correction streams (RTX), GNSS receivers' Programmatic Interface (`/prog/...` on the device) and OEM receiver interface protocol, and Settop M1 instrument control. PeopleNet telematics was sold to Platform Science in 2025 and is no longer a Trimble API.
- Anything that would operate machinery or vehicles (prohibited).

Vehicle positions and fleet data are catalogued for some products (Transporeon visibility and telematics, Trimble Maps RouteReporter and Dwell Time, Civil Site Management), but only as reference plans: the bridge returns no location data, and locations are personal data.

## Telling products apart

- "Connect project", "TC project", "Trimble Connect", or a construction document folder: `trimble-connect`.
- A truck, route, fleet, or hours-of-service question: no data is available. For TMWSuite, TruckMate, TMT, Transporeon or Trimble Maps fleet and routing operations, reference plans only (with the safety exclusions above).
- A Vista, ProjectSight, Viewpoint For Projects, Unity, e-Builder, Cityworks, Accubid, Jobpac or PowerFab Go question: reference plans only; the bridge returns none of their data.
- A drawing open in Tekla or SketchUp on the user's machine: a desktop SDK, not reachable here.
- "Open it in Trimble Connect for Windows", "the desktop app", "show the ToDos panel": `trimble-connect-desktop`. Resolve the project through `trimble-connect` first.
- Unsure: ask the user which product, after showing what `trimble_get_capabilities` reports.
