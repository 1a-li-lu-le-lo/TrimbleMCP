# Assumptions

| ID | Assumption | Impact if wrong | Mitigation |
|---|---|---|---|
| A-1 | Trimble Connect is an acceptable first product (the brief left it open) | Rework the adapter choice | Adapter isolation: products plug into `gateway.Registry` |
| A-2 | **Resolved:** the v2.1 `items[].type` values are documented as `FOLDER` and `FILE` | Items reported with the raw lower-cased type | Unknown values pass through unchanged; nothing is guessed |
| A-3 | **Resolved:** folder-item `size` is documented in bytes (requested with `fields=size`); the file `size` is treated the same | Wrong unit label | Labelled "as reported upstream"; verify with the full spec |
| A-4 | Timestamps are RFC 3339 | Unparseable timestamps are omitted | Tested; omitted rather than guessed |
| A-5 | **Resolved:** the folder-item `hash` is documented as MD5 and reported as `checksum_algorithm: "md5"`. `GET /files/{id}` documents no hash, so none is read | n/a | A checksum without a stated algorithm still carries a warning |
| A-6 | One tenant per bridge process (single Trimble Identity session) | Multi-tenant hosting needs a per-tenant registry config | The registry already keys adapters by tenant |
| A-7 | MCP revision 2026-07-28 is current, per the research report | Modern-mode details may drift | Legacy mode is the primary path; modern mode is marked Partial |
| A-8 | The `[project-id]` in `trimbleconnect:` links equals the REST API project `id` | The desktop app opens the wrong project or none | Adapter marked Provisional; every result carries a warning; confirm with Trimble or on a real install |
| A-9 | Only the two-value `show=[view],[panel]` form is valid, and `3D,models` is the application's default. The page names no default; its example opens ToDos "instead of the models tab", which implies models is the usual panel | Over-strict if single values also work; a different default view if 3D is not it | Single-value input is rejected rather than guessed; omitting both emits `3D,models`, which is a documented combination whatever the real default is |
| A-10 | The Trimble Identity access token used for the Core API is accepted by the other Connect APIs (Model, Model Feature, Org, PSet, Topics, Issues, Support, Drive, File Service) | Those reads fail with `authentication_error` | Their definitions declare bearer or OAuth security with no separate audience; verify against a sandbox |
| A-11 | Definitions embedded in Trimble Developer Portal reference pages, and those that products publish directly, describe the current production APIs | Reference plans describe outdated requests | `make catalog` re-fetches and diffs every definition (SHA-256 per source); plans are dry runs for human review; nothing is executed |
| A-12 | Vista's per-operation OpenAPI fragments, merged per module, are complete (the full-definition download is disabled on Vista's documentation site) | An operation missing from `llms.txt` would be missing from the catalogue | The fetch reports the page count and fails on any page it cannot parse; 1,970 pages parsed with 0 errors on 2026-09-29 |
