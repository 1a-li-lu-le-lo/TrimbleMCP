# Product selection

"Trimble API" is not one API. Trimble business units publish separate APIs with different portals, identity models, tenants, and contracts. Credentials and IDs from one product never apply to another.

## Supported in this release (an operator must enable each)

| Product ID | What it is | Tools | Status and switch |
|---|---|---|---|
| `trimble-connect` | Trimble Connect REST API (Core), construction project collaboration: projects, folders, files | `trimble_list_projects`, `trimble_list_folder_items`, `trimble_get_file_metadata` | Provisional (re-verify before production). Enable with `TRIMBLE_CONNECT_ENABLED=true` |
| `trimble-connect-desktop` | Trimble Connect for Windows links: `trimbleconnect:/projects/<id>?show=<view>,<panel>` | `trimble_build_desktop_link`, `trimble_open_in_desktop` (local Windows only, opt-in) | Provisional: link syntax verified, ID equivalence assumed. Enable with `TRIMBLE_CONNECT_DESKTOP_ENABLED=true` |
| `mock` | Simulated adapter with synthetic data for development and evaluation | `trimble_list_projects`, `trimble_get_project`, `trimble_list_folder_items`, `trimble_get_file_metadata` | Simulated. On by default; disable with `TRIMBLE_MCP_ENABLE_MOCK=false` |

Always confirm with `trimble_get_capabilities`; an operator may have disabled a product.

`trimble_get_project` is listed only when a configured adapter supports it, which is currently only `mock`. The `trimble-connect` adapter does not support it, so resolve Connect projects through `trimble_list_projects`.

For anything beyond capabilities: a product is served by an adapter, and each tool is listed only when an enabled adapter supports it.

## Not configured (do not attempt)

These Trimble products exist but have no adapter here. Say so and stop:

- Trimble Maps / PC*MILER (routing, geocoding).
- Trimble Agriculture APIs.
- Viewpoint Vista, Spectrum, Trimble Construction One, ProjectSight, e-Builder, Trimble Unity, Cityworks.
- Tekla Structures and SketchUp (desktop SDKs, not server APIs).
- Trimble Transportation / fleet and telematics APIs.
- GNSS and correction services.
- Trimble Connect model, topic, property-set, and organizer APIs.

## Telling products apart

- "Connect project", "TC project", "Trimble Connect", or a construction document folder: `trimble-connect`.
- A truck, route, fleet, or hours-of-service question: not configured.
- A drawing open in Tekla or SketchUp on the user's machine: a desktop SDK, not reachable here.
- "Open it in Trimble Connect for Windows", "the desktop app", "show the ToDos panel": `trimble-connect-desktop`. Resolve the project through `trimble-connect` first.
- Unsure: ask the user which product, after showing what `trimble_get_capabilities` reports.
