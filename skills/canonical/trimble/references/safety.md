# Safety limits

## Prohibited, always

- Operating or steering machinery, equipment, or vehicles, or modifying machine-control or guidance files.
- Certifying survey results, measurements, engineering designs, or construction, or signing regulatory documents.
- Bypassing tenant controls, permissions, licensing, MFA, CAPTCHA, or rate limits.
- Using private or undocumented endpoints or scraping portals.
- Exposing worker, vehicle, or customer location beyond the stated purpose.
- Fabricating IDs, coordinates, or measurements.
- Deleting or altering audit evidence.

## Output labels

Use the labels a tool reports in `data_labels`:

- `observed_data`: returned by the upstream API.
- `simulated_data`: synthetic data from the mock adapter.
- `not_certified`: never professional certification.

Label your own summaries as "model-generated summary" and planning output as "planning aid; qualified review required".
