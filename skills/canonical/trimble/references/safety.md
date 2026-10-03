# Safety limits

## Prohibited, always

- Operating or steering machinery, equipment, or vehicles, or modifying machine-control or guidance files.
- Certifying survey results, measurements, engineering designs, or construction, or signing regulatory documents.
- Bypassing tenant controls, permissions, licensing, MFA, CAPTCHA, or rate limits.
- Using private or undocumented endpoints or scraping portals.
- Exposing worker, vehicle, or customer location beyond the stated purpose.
- Fabricating IDs, coordinates, or measurements.
- Deleting or altering audit evidence.
- Planning or describing how to perform an operation excluded with a `safety:` reason: FarmENGAGE operations that send prescriptions, work orders or resource files (guidance lines, boundaries, vehicle and implement profiles) to in-cab devices, including both steps of a prescription import; Trimble Maps Fleet and Routing Profile changes to the routing, configuration or map data that in-cab CoPilot navigation uses; Mobile Manager changes to GNSS receiver, antenna, correction or position-stream settings; TruckMate Mobile Communications writes (driver hours of service, positions, sensor data, logins and driver messages); licence, entitlement, activation and invitation-key changes in the Trimble Connect ECom Service; and every operation that signs in, issues or shares a token (such as Transporeon eCMR sharing links), or whose request carries a credential (agents never handle credentials).
- Calling another product's API, or asking for its credentials. Plans refuse any parameter or body field whose name carries a credential (for example LoginPassword, client_secret, X-Api-Key or a token), including inside string and form-encoded bodies. `reference` operations are plans for a qualified person, never actions.

## Output labels

Use the labels a tool reports in `data_labels`:

- `observed_data`: returned by the upstream API.
- `simulated_data`: synthetic data from the mock adapter.
- `not_certified`: never professional certification.
- `configuration`: capability and configuration data.
- `local_launch_link`, `no_project_data`, `dry_run`, `local_side_effect`: results for a Trimble Connect for Windows link or launch. No project data was read.
- `dry_run`, `planning_aid`, `not_executed`: a plan from `trimble_api_plan`; nothing was sent. `reference_only` adds that the product is never called by the bridge.
- `catalogue`: operation documentation from the official definitions.

Label your own summaries as "model-generated summary" and planning output as "planning aid; qualified review required".
