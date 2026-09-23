# Product selection

"Trimble API" is not one API. Trimble business units publish separate APIs with different portals, identity models, tenants, and contracts. Credentials and IDs from one product never apply to another.

## Configured in this release

| Product ID | What it is | Tools | Status |
|---|---|---|---|
| `trimble-connect` | Trimble Connect REST API (Core), construction project collaboration: projects, folders, files | `trimble_list_projects`, `trimble_list_folder_items`, `trimble_get_file_metadata` | provisional (re-verify before production) |
| `mock` | Simulated adapter with synthetic data for development and evaluation | all read tools | simulated |

Always confirm with `trimble_get_capabilities`; an operator may have disabled a product.

`trimble_get_project` appears only for adapters with a verified single-project endpoint. Trimble Connect does not have one in this release, so resolve projects through `trimble_list_projects`.

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
- Unsure: ask the user which product, after showing what `trimble_get_capabilities` reports.
