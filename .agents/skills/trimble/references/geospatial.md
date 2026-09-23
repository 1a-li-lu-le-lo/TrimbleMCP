# Geospatial and unit rules

No configured tool in this release returns or transforms coordinates. These rules still apply whenever coordinates come up.

- Never assume WGS 84, and never infer an EPSG code from numeric ranges.
- State the axis order: EPSG:4326 is formally latitude-first, while GeoJSON is longitude-first.
- Height needs a reference: ellipsoidal or orthometric, with its vertical datum.
- Keep feet distinct. The international foot is exactly 0.3048 m and the US survey foot is exactly 1200/3937 m. They differ by 2 ppm, about 0.61 m over 1,000,000 ft.
- Do not transform coordinates without an explicit source CRS, an explicit target CRS, a verified transformation, and recorded transformation metadata. No transformation tool is configured, so stop and say so.
- Telemetry and consumer positions are not survey-grade. Never present them as certified measurements.
- Timestamps from the bridge are UTC. Convert to local time only with an explicit IANA time zone, and be careful around daylight-saving transitions.
