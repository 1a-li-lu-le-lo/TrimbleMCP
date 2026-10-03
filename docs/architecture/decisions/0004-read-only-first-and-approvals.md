# ADR-0004: No mutation tools until the approval framework exists

- **Status:** Accepted, 2026-09-23.
- **Decision:**
  - This release exposes only Level 0 tools.
  - Write and delete scopes are rejected at configuration load.
  - Requested changes produce an operation plan (`templates/operation-plan.md`), never a simulated change.
- **Next:** The approval framework (PR 11) must issue server-side approval references bound to actor, tenant, resource, expected version, and expiry, together with idempotency records. Only then can the first reversible mutation (a Connect file version upload using `If-Match`) be added.
