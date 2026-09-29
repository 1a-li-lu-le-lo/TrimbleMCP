# Safety limits

## Prohibited, always

- Operating or steering machinery, equipment, or vehicles, or modifying machine-control or guidance files.
- Certifying survey results, measurements, engineering designs, or construction, or signing regulatory documents.
- Bypassing tenant controls, permissions, licensing, MFA, CAPTCHA, or rate limits.
- Using private or undocumented endpoints or scraping portals.
- Exposing worker, vehicle, or customer location beyond the stated purpose.
- Fabricating IDs, coordinates, or measurements.
- Deleting or altering audit evidence.
- Planning or describing how to perform an operation excluded with a `safety:` reason: sending prescriptions or work orders to in-cab vehicle devices, or changing GNSS receiver, antenna, correction or position-stream settings.
- Calling another product's API, or asking for its credentials. `reference` operations are plans for a qualified person, never actions.

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
