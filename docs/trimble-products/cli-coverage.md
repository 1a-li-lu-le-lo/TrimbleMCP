# Command-line coverage

Every command-line interface that Trimble documents or publishes is accounted for here, with each command or switch and its disposition. So is every `trimblectl` command. `internal/skilltest` checks that this page covers every Trimble Connect view and panel and every `trimblectl` command.

Only the Trimble Connect for Windows link (section 1) is supported. The bridge exposes no shell or process-launching tool, handles no credentials, and never changes a host. Every other command below is therefore excluded, with the reason.

## 1. Trimble Connect for Windows command line (supported)

Source: https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line (verified 2026-09-23).

Syntax, as documented: `"trimbleconnect:/projects/[project-id]?show=[view parameter],[panel parameter]"`. Parameters are case-insensitive. The same link can be started from a command line or a browser.

| Surface | MCP tool | CLI |
|---|---|---|
| Build the link (no side effects) | `trimble_build_desktop_link` | `trimblectl desktop link` |
| Open the link on this Windows machine (opt-in, dry run by default) | `trimble_open_in_desktop` | `trimblectl desktop open` |

Every documented combination is supported. Omitting both values gives the documented default, `3D,models` (the documented example opens ToDos "instead of the models tab").

| View | Panel | Link |
|---|---|---|
| `projects` | `clashes` | `trimbleconnect:/projects/<id>?show=projects,clashes` |
| `projects` | `models` | `trimbleconnect:/projects/<id>?show=projects,models` |
| `projects` | `objects` | `trimbleconnect:/projects/<id>?show=projects,objects` |
| `projects` | `ToDos` | `trimbleconnect:/projects/<id>?show=projects,ToDos` |
| `projects` | `views` | `trimbleconnect:/projects/<id>?show=projects,views` |
| `data` | `clashes` | `trimbleconnect:/projects/<id>?show=data,clashes` |
| `data` | `models` | `trimbleconnect:/projects/<id>?show=data,models` |
| `data` | `objects` | `trimbleconnect:/projects/<id>?show=data,objects` |
| `data` | `ToDos` | `trimbleconnect:/projects/<id>?show=data,ToDos` |
| `data` | `views` | `trimbleconnect:/projects/<id>?show=data,views` |
| `3D` | `clashes` | `trimbleconnect:/projects/<id>?show=3D,clashes` |
| `3D` | `models` | `trimbleconnect:/projects/<id>?show=3D,models` |
| `3D` | `objects` | `trimbleconnect:/projects/<id>?show=3D,objects` |
| `3D` | `ToDos` | `trimbleconnect:/projects/<id>?show=3D,ToDos` |
| `3D` | `views` | `trimbleconnect:/projects/<id>?show=3D,views` |

Not supported, with reasons:

- **A view or panel on its own:** undocumented, so it is rejected rather than guessed.
- **Other URL forms:** undocumented; the `trimbleconnect://` double-slash form and extra parameters are never emitted.

## 2. Trimble Connect for Windows installer (excluded)

Source: https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/install-trimble-connect-for-windows/enterprise-installation-guide (verified 2026-09-23).

| Command | What it does | Disposition |
|---|---|---|
| `TrimbleConnectSetup-VersionNumber-x64.exe /ad:\preq` | Extracts `Trimble Connect.msi` for administrator deployment | **Excluded:** installs software and needs administrator rights; host changes are out of scope for an agent (safety Level 5) |
| MSI deployment of `Trimble Connect.msi` (the installer forces `ALLUSERS=1`) | Machine-wide install | **Excluded:** same reason |

Silent-install switches such as `/qn` and `/quiet` appear only on third-party sites. They are not official, and are not used.

## 3. Trimble Connect Sync (no command line)

The official installation and sign-in pages describe only an interactive application (https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-sync/getting-started/install-trimble-connect-sync). No command line is documented, so there is nothing to cover. Syncing is bidirectional data movement, and would be excluded in any case.

## 4. Web links (not a command line)

The Workspace API documents the web viewer path `https://web.connect.trimble.com/projects/:projectId/viewer/3d` and an embedded viewer (`?isEmbedded=true`). These are browser integrations, not command-line tools, so they are not built by the bridge. Their query parameters beyond the path are documented only for the embedded JavaScript API.

## 5. `trimblectl` (this project's CLI)

