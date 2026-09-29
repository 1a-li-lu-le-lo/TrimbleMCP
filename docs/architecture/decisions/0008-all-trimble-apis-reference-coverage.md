# ADR-0008: Account for every published Trimble API; other products as reference

- **Status:** Accepted, 2026-09-29. Extends ADR-0007.
- **Context:** The owner asked that the MCP cover all of Trimble's APIs and command-line tools and leave no endpoint or command unaccounted for. ADR-0007 covered the Trimble Connect API family. Trimble publishes definitions for many other products:
  - most are embedded in Trimble Developer Portal reference pages;
  - some are published by the product itself;
  - Vista's are published as one fragment per operation.

  These products differ from Trimble Connect in ways that matter for safety:
  - each has its own credentials and licensing (API keys, per-product OAuth clients, password grants);
  - several are single-tenant or on-premises, with a customer-specific host;
  - some manage vehicles, drivers, field machinery or survey positioning.

  The bridge has no credentials for any of them. The original brief forbids generic HTTP proxies and requires failing closed when the API version, tenant, authorization or host is unknown.
- **Decision:**
  - **Discovery is mechanical and repeatable.** `scripts/fetch-trimble-other-specs.py` covers four sources:
    - it crawls every developer.trimble.com product section's sitemap for reference pages that embed an OpenAPI viewer (`spec-url`);
    - it downloads the definitions that products publish directly;
    - it merges Vista's per-operation fragments into one definition per module;
    - it saves the Trimble Identity OpenID Connect discovery document.

    `cmd/trimble-catalog` classifies each result with the reviewed rules in `cmd/trimble-catalog/sources-other.json`. The build fails on any of these:
    - a definition that matches no rule;
    - a failed download;
    - a rule that matches nothing;
    - a path-item key that is neither an operation nor a documented field;
    - webhooks or callbacks.
  - **New disposition `reference`.** Every operation of another product's API is catalogued, with that API's authentication, access model and "what calling it would need". Such operations are:
    - searchable with `trimble_api_operations` (filter with `family`);
    - validated by `trimble_api_plan` into a dry-run request that lists the documented servers verbatim and does not choose one. The plan takes no `product`, because the key names the API;
    - refused by `trimble_api_read`, and never executed by any tool. `catalog.API.BaseURL` refuses reference APIs, and the Trimble Connect adapter refuses non-Connect APIs.
  - **Safety exclusions.** Operations that could reach machinery, vehicles or field positioning are `excluded` with a `safety:` reason, so they cannot even be planned:
    - FarmENGAGE operations that send prescriptions or work orders to in-cab vehicle devices;
    - Mobile Manager changes to GNSS receiver, antenna, correction source or position stream.

    Plans for transportation and agriculture operations carry a warning that a qualified person must review and perform them.
  - **Project-restricted callers cannot plan reference operations.** Their grants name Trimble Connect projects, which cannot be checked against another product's project IDs, so the plan fails closed.
  - **Trimble Identity endpoints** are catalogued and `excluded`. Only the bridge's own sign-in uses authorize, token and revoke, and agents never handle credentials. The generator fails if the discovery document gains an endpoint not classified in `identityEndpoints`.
  - **Undocumented or placeholder definitions** are catalogued and `excluded`:
    - publicly reachable but not linked from Trimble documentation (Maps RouteReporter, Maps Content API, the Tekla development support API);
    - an empty stub;
    - a placeholder example served from a personal site.
  - **Trimble Connect `/regions` services with no definition** are listed with a note. The generator fails on a new service it does not know.
  - **Products with no machine-readable definition** are classified in the capability matrix: SDKs, desktop APIs, prose-only, WCF help-only and Postman-only APIs.
  - **Command-line tools** are accounted for in `docs/trimble-products/cli-coverage.md`: Trimble Connect for Windows and its installer, the App Xchange `xchange` CLI, `teklaenv`, and the Tekla Structures and SketchUp start-up switches. Only the Trimble Connect for Windows link is supported (ADR-0006). The others are excluded because running local programs or changing remote state is outside the bridge's scope.
  - The catalogue is embedded gzip-compressed (`internal/catalog/catalog.json.gz`, about 340 KB). Large multi-definition APIs get one generated page per definition.
- **Why not execute reference operations now:** executing them would need, for every product:
  - a credential model and a tenant binding;
  - a verified host per customer;
  - an entitlement check;
  - live verification against a sandbox.

  None of these is available, so failing closed means documenting and planning only. Adding execution for a product needs its own adapter, sandbox verification and an ADR, as ADR-0002 did for Trimble Connect.
- **Consequences:**
  - The catalogue grows from 1,747 to **7,016 operations in 230 definitions**:
    - 254 read, 223 plan;
    - 5,190 reference, 936 variant;
    - 413 excluded.
  - `trimble:api:plan` is now a default local scope, because plans send nothing, and `trimble_api_plan` no longer needs a configured Trimble Connect adapter.
  - `make catalog` needs Python 3 with PyYAML, in addition to network access.
  - Assumption A-11: portal-embedded definitions describe the production APIs; Trimble does not state this on every page. Open question Q-9 tracks whether any product should get an executing adapter.
