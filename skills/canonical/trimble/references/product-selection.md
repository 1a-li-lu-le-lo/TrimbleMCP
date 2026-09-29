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

| Family | Products (catalogue API ids) |
|---|---|
| construction | Vista (`vista`), ProjectSight (`projectsight`), Unity Construct / e-Builder (`unity-construct`), Unity Maintain / Permit / Cityworks (`unity-maintain-permit`), Accubid Anywhere (`accubid-*`), Civil Site Management (`civil-site-management`) |
| geospatial | Field Configuration, Field Data / Jobs, Mobile Manager |
| transportation | TMT Fleet Maintenance (`tmt`), TMWSuite (`tmwsuite-*`), TruckMate (`truckmate*`) |
| maps | Trimble Maps Places (`trimble-maps-places`) |
| agriculture | PTx FarmENGAGE (`ptx-farmengage`) |

## Not available at all (say so and stop)

These have no public machine-readable API definition, or are out of scope:

- Trimble Maps routing and PC*MILER (WCF help pages only), and other Trimble Maps REST APIs.
- Spectrum, Jobpac Connect, TAP Store (prose or Postman only).
- Tekla Structures and SketchUp (desktop SDKs, not server APIs); CoPilot and other SDK-only products.
- Fleet telematics and vehicle position.
- GNSS and correction services.
- Anything that would operate machinery or vehicles (prohibited).

## Telling products apart

- "Connect project", "TC project", "Trimble Connect", or a construction document folder: `trimble-connect`.
- A truck, route, fleet, or hours-of-service question: no data is available. For TMWSuite, TruckMate or TMT operations, reference plans only.
- A Vista, ProjectSight, Unity, e-Builder, Cityworks or Accubid question: reference plans only; the bridge returns none of their data.
- A drawing open in Tekla or SketchUp on the user's machine: a desktop SDK, not reachable here.
- "Open it in Trimble Connect for Windows", "the desktop app", "show the ToDos panel": `trimble-connect-desktop`. Resolve the project through `trimble-connect` first.
- Unsure: ask the user which product, after showing what `trimble_get_capabilities` reports.