`trimblectl` runs every data command through the same gateway as MCP clients, with the same authorization and audit.

| Command | MCP tool or function |
|---|---|
| `trimblectl capabilities` | `trimble_get_capabilities` |
| `trimblectl projects list` | `trimble_list_projects` |
| `trimblectl projects get` | `trimble_get_project` |
| `trimblectl files list` | `trimble_list_folder_items` |
| `trimblectl files metadata` | `trimble_get_file_metadata` |
| `trimblectl desktop link` | `trimble_build_desktop_link` |
| `trimblectl desktop open` | `trimble_open_in_desktop` |
| `trimblectl api operations` | `trimble_api_operations` |
| `trimblectl api read` | `trimble_api_read` |
| `trimblectl api plan` | `trimble_api_plan` |
| `trimblectl auth login` | Trimble Identity sign-in (PKCE, Serial PKCE refresh) |
| `trimblectl auth logout` | Revoke the refresh token and delete the local store |
| `trimblectl audit verify` | Verify the audit log's hash chains |
| `trimblectl token new` | Create a remote caller token (the SHA-256 goes in the tokens file) |
| `trimblectl diagnostics` | Configuration and secret-file checks |
| `trimblectl help` | Usage |

## 6. App Xchange Connector CLI (`xchange`), excluded

Source: https://trimble-xchange.github.io/connector-docs/cli-usage/, linked from https://developer.trimble.com/docs/app-xchange/tools/sdk/connector/v1/ (verified 2026-09-29). NuGet package `Trimble.Xchange.Connector.CLI` (owner Trimble), installed with `dotnet tool install Trimble.Xchange.Connector.CLI --global`. It is developer tooling for building App Xchange connectors in a local repository.

**Why it is excluded:** it runs in a developer's source tree and writes files there. Several commands sign in with Trimble ID, take an Xchange API key on the command line, or change the Xchange platform. The bridge runs no local programs and handles no credentials.

| Command | What it does | Disposition |
|---|---|---|
| `xchange --help`, `xchange --version`, `--verbosity` (`-v`) | Help, CLI and SDK versions, log level | Excluded: local tooling |
| `xchange action new` | Generates an action handler (`--module-id`, `--object-name`, `--name`) | Excluded: writes source files |
| `xchange client new` | Generates an API client (`--auth-type` Custom, ApiKey, Basic, OAuth2CodeFlow, OAuth2ClientCredentials or OAuth2Password; `--type Http`) | Excluded: writes source files |
| `xchange code close` | Closes an open code submission (`--correlation-id`) | Excluded: changes platform state |
| `xchange code feedback` | Downloads the feedback archive of a submission | Excluded: needs an Xchange API key; downloads files |
| `xchange code get` | Gets a submission | Excluded: needs an Xchange API key |
| `xchange code list` | Lists submissions (`--submitter`, `--getFinished`, `--getOpen`, `--limit`, `--status`) | Excluded: needs an Xchange API key |
| `xchange code status` | Shows a submission's status | Excluded: needs an Xchange API key |
| `xchange code submit` | Submits connector code (`--api-key`, `--submitter`, `--draft`, `--dry-run`, `--override-archive`, `--allow-breaking-changes`) | Excluded: changes platform state |
| `xchange connector new` | Creates a connector project | Excluded: writes source files |
| `xchange connector repository` | Creates the connector's repository on the platform (`--name`, `--key`, `--team-name`) | Excluded: changes platform state |
| `xchange data-object new` | Generates a data object (`--reader-type` Full or Incremental, `--no-pagination`) | Excluded: writes source files |
| `xchange extract` | Extracts and validates the connector definition (`--validate-only`, `--output-filename`, `--allow-breaking-changes`) | Excluded: local build step |
| `xchange login` | Signs in with Trimble ID | Excluded: credential handling |
| `xchange logout` | Signs out | Excluded: credential handling |
| `xchange module new` | Generates a module (`--id`, `--name`, `--key`, `--version`) | Excluded: writes source files |
| `xchange test init` | Initialises the connector's test project | Excluded: writes source files |
| `xchange test reset` | Resets test settings (`--configuration`) | Excluded: writes source files |

## 7. Tekla Environments CLI (`teklaenv`), excluded

Source: the readme of NuGet package `Tekla.Environment.Api.Client.Console` 1.0.0 (author Trimble, published 2026-09-20). No Tekla or Trimble documentation page was found, so its official status is unverified. Its default API is `https://cloud.api.trimble.com/tekla/environments/v1`, for which no definition is published.

