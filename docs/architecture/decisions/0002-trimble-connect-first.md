# ADR-0002: Trimble Connect read-only as the first adapter

- **Status:** Accepted, 2026-09-23.
- **Context:** The brief left products and use cases as placeholders ("discover and recommend"). Of the researched Trimble products, Trimble Connect has official public documentation, an OpenAPI definition, documented regional hosts, a documented identity flow, and staging hosts.
- **Decision:** Build one adapter well: Trimble Connect, read-only, covering projects, folder items, and file metadata. Do not implement `GET /projects/{id}` or `/users/me`, because they could not be verified in the retrieved spec.
- **Update (2026-09-23):** the complete OpenAPI definition became retrievable. `GET /projects/{projectId}` is now implemented, and the undocumented `revision` and `hash` fields of `GET /files/{fileId}` are no longer read.
- **Consequences:**
  - `trimble_get_project` is hidden when only Connect is configured.
  - Other products are listed in the capability matrix as candidates only.
