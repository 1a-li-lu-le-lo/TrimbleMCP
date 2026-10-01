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
    - webhooks or callbacks in a Trimble Connect definition (other products' OpenAPI 3.1 webhooks are catalogued as `excluded`: the provider calls the customer).
  - **New disposition `reference`.** Every operation of another product's API is catalogued, with that API's authentication, access model and "what calling it would need". Such operations are:
    - searchable with `trimble_api_operations` (filter with `family`);
    - validated by `trimble_api_plan` into a dry-run request that lists the documented servers verbatim and does not choose one. The plan takes no `product`, because the key names the API;
    - refused by `trimble_api_read`, and never executed by any tool. `catalog.API.BaseURL` refuses reference APIs, and the Trimble Connect adapter refuses non-Connect APIs.
  - **Safety exclusions.** Operations that could reach machinery, vehicles or field positioning are `excluded` with a `safety:` reason, so they cannot even be planned:
    - FarmENGAGE operations that send prescriptions, work orders or resource files (guidance lines, boundaries, vehicle and implement profiles) to in-cab devices, and both steps of a prescription import (it can target devices);
    - Trimble Maps Fleet and Routing Profile changes to the routing, configuration or map data that in-cab CoPilot navigation uses;
    - Mobile Manager changes to GNSS receiver, antenna, correction source or position stream;
    - every other product's sign-in, token and secret-storing operations, as for Trimble Identity (added after the 2026-09-29 audit).

    Every plan also refuses a parameter or body field that carries a credential.

    Plans for transportation and agriculture operations carry a warning that a qualified person must review and perform them.
  - **Project-restricted callers cannot plan reference operations.** Their grants name Trimble Connect projects, which cannot be checked against another product's project IDs, so the plan fails closed.
  - **Trimble Identity endpoints** are catalogued and `excluded`. Only the bridge's own sign-in uses authorize, token and revoke, and agents never handle credentials. The generator fails if the discovery document gains an endpoint not classified in `identityEndpoints`.
  - **Undocumented or placeholder definitions** are catalogued and `excluded`:
    - publicly reachable but not linked from Trimble documentation (Maps RouteReporter, Maps Content API, the Tekla development support API);
    - an empty stub;
    - a placeholder example served from a personal site.
  - **Trimble Connect `/regions` services with no definition** are listed with a note. The generator fails on a new service it does not know.
  - **Products with no machine-readable definition** are classified in the capability matrix: SDKs, desktop APIs, prose-only, WCF help-only and Postman-only APIs.
  - **Command-line tools and local launchers** are accounted for in `docs/trimble-products/cli-coverage.md`, command by command (see that page for the full list of tools). Only the Trimble Connect for Windows link is supported (ADR-0006); the others are excluded because running local programs, changing a host or handling credentials is outside the bridge's scope.
  - The catalogue is embedded gzip-compressed (`internal/catalog/catalog.json.gz`, about 600 KB). Large multi-definition APIs get one generated page per definition.
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
  - **Update, 2026-10-01 (after an independent audit):** discovery now also covers every page of developer.trimblemaps.com, the App Xchange connector pages, the Transporeon developer documentation, AsyncAPI definitions, Viewpoint For Projects, and endpoints documented only in prose. Discovery failures are recorded and fail the build. The catalogue is **12,963 operations in 347 definitions** (61 APIs: 254 read, 223 plan, 6,075 reference, 2,985 variant, 3,426 excluded). Credential operations of every product, more vehicle-reaching operations (FarmENGAGE resource files and prescription imports; Trimble Maps fleet and routing profiles) and webhooks are excluded, and plans refuse credential fields.
  - **Update, 2026-10-01 (second audit):** see the totals in `docs/trimble-products/endpoints/README.md`. Added TruckMate Imaging, Transporeon definitions embedded inline in its documentation, the legacy Vista Viewpoint API and Kuebix (both excluded), and the App Xchange S/FTP connector; Mobile Manager's WebSocket definitions are separate APIs. The generator now excludes any operation whose request carries a credential field or requires a credential parameter, and plans match credential words inside field names and string bodies.
  - **Update, 2026-10-01 (sixth audit):** added Jobpac Connect (`jobpac-connect`, a YAML definition embedded in its documentation page; discovery now also reads `swagger-data` YAML script blocks) and Tekla PowerFab Go (`tekla-powerfab-go`). Jobpac passes its session token as a query parameter, so 126 of its 129 operations are excluded as credential operations. The capability matrix now also classifies the SketchUp Connector MCP service, the GNSS receivers' Programmatic Interface, the Tekla Structural Designer Remote Access API, and APIs advertised without public documentation; the command-line page adds B2W Estimate, Stabicad, Novapoint, the Tekla Template Editor, `.tsep` central installation, `lmutil lmstat` and the DSTV to DXF Converter.
  - **Update, 2026-10-01 (seventh audit):** discovery reads definitions embedded in the Forge (`adf-extension`) version of Confluence's OpenAPI viewer, which added the Transporeon Freight Audit SelfService and Market Insights APIs and the appointment "Discovery endpoints". An unpublished TruckMate SwaggerHub draft is catalogued as excluded. The capability matrix corrects Spectrum (it also has REST services), records the 2025 sale of PeopleNet, and adds the TBC Data Service, Appian DRTrack web services and the OEM receiver protocol; the command-line page adds Stabicad's client and SCCM sequences, more Tekla licence and multi-user server commands, and the TMWSuite Cloud Hub Connector installer.
  - **Update, 2026-10-01 (eighth audit):** Trimble Connect's ECom Service definition, linked from Trimble's .NET SDK guide, replaces the prose-built `connect-ecom`; its licensing, entitlement and invitation-key changes are safety-excluded. The Project, Batch, User App and WOPI services serve definitions at `/v1/api-docs` that Trimble's documentation does not link, so they are catalogued as excluded. The capability matrix adds Trimble Maps Trip Management, the PC*Miler Rail web services and Rail-Connect, the Viewpoint Team connector, the Master Builder API and the retired e-Builder OData API; the command-line page adds the TSEP builders, Rebar mesh view creator, `tsd.exe /reg`, Tedds MSI logging, both DSTV converters as documented for 2026, and the App Xchange Agent scripts.
  - **Update, 2026-10-01 (ninth audit):** the help-site sweep found nothing new. The capability matrix adds Transporeon's SOAP services (discovery now reports Confluence pages that link a WSDL), TMWSuite SystemsLink and CashLink, the TMT SOAP API, and describes the Trimble Connect for Windows .NET API as the local WCF service it is; the command-line page documents SketchUp's `-RubyStartup` debug launch and the `yard-sketchup` commands.
  - **Update, 2026-10-01 (tenth audit):** the registries sweep found nothing new. The capability matrix adds the PC*MILER and PC*Miler Rail SOAP services (public WSDLs, outside the catalogue's formats), the DRTrack Users API, the MPS865 receiver's SMS commands and the advertised TruckMate MCP Server, and records the 2024 sale of Telog and Unity Remote Monitoring to Badger Meter and that Kuebix is now part of FreightWise.
  - **Update, 2026-10-01 (eleventh audit):** the developer-portal and registry sweeps found nothing new. Trimble Business Systems' licensing definitions (EMS, Transition Layer, DX Trials, EMS Migration) are catalogued as excluded, because agents never inspect or change licences. The capability matrix adds the Unity Work Management ArcGIS connector and form-script API and WorksManager's ISO 15143-4 exchange, and notes AGCO's control of PTx Trimble; the command-line page adds Quadri's `--batchtaskerror`.
  - `trimble:api:plan` is now a default local scope, because plans send nothing, and `trimble_api_plan` no longer needs a configured Trimble Connect adapter.
  - `make catalog` needs Python 3 with PyYAML, in addition to network access.
  - Assumption A-11: portal-embedded definitions describe the production APIs; Trimble does not state this on every page. Open question Q-9 tracks whether any product should get an executing adapter.