**Why it is excluded:** it takes a Trimble ID access token (`-t`) or an API key (`-k`) on the command line, where other local processes can see it. Most commands change packages, definitions, permissions or API keys, and the downloads write files to disk.

| Command | What it does | Disposition |
|---|---|---|
| `teklaenv DownloadPackage` | Downloads a package (`-t` or `-k`, `-i`, `-v`, `-o`, `--urlOnly`, `--versionConstraint`) | Excluded: credential on the command line; writes files; `--urlOnly` returns a download URL |
| `teklaenv UploadPackage` | Uploads a package with metadata (`-f`, `-m key=value`) | Excluded: changes remote state |
| `teklaenv CreatePackageDefinition` | Bulk-creates or updates package definitions from CSV (`-d`, `--patch`) | Excluded: changes remote state |
| `teklaenv CreateMetadataDefinition` | Bulk-creates metadata definitions from CSV | Excluded: changes remote state |
| `teklaenv CreatePermission` | Grants a permission on a package definition | Excluded: changes access control |
| `teklaenv DeletePermission` | Removes a permission | Excluded: changes access control |
| `teklaenv ReportPermissions` | Reports the permissions on package definitions | Excluded: credential on the command line |
| `teklaenv apikeys` | Creates, rotates or lists API keys (`--type key\|client`, `--token`, `--expire`, `--rotate`, `--list`) | Excluded: credential management |
| `-u` / `--apiurl`, `--help` | Overrides the API URL; help | Excluded with the tool |

## 8. Tekla Structures start-up switches (`TeklaStructures.exe`), excluded

Source: https://support.tekla.com/doc/tekla-structures/2025/cus_create_startup_shortcuts_with_customized_initializations (verified 2026-09-29).

**Why it is excluded:** it launches a desktop modelling application and can create models or run macros, which are code. Model and initialisation paths are local file paths, and the bridge has no filesystem tool. Unlike the Trimble Connect link, there is no API with which to verify the target first.

| Switch | What it does | Disposition |
|---|---|---|
| `-I <ini_file_path>` | Loads an initialisation file before the environment files; can bypass the setup and sign-in dialog | Excluded: changes start-up configuration and sign-in behaviour |
| `-i <ini_file_path>` | Loads an initialisation file after the role files | Excluded: local file path |
| `<model_path>` | Opens a model | Excluded: local file path |
| `<model_path> /autosaved` | Opens the autosaved copy of a model | Excluded: local file path |
| `/create:<model_path>` | Creates a model | Excluded: creates data |
| `/modelTemplate:<template_name>` | Model template for `/create` | Excluded: creates data |
| `/server:<server_name>` | Multi-user server for `/create` | Excluded: creates data |
| `-m <macro_file_path>` | Runs a macro after start-up | Excluded: runs code |

## 9. SketchUp installer, excluded

Source: https://help.sketchup.com/en/sketchup/performing-silent-install-sketchup (verified 2026-09-29).

| Switch | What it does | Disposition |
|---|---|---|
| `/silent` | Silent installation | Excluded: installs software (a host change) |
| `/FEATURES=<list>` | Chooses features, for example `fr,scan_essentials,revit_importer` | Excluded: same |
| `/INSTALLDIR=<path>` | Chooses the installation folder | Excluded: same |

## 10. Checked, with no official command line

- **Trimble Connect Sync:** GUI only (section 3).
- **Trimble Business Center:** search results mention command-line use, but no official command-line documentation was found (unverified; re-check on the next review).
- **SketchUp, Tekla Structural Designer, Tedds, PowerFab, Trimble Access, CoPilot, PC\*MILER:** automated through SDKs or APIs, not documented command lines. See [capability-matrix.md](capability-matrix.md).
- **Not product command lines:** `@trimble-oss/modus-wc-cli`, a CLI for the Modus design system's web components, and the open-source tools in the `trimble-oss` GitHub organisation (for example DBA Dash). These are not Trimble product interfaces.

Command-line access to Trimble APIs in general goes through `trimblectl` (section 5). `trimblectl api operations` searches every catalogued operation of every product. `trimblectl api read` calls Trimble Connect reads, and `trimblectl api plan` prepares dry-run requests for Trimble Connect changes and other products' reference operations.
