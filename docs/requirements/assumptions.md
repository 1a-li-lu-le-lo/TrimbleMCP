# Assumptions

| ID | Assumption | Impact if wrong | Mitigation |
|---|---|---|---|
| A-1 | Trimble Connect is an acceptable first product (the brief left it open) | Rework the adapter choice | Adapter isolation: products plug into `gateway.Registry` |
| A-2 | The v2.1 `items[].type` values are `FOLDER` and `FILE`, matching the `objectTypes` parameter values | Items reported with the raw lower-cased type | Unknown values pass through unchanged; nothing is guessed |
| A-3 | `size` is in bytes | Wrong unit label | Labelled "as reported upstream"; verify with the full spec |
| A-4 | Timestamps are RFC 3339 | Unparseable timestamps are omitted | Tested; omitted rather than guessed |
| A-5 | The file `hash` algorithm is undocumented | Not used for integrity | Warning emitted with every metadata result |
| A-6 | One tenant per bridge process (single Trimble Identity session) | Multi-tenant hosting needs a per-tenant registry config | The registry already keys adapters by tenant |
| A-7 | MCP revision 2026-07-28 is current, per the research report | Modern-mode details may drift | Legacy mode is the primary path; modern mode is marked Partial |
| A-8 | The `[project-id]` in `trimbleconnect:` links equals the REST API project `id` | The desktop app opens the wrong project or none | Adapter marked Provisional; every result carries a warning; confirm with Trimble or on a real install |
| A-9 | Only the two-value `show=[view],[panel]` form is valid; `3D,models` is the default (the doc's example opens ToDos "instead of the models tab") | Over-strict if single values also work | Single-value input is rejected rather than guessed; omitting both uses the documented default |
