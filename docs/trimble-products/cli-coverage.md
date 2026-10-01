# Command-line coverage

Every command-line interface that Trimble documents or publishes is accounted for here, with each command or switch and its disposition. So is every `trimblectl` command. `internal/skilltest` checks that this page covers every Trimble Connect view and panel and every `trimblectl` command.

Only the Trimble Connect for Windows link (section 1) is supported. The bridge exposes no shell or process-launching tool, handles no credentials, and never changes a host. Every other command below is therefore excluded, with the reason.

The page also lists the app-to-app URL launchers of CoPilot and Trimble Mobile Manager (sections 19 and 20). They are not command lines, but they start and control a local program just as the Connect link does. Where an official page could not be read directly, its entry says so.

## 1. Trimble Connect for Windows command line (supported)

Source: https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line (verified 2026-09-23).

Syntax, as documented: `"trimbleconnect:/projects/[project-id]?show=[view parameter],[panel parameter]"`. Parameters are case-insensitive. The same link can be started from a command line or a browser.

| Surface | MCP tool | CLI |
|---|---|---|
| Build the link (no side effects) | `trimble_build_desktop_link` | `trimblectl desktop link` |
| Open the link on this Windows machine (opt-in, dry run by default) | `trimble_open_in_desktop` | `trimblectl desktop open` |

Every documented combination is supported. Omitting both values makes the bridge emit `3D,models`, a documented combination. The page documents no default: its only example opens ToDos "instead of the models tab", which implies that models is the usual panel. 3D as the default view is [assumption A-9](../requirements/assumptions.md).

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

Other Trimble products document app-to-app URL launchers on a device: CoPilot's `copilot://` URLs (section 19) and Trimble Mobile Manager's Android intents and iOS and Windows URL schemes (section 20). They are excluded there, with their reasons.

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

## 10. SketchUp start-up switches and developer tooling, excluded

Sources (verified 2026-09-29):

- https://github.com/SketchUp/sketchup-ruby-debugger/blob/main/README.md, section "Command-line arguments".
- https://github.com/SketchUp/testup-2/blob/main/README.md, the TestUp 2 continuous-integration examples.
- https://ruby.sketchup.com/file.ReleaseNotes.html, the SketchUp 2018 M0 and SketchUp 7 notes.
- https://github.com/SketchUp/rubocop-sketchup (Ruby gem `rubocop-sketchup` 2.1.1, authors "Trimble Inc, SketchUp Team", RubyGems owners `sketchup` and `thomthom`).

These switches are passed to `SketchUp.exe`, or to the macOS binary `SketchUp.app/Contents/MacOS/SketchUp`. The SketchUp installer is section 9.

**Why it is excluded:** every switch makes SketchUp run Ruby code at start-up, or opens a debugger port that accepts connections from an IDE. With `wait`, SketchUp freezes until an IDE attaches. `rubocop-sketchup` is a developer's static-analysis tool that reads a source tree and writes report files. The bridge runs no local programs and has no filesystem tool.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `SketchUp.exe -rdebug "ide [port=<number>] [wait]"` (Windows, with `SURubyDebugger.dll` copied into the SketchUp folder) | Starts the Ruby debugger listener for a `ruby-debug-ide` IDE. `port` defaults to 1234; `wait` blocks start-up until an IDE attaches | Excluded: opens a debug port through which code runs |
| `/Applications/SketchUp\ 2024/SketchUp.app/Contents/MacOS/SketchUp -rdebug "ide port=6123"`, or `open -a /Applications/SketchUp\ 2024/SketchUp.app --args -rdebug "ide port=6123"` (macOS, with `SURubyDebugger.dylib` in the bundle's Frameworks folder) | The same on macOS | Excluded: same |
| `SketchUp.exe -RubyStartupArg "TestUp:CI:Path: <dir>" > results.json` (macOS: `'/Applications/SketchUp 2023/SketchUp.app/Contents/MacOS/sketchup' -RubyStartupArg '...'`) | Passes a string to Ruby at start-up. TestUp 2 reads it, runs the test suite in `<dir>` and writes JSON results to standard output | Excluded: runs Ruby code (test suites) |
| `SketchUp.exe -RubyStartupArg "TestUp:CI:Config: <Config.yml>"` | TestUp 2 runs the suite described by a YAML file (`Path`, `Output`, `%CONFIG_DIR%`) | Excluded: runs Ruby code; reads and writes local files |
| `-RubyStartup` | Known only from one release note (SketchUp 2018 M0: "Fixed a crash when using `-RubyStartup` command line argument with a file that raises errors while loading"). It evidently loads a Ruby file at start-up. No page documents its syntax | Excluded: runs Ruby code. Its syntax is undocumented, so it is never guessed |
| `Sketchup.exe > myRubyLog.txt` | Sends Ruby console output (`puts`) to standard output, here redirected to a file (SketchUp 7 release notes) | Excluded: local process and file |
| `rubocop --format json --out results.json` | `rubocop-sketchup` (installed with `gem install rubocop` and `gem install rubocop-sketchup`, configured in `.rubocop.yml`) analyses a SketchUp extension's source and writes JSON results | Excluded: developer tooling; reads a source tree and writes files |
| `rubocop -f extension_review -o report.html` | The Extension Review formatter; writes an HTML report | Excluded: same |
| `LayOut -lang <locale>` | Starts LayOut in the given language. Known only from the SketchUp 2024.0 release notes (https://help.sketchup.com/en/release-notes/sketchup-desktop-20240, verified 2026-10-01: "launching LayOut from the command line using the -lang option"); the value format is not documented | Excluded: launches a desktop application; the syntax is never guessed |
| `LayOutExporter` | A LayOut C API SDK sample "command line tool that exports .layout documents to .pdf, .png, or .jpg" (https://extensions.sketchup.com/developers/layout_c_api/layout/index.html, verified 2026-10-01); no switches are documented | Excluded: SDK sample that reads and writes local files |

## 11. Tekla Tedds `TeddsCalcCommand.exe` and `TedToPdf.exe`, excluded

Source: https://support.tekla.com/doc/tekla-tedds/2026/oth_teddscalccommand (verified 2026-09-29; the same page exists for 2024). Tekla describes it as "a utility application which allows the Tedds Calculator to be used in automation scenarios without having to use the Tekla Tedds API". It is in the Tedds program folder, typically `C:\Program Files\Tekla\Structural\Tedds`.

Syntax, as documented: `TeddsCalcCommand.exe -lib <calculation_library_filename> -item <calculation_library_itemname> -varin <input_variables_filename> [-varout <output_variables_filename>] [-ui (enabled/disabled/hidden)]`.

**Why it is excluded:** it runs engineering design calculations on the local machine from local input files and writes the results to local files. With `-ui disabled`, "no input validation will occur", and Tekla warns that the design is completed "regardless of whether that input or the calculated results are valid". Engineering results must be produced and checked by a qualified person. The bridge runs no local programs and has no filesystem tool.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `-lib <calculation_library_filename>` | Required. Full path, in quotes, of the Calc Library file that holds the calculation | Excluded: local file path |
| `-item <calculation_library_itemname>` | Required. Short name of the Calc Item to run from that library | Excluded: runs an engineering calculation |
| `-varin <input_variables_filename>` | Required. Tedds variables XML file with the input variables (exported from Tedds for Word or created with the Tedds API) | Excluded: reads a local file |
| `-varout <output_variables_filename>` | Optional. Tedds variables XML file saved when the calculation finishes | Excluded: writes a local file |
| `-ui enabled` | Shows the calculation's user interface; the user must satisfy every input validation check. This is the default when `-ui` is omitted | Excluded: launches a desktop application |
| `-ui hidden` | Hides the user interface and simulates progress through it; shows it if validation fails | Excluded: same |
| `-ui disabled` | No user interface and no input validation; the design completes whatever the input | Excluded: skips input validation, so the results may be invalid (safety) |

`TedToPdf.exe`, a second Tedds utility that batch-converts Tedds documents (`.ted`) to PDF.

Source: https://support.tekla.com/article/how-can-i-batch-convert-tedds-document-ted-files-to-pdf-files (Tekla Tedds 2026 - 2022; verified 2026-10-01). It is in the Tedds installation folder, typically `C:\Program Files (x86)\Tekla\Structural\Tedds`, needs .NET Core 3.1 or later, and was introduced in Tedds 2020 Service Pack 1. Each PDF is written next to its `.ted` file with the same name. The article says that the C# source is on GitHub (TrimbleSolutionsCorporation/TeddsTedToPdfConverter); that repository was not opened for this page.

Syntax, as documented: `TEDTOPDF [drive:][path][filename] [/R] [/O]`. Started without arguments, it opens a window that asks for a file or folder path.

**Why it is excluded:** it reads local Tedds documents and writes PDF files to local folders, overwriting existing PDFs with `/O`. The bridge runs no local programs and has no filesystem tool.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `TedToPdf.exe` with no arguments | Opens the application, which prompts for the path of a file or folder to convert | Excluded: launches a desktop application |
| `[drive:][path][filename]` | The drive, folder or files to convert | Excluded: local file path; writes PDF files |
| `/R` | If the path is a folder, also converts every file in its child folders | Excluded: same |
| `/O` | Overwrites existing PDF files instead of prompting | Excluded: overwrites local files |

## 12. Tekla installers, licensing and servers, excluded

Sources (verified 2026-09-29):

- Tekla Structures: https://support.tekla.com/article/centralized-distribution-of-tekla-structures-2026, and the page it links for the Switzerland environment, https://support.tekla.com/article/switzerland-environment-installation-using-command-prompt.
- Tekla Structural Designer, Tedds, Tedds for Word, Portal Frame Designer and Connection Designer: https://support.tekla.com/article/is-there-a-way-to-silently-install-tekla-structural-design-products.
- Tekla PowerFab: https://support.tekla.com/article/tekla-powerfab-silent-installation-guide.
- Tekla licence server: https://support.tekla.com/doc/tekla-structures/not-version-specific/lic_installing_tekla_structures_license_server_manually.
- Tekla Structures Multiuser Server: https://support.tekla.com/article/setting-up-multiple-instances-of-tekla-structures-multiuser-server.

Sources added on 2026-10-01 (verified 2026-10-01):

- Earlier Tekla Structures distribution pages, still published: https://support.tekla.com/article/centralized-distribution-of-tekla-structures-2022, https://support.tekla.com/article/centralized-distribution-of-tekla-structures-2020 and https://support.tekla.com/article/centralized-distribution-of-tekla-structures-2018.
- Tekla licence activation: https://support.tekla.com/article/how-to-activate-licenses-using-command-lines-manual-activation.
- Tekla licence deactivation: https://support.tekla.com/doc/tekla-structures/not-version-specific/lic_license_deactivation (the redirect target of the article "How to deactivate licenses using command lines – Manual deactivation").
- Tekla licence borrowing: https://support.tekla.com/article/how-to-borrow-a-license-using-command-lines.
- Tekla licence server post-set-up checks: https://support.tekla.com/doc/tekla-structures/not-version-specific/lic_tekla_on-demand_licensing_post_set-up_checks.
- Tekla PowerFab database repair: https://support.tekla.com/doc/tekla-powerfab/2024i/adm_repair_program_errors_command_line, and the MySQL update procedure: https://support.tekla.com/doc/tekla-powerfab/2025i/adm_update_sql.

**Why it is excluded:** each command installs or removes software, a Windows service or a server, needs administrator rights, and changes the host. Host changes are out of scope for an agent (safety Level 5), as for the Trimble Connect installer in section 2. Some also set licensing, or start a service that listens on a network port. The licence commands activate, return or borrow licences, which changes entitlements at Trimble's activation server or the company licence server. The PowerFab repair command administers the production database with the database administrator's password. The bridge handles no credentials and never changes licensing or a host.

Tekla Structures (the properties go inside `/v"..."` for the `.exe`, or on the `msiexec` command line):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `TeklaStructures2026.exe /s /v"/qn /lvoicewarmupx TS2026_logfile.log"` | Silent installation of the software, with a verbose log. `/s` is silent; `/v` passes the quoted settings to the MSI | Excluded: installs software (a host change) |
| `INSTALLDIR=<path>` | Software installation folder | Excluded: same |
| `TSMODELDIR=<path>` | Default model folder | Excluded: same |
| `CREATELAUNCHER=False` | Does not create the Tekla Launcher shortcut | Excluded: same |
| `INSTALLDEFAULT=true` | Also installs the Default environment | Excluded: same |
| `INSTALLASSISTANT=False` | Does not copy Trimble Assistant for Tekla | Excluded: same |
| `MODELASSISTANT=False` | Does not copy the Tekla Model Assistant package (shown on the service pack installer, 2026 SP2 and later) | Excluded: same |
| `Env_Default_2026.exe /s /v"/qn /lvoicewarmupx TS2026Default_logfile.log"` | Silent installation of the Default environment | Excluded: same |
| `TeklaStructures2026.exe /a"<folder>"` | Administrative extraction of the MSI package; the prerequisites go to `<folder>` | Excluded: prepares a deployment; needs administrator rights |
| `msiexec /i "Tekla Structures 2026.msi" /qn [INSTALLDIR=... TSMODELDIR=...] /lvoicewarmupx <log>` | MSI installation of the software | Excluded: installs software |
| `msiexec /i "Tekla Structures 2026 Default Env.msi" /qn /lvoicewarmupx <log>` | MSI installation of the Default environment | Excluded: same |
| `Env_Switzerland_2021.exe /v"ADDLOCAL=Switzerland_GER,Switzerland_FRA,Switzerland_ITA"` (any subset; `/s /v"/qn ADDLOCAL=..."` for silent) | Installs only the chosen language content of the Switzerland environment | Excluded: same |
| `RUNATTSOPENING=true`, for example `Env_Default_2022.exe /s /v"/qn /lvoicewarmupx TS2022Default_logfile.log RUNATTSOPENING= true "` (environment installers from Tekla Structures 2019 SP1; shown on the 2020 and 2022 pages) | Turns off extraction of the environment's `.tsep` packages at installation time, which is otherwise the default | Excluded: installs software |
| `INSTALL_WAREHOUSE_OFFLINE_CONTENT="Yes"` or `"No"`, for example `TeklaStructures2018.exe /s /v"/qn INSTALLDIR=\"C:\TeklaStructures\" TSMODELDIR=\"C:\TeklaStructuresModels2018\" INSTALL_WAREHOUSE_OFFLINE_CONTENT=\"Yes\" /lvoicewarmupx TS2018_logfile.log"` (Tekla Structures 2018 page) | `Yes` includes the Tekla Warehouse Offline content `.tsep` files, which install the next time Tekla Structures starts; `No`, or no value, leaves them out | Excluded: same |

Tekla Structural Designer, Tedds, Tedds for Word, Portal Frame Designer and Connection Designer:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `msiexec /a <package>.msi` | The administrative installation described in Tekla's Distribution Deployment document | Excluded: prepares a deployment |
| `msiexec /i <package>.msi /qn`, for example `msiexec /i TeklaTedds.msi /qn TEKLA_LICENSE_METHOD=#18 TEKLA_LICENSE_SERVER=licserve01` | Silent installation. The page adds that standard MSI properties such as `INSTALLDIR` also work, without listing them | Excluded: installs software |
| `TEKLA_LICENSE_METHOD=#17`, `#33`, `#18`, `#68` or `#65535` | Licensing method: Local, USB, Server, Tekla Online or Automatic | Excluded: changes licensing configuration |
| `TEKLA_LICENSE_SERVER=<server>` | Name, or static IP address, of the licence server | Excluded: same |

Tekla PowerFab, licence server and Multiuser Server:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `TeklaPowerFab[version].exe /S` | Silent client installation, or silent update of an existing installation | Excluded: installs software |
| `/INSTTYPE=Server` | Server installation. Tekla warns that running it on an existing PowerFab server "could lead to data loss" | Excluded: installs a server; risk of data loss |
| `/D=<directory>` | Custom installation folder; must be the last part of the command | Excluded: installs software |
| `installanchorservice.exe` (run from `%SYSTEMDRIVE%\Tekla\License\Server` in an administrator command prompt) | Installs FlexNet Licensing Service for a manually installed Tekla licence server | Excluded: installs a Windows service |
| `uninstallanchorservice.exe` | Uninstalls FlexNet Licensing Service | Excluded: removes a Windows service |
| `MUSaaS_Install.cmd` (run as administrator from the Multiuser Server folder) | Creates another Tekla Structures Multiuser Server instance: prompts for an identifier (default 2) and a TCP port (default 1239), creates a folder and a Windows service, and starts it | Excluded: creates and starts a network service |
| `MUSaaS_Uninstall.cmd` | Deletes an instance created by `MUSaaS_Install.cmd` | Excluded: removes a service |

Tekla on-premises licence commands (FlexNet utilities; the commands are case-sensitive and may need an administrator command prompt):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `serveractutil -served -commServer http://193.64.145.74:80/flexnet/services/ActivationService?wsdl -activationID <activation ID> -hybrid <count>` (run in the licence server's `Server` folder) | Activates `<count>` licences of an activation ID on the licence server, through Trimble's activation server | Excluded: licence activation; changes entitlements |
| `serveractutil -view` (written `Serveractutil –view`; run in `C:\TeklaStructures\License\Server` by default) | Lists the licences activated on the server, with their fulfilment IDs | Excluded: licensing administration on a server |
| `serveractutil -return <fulfillment_id> -commServer https://activate.tekla.com:443/flexnet/services/ActivationService?wsdl` | Deactivates (returns) a licence to Trimble's activation server; needs internet access | Excluded: licence return; changes entitlements |
| `appactutil.exe -served -commServer 27007@<server_name> -productID NAME=<product>;VERSION=<version> -expiration <dd-Mmm-yyyy>` (run in `C:\TeklaStructures\License\Borrow`) | Borrows a licence from the company licence server until the expiry date. `<product>` is `ProjectViewer`, `Full`, `SteelDetailing`, `Primary`, `Educational`, `ConstructionModeling`, `Drafter`, `Engineering`, `PrecastConcreteDetailing` or `RebarDetailing` | Excluded: licence borrowing; changes entitlements |
| `appactutil.exe -view -long` | Shows the borrowed licences in trusted storage | Excluded: licensing administration on a client |
| `appactutil.exe -return <fulfillment ID> -commServer 27007@<server_name>` | Returns a borrowed licence | Excluded: changes entitlements |
| `tekla_composite.exe` (run from a command prompt on the licence server) | Prints the composite host ID to put in the `SERVER` line of `tekla.lic`. The same page uses the Windows command `hostname` for the server name | Excluded: exposes a licence-binding host identifier; local program |

Tekla PowerFab database maintenance (on the PowerFab server; Tekla says the commands do not work on other workstations):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `cd <MySQL bin folder>` (default `C:\mysql\bin`) | Changes to the MySQL `bin` folder, found in MySQL Service Manager | Excluded with the procedure |
| `mysqlcheck -u admin -p --auto-repair --check --all-databases [--port=<port>]` | Checks every database on the PowerFab MySQL server and repairs corrupt tables; `-p` prompts for the PowerFab database admin password. `--port` may be left out for the default port 3306 | Excluded: database administration with credentials; repairs server databases |
| `uninstall.exe` (in the MySQL installation folder, typically `C:\mysql\`) | Optionally uninstalls the old MySQL version during a MySQL update. The page says to run it and documents no switches | Excluded: removes software from a server |
| `mysql_setup_<version>.exe` (in `C:\Users\Public\Documents\Tekla\Backup` by default, or from Tekla Warehouse) | Installs the new MySQL version; Trimble's `mysql_setup_8.exe` creates a `MySQL_TeklaPowerFab` service on port 3306. The page documents no switches | Excluded: installs software and a database service |

## 13. Tekla extension packaging, excluded

Sources (verified 2026-09-29):

- https://developer.tekla.com/tsep-2025-modernization-tsep-and-signing-process, sections "Signing Recommendation" and "Command line refactorings".
- https://developer.tekla.com/tekla-structures/documentation/tekla-structures-2017-open-api-release-notes.
- NuGet package `AzureTrustedSignTool` (https://www.nuget.org/packages/AzureTrustedSignTool), a dotnet tool owned by `buildmaster_Tekla`, now at version 2.0.0; Tekla's page cites 1.0.0.

The full switch list of the TSEP builder is not public. The 2017 release notes point to the "TSEP documentation in Tekla Open API startup package", and both https://developer.tekla.com/documentation/complete-guide-tsep-creating-tekla-structures-extension-packages and https://developer.tekla.com/documentation/tekla-structures-extension-package-tsep show "This content is available only after signing in". Only the switches below are publicly documented.

**Why it is excluded:** these are build and code-signing tools on a developer's machine. The builder writes `.tsep` packages and logs. The signing tool uses an Azure Trusted Signing account and certificate profile, which are signing credentials. The bridge runs no local programs and handles no credentials.

| Command/Switch | What it does | Disposition |
|---|---|---|
| TSEP batch builder (Tekla's pages call it "Batch builder" and the "command line TSEP builder"; the public pages do not name its executable) | Builds a `.tsep` package from a manifest XML file | Excluded: build tooling; writes files |
| `-o <output path>` | Output path. Optional since TSEP 2025, when the `.tsep` is created next to the manifest; no longer needs a full path | Excluded with the builder |
| `--verbose` | New in TSEP 2025: shows every log message on the console | Excluded with the builder |
| Version options (Tekla Structures 2017) | "Options to modify extension product version, and to append product version to the .tsep file name". Their names are only in the sign-in documentation, so they are not listed | Excluded with the builder |
| `dotnet tool install --global AzureTrustedSignTool` | Installs the signing tool | Excluded: installs software |
| `AzureTrustedSignTool sign --filePath --accountname AccountName --profilename ProfileName` | Signs a `.tsep` with Azure Trusted Signing; it wraps `dotnet/sign` | Excluded: uses code-signing credentials; writes files |

The same page also installs `Knapcode.CertificateExtractor` and `sign --prerelease`, which are third-party tools, not Trimble command lines. The NuGet owner `buildmaster_Tekla` publishes one more dotnet tool, `CxxSonarQubeRunner` 3.8.1. It has no documentation (its description is "Package Description"), so it has no documented command to list, and it is build tooling in any case. That owner's third tool, `teklaenv`, is section 7.

## 14. Trimble Business Center deployment, excluded

Sources (verified 2026-09-29):

- https://help.fieldsystems.trimble.com/tbc/deploy-setupexe.htm (Setup.exe).
- https://help.fieldsystems.trimble.com/tbc/tbc-deployment-options.htm (MSI properties and removal).
- https://help.fieldsystems.trimble.com/tbc/deploy-individual-packages.htm (each package).
- https://help.fieldsystems.trimble.com/tbc/deploy-software-updates.htm (updates and patches).

**Why it is excluded:** these commands install, repair, update or remove software, Windows services and licensing runtimes machine-wide, and need administrator rights. Host changes are out of scope for an agent (safety Level 5). Some packages also install machine-control exporters or configure GNSS receivers for machine control.

TBC's in-application CAD command line (https://help.fieldsystems.trimble.com/tbc/23380.htm, "CAD Command Line Quick Reference Guide") is not an operating-system command line. It is a prompt inside TBC's CAD view, with command aliases such as `BR` (Break Line), `CHA` (Chamfer) and `CL` (Clip Lines), and keyword shortcuts. It belongs to the desktop user interface, which the bridge does not drive, so it is outside this page.

Setup.exe (`setup.exe /<parameter>`; parameters can be combined):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `/silent` | Installs without a user interface | Excluded: installs software (a host change) |
| `/ISCacheDir:<path>` | Package cache folder (default `C:\Programdata\Trimble\Package Cache`) | Excluded: same |
| `/remove` | Uninstalls the installed TBC | Excluded: removes software |
| `/repair` | Repairs the installed TBC | Excluded: changes software |
| `/ISFeatureInstall:<features>` | Comma-separated program features, for example `TBC,CSM` (all features by default) | Excluded: installs software |
| `/language:<lcid>` | Default installation language | Excluded: same |
| `/ISLanguage<Language>:true` | Adds a language; `<Language>` is ChineseSimplified, Czech, Danish, Dutch, EnglishAU, EnglishUK, EnglishUS, Finnish, French, German, Italian, Japanese, Korean, Norwegian, Polish, Portuguese, Russian, Spanish, Swedish or Ukrainian | Excluded: same |
| `/ISInstallDir_TBC:<path>` | TBC installation folder | Excluded: same |
| `/ISDesktop_TBC:""` | No desktop shortcut | Excluded: same |
| `/ISInstallDir_FDM:<path>` | Feature Definition Manager folder | Excluded: same |
| `/ISInstallDir_POS:<path>` | POSPac Command-line TBC Subscription folder | Excluded: same |
| `/ISInstallDir_SDM:<path>` | SCS Data Manager folder | Excluded: same |
| `/ISInstallState_MM:false` | Disables the Mobile Mapping commands | Excluded: same |
| `/ISInstallState_RCP:false` | Disables RCP export for Autodesk ReCap | Excluded: same |
| `/ISInstallState_SDX:false` | Disables GPU rendering of point clouds and imagery | Excluded: same |
| `/ISInstallState_WOV:false` | Does not install Work Order Viewer | Excluded: same |
| `/ISInstallState_FD:false` | Disables export of job-site design files for GCS and SCS. It appears only as the example for the MSI property `FD` | Excluded: same |
| `/ISMachineAddon:true` | Installs the GCS900 12.5 to 12.8 machine-control exporters | Excluded: same; machine-control output |

`TrimbleBusinessCenter.msi <parameter>=<value> /qn` (`/qn` is silent), and Windows Installer commands:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `INSTALLDIR=<path>` | Installation folder | Excluded: installs software |
| `ProductLanguage=<lcid>` | Default language | Excluded: same |
| `ADDLOCAL=<features>` | Comma-separated features. `Trimble_Business_Center`, `Program_Files`, `Trimble_Modules` and `External_Modules` are required, plus `Program_Files_<Language>` for each extra language (Chinese, Czech, Danish, Dutch, EnglishAU, EnglishUK, EnglishUS, Finnish, French, German, Italian, Japanese, Korean, Norwegian, Polish, Portuguese, Russian, Spanish, Swedish, Ukrainian) | Excluded: same |
| `DESKTOP=""` | No desktop shortcut | Excluded: same |
| `FD=false` | Disables job-site design export | Excluded: same |
| `ICM=false` | Does not install the Bentley i-model importers | Excluded: same |
| `MM=false` | Disables the Mobile Mapping commands | Excluded: same |
| `RCP=false` | Disables RCP export | Excluded: same |
| `SDX=false` | Disables GPU rendering | Excluded: same |
| `VC90=true` | Installs the GCS900 12.5 to 12.8 machine-control exporters | Excluded: same; machine-control output |
| `WOV=false` | Does not install Work Order Viewer | Excluded: same |
| `TRIMBLE_SYNCHRONIZER_DATA=<path>` | Sync root folder (default `C:\Trimble Synchronizer Data`) | Excluded: same |
| `msiexec /x <package>.msi /qn`, for example `msiexec /x TrimbleBusinessCenter.msi /qn` | Silent removal | Excluded: removes software |
| `msiexec /i <package>.msi /qn` | Deploys a full update package saved from Help > Check for Updates | Excluded: installs software |
| `msiexec /p <package>.msp /qn` | Deploys a patch package | Excluded: changes software |

Individual packages, in the recommended installation order, with each command line as documented. All are excluded because they install software (a host change).

| Command/Switch | What it does | Disposition |
|---|---|---|
| `ndp48-x86-x64-allos-enu.exe /q /norestart` | Microsoft .NET Framework 4.8 (required) | Excluded: installs software |
| `ndp48-x86-x64-allos-<iso>.exe /q /norestart` | .NET Framework 4.8 language pack (required) | Excluded: same |
| `VC_redist.x86.exe /q /norestart` | Visual C++ v14 Redistributable, x86 (required) | Excluded: same |
| `VC_redist.x64.exe /q /norestart` | Visual C++ v14 Redistributable, x64 (required) | Excluded: same |
| `vcredist_x64.exe /q /rorestart` (sic) | Visual C++ 2013 Redistributable, x64, for POSPac (required) | Excluded: same |
| `vcredist_x86.exe /q /rorestart` (sic) | Visual C++ 2008 Redistributable, x86, for the GCS900 exporters (optional) | Excluded: same |
| `vstor_redist.exe /q:a /c:"install /q /l"` | Visual Studio 2010 Tools for Office Runtime (required) | Excluded: same |
| `SQLSysClrTypes.msi /qn` | SQL Server 2014 System CLR Types, for POSPac (optional) | Excluded: same |
| `ReportViewer.msi /qn` | Microsoft Report Viewer 2015, for POSPac (optional) | Excluded: same |
| `HASP_Setup.msi /qn` | Sentinel HASP runtime (required) | Excluded: same; licensing runtime |
| `SentinelHASPVendorLibrary.msi /qn` | Trimble HASP vendor library (required) | Excluded: same |
| `OfficeComponents.msi ProductLanguage=<lcid> /qn` | Office Shared Components (required) | Excluded: same |
| `IfcPlugin-<version>_x64.msi /qn` | IFC plug-in (optional) | Excluded: same |
| `ANZToolbox_<version>.msi /qn` | ANZ Toolbox (optional) | Excluded: same |
| `FAROLS.msi /qn` (the package is listed as `FARO.LS.msi`) | FARO LS point-cloud import (optional) | Excluded: same |
| `CoordinateSystemManager.msi ProductLanguage=<lcid> /qn` | Coordinate System Manager (required) | Excluded: same |
| `ConvertToRinex_v<version>.msi /qn` | Convert to RINEX (optional; its own command line is section 15) | Excluded: same |
| `ConfigurationToolbox.msi /qn` | Configuration Toolbox, which configures Trimble GNSS receivers for machine control (optional) | Excluded: same; machine-control configuration |
| `TensorFlowGPU.msi /qn` | TensorFlow GPU (optional) | Excluded: same |
| `ExternalServiceAPI.msi /qn` | Trimble External Service API, a Windows service host (optional) | Excluded: same; installs a service |
| `Trimble.Scs.LocalDatasetWorkspaces.API.Installer.msi /qn` | Trimble Local Dataset Workspaces API (optional) | Excluded: same |
| `Trimble.Scs.Activities.API.Installer.msi /qn` (the package is listed as `Trimble.Scs.Activities.installer.msi`) | Trimble Activities API (optional) | Excluded: same |
| `TrimbleDesktopUtility.msi /qn` | Trimble Desktop Utility, which provides authentication services to Trimble services on the device (optional) | Excluded: same; credential service |
| `Trimble.Scs.LocalDatasetWorkspaces.Synchronization.Installer.msi /qn` | Synchronises dataset workspaces from the device to the cloud (optional) | Excluded: same |
| `Trimble.Scs.Activities.Runner.Installer.msi /qn` | Trimble Activities Runner (optional) | Excluded: same |
| `TrimbleBusinessCenter.msi <parameter> /qn` | Trimble Business Center (required; properties above) | Excluded: same |
| `TBCHelpEnglish.msi /qn`, `TBCHelpFrench.msi /qn`, `TBCHelpGerman.msi /qn`, `TBCHelpJapanese.msi /qn`, `TBCHelpSpanish.msi /qn` | Offline help (optional) | Excluded: same |
| `FeatureExtractionEngine.msi /qn` | Feature Extraction Engine (required) | Excluded: same |
| `FeatureExtractionRulesets.msi /qn` | Feature Extraction Rulesets (required) | Excluded: same |
| `pctcore.msi /qn` | Point Cloud Training Environment, Core (optional) | Excluded: same |
| `pctcuda.msi /qn` | Point Cloud Training Environment, CUDA (optional) | Excluded: same |
| `tmxfilter.msi /qn` | TMX Filter for the MX50 and MX60 mobile mapping systems (optional) | Excluded: same |
| `FeatureDefinitionManager.msi ProductLanguage=<lcid> INSTALLDIR=<path> /qn` | Feature Definition Manager (optional) | Excluded: same |
| `SCSDataManager.msi ProductLanguage=<lcid> INSTALLDIR=<app path> TRIMBLE_SYNCHRONIZER_DATA=<sync path> WOV=false /qn` | SCS Data Manager (optional) | Excluded: same |
| `UASMaster-<version>.msi APPLICATIONFOLDER=<path> /qn` | UASMaster (optional) | Excluded: same |
| `UASMaster-AddOn<version>.msi APPLICATIONFOLDER=<path> /qn` | UASMaster Add-On (optional) | Excluded: same |
| `POSPacCommandLineTBCSubscription.msi INSTALLDIR=<path> /qn` | POSPac Command-Line TBC Subscription; needs MATLAB Runtime R2024b (optional) | Excluded: same |
| `HaspLicenseUpdater.msi ProductLanguage=<lcid> /qn` | HASP License Updater (optional) | Excluded: same; licensing |
| `Cleaner_x64.msi /qn` | Office Cleanup, which restores a clean state for reinstallation (optional) | Excluded: same |

## 15. Trimble Convert to RINEX, excluded

Source: "Trimble Convert to RINEX Utility Release Notes", version 1.0.1.32, September 2008 (Revision A), PDF author "Trimble Navigation Limited", section "Command line operation". The copy found is hosted by UNAVCO (now EarthScope), not by Trimble: https://kb.unavco.org/file.php?id=65 (verified 2026-09-29). It is 18 years old and third-party hosted, so current versions may differ. TBC still deploys the utility as `ConvertToRinex_v<version>.msi /qn` (section 14), but TBC's own help for it (https://help.fieldsystems.trimble.com/tbc/17261.htm) describes only the dialog.

Syntax, as documented: `convertToRINEX input.dat [-p outputPath] [-r RUNBY] [-o OBSERVER] [-ag AGENCY] [-ac ANTENNACODE] [-an ANTENNANUMBER] [-h HEIGHT] [-n] [-rc RECEIVERCODE] [-rn RECEIVERNUMBER] [-mo MARKERNAME] [-mn MARKERNUMBER] [-v X.XX] [-k] [-d] [-s] [-m] [-t] [-c] [-g] [-co]`, or `convertToRINEX @ParamFile`.

**Why it is excluded:** it converts local GNSS measurement files (DAT or T01) into RINEX files on disk. The bridge runs no local programs and has no filesystem tool. Antenna-height and clock-offset options also change survey observations, which a qualified person must check.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `convertToRINEX` with no arguments | Runs in interactive mode | Excluded: local program |
| `input.dat` | The DAT or T01 file to convert | Excluded: local file path |
| `-?` | Displays help | Excluded with the tool |
| `-p outputPath` | Folder for the output files (OBS, NAV, MET and others) | Excluded: writes local files |
| `-r RUNBY` | Person or agent performing the conversion | Excluded with the tool |
| `-o OBSERVER` | Person performing the survey | Excluded with the tool |
| `-ag AGENCY` | Observer's agency | Excluded with the tool |
| `-v X.XX` | Output RINEX version (default 2.11) | Excluded with the tool |
| `-ac ANTENNACODE` | One- or two-character antenna code, from `ANTENNA.INI` | Excluded: changes survey metadata |
| `-an ANTENNANUMBER` | Value for the NUMBER field of "ANT # / TYPE" | Excluded: same |
| `-h HEIGHT` | Adds HEIGHT, in metres, to all antenna heights | Excluded: changes observations |
| `-n` | Skips the default antenna-height corrections | Excluded: changes observations |
| `-k` | Marks the first observation as kinematic | Excluded: same |
| `-ca` | Accounts for millisecond time steps in observations and time of observation (described, but not in the usage line) | Excluded: same |
| `-co` | Includes receiver clock offsets in the OBS output | Excluded: same |
| `-d` | Includes Doppler values | Excluded with the tool |
| `-s` | Includes raw signal strengths | Excluded with the tool |
| `-g` | Includes only GPS observations | Excluded with the tool |
| `-m` | Writes a RINEX meteorological file if data are present | Excluded: writes local files |
| `-t` | Writes a RINEX auxiliary file with tilt data if present | Excluded: writes local files |
| `-mo MARKERNAME` | Value for the MARKER NAME field | Excluded with the tool |
| `-mn MARKERNUMBER` | Value for the MARKER NUMBER field | Excluded with the tool |
| `-rc RECEIVERCODE` | Receiver numeric code, from `RECEIVER.INI` | Excluded with the tool |
| `-rn RECEIVERNUMBER` | Value for the NUMBER field of "REC # / TYPE / VER" | Excluded with the tool |
| `-c` | In the usage line, with no description | Excluded; its meaning is undocumented |
| `@ParamFile` | Reads `<ParameterName> = <Value>` lines instead of switches (`#` starts a comment). Parameters: `DatFile`, `ObsFile`, `NavFile`, `MetFile`, `RunBy`, `Observer`, `Agency`, `RinexVersion`, `GenMetFile`, `AntCode`, `AntNumber`, `AntType`, `CorrToBase`, `AntCorrect`, `AntOffset`, `RcvrCode`, `RcvrNumber`, `RcvrType`, `RcvrVersion`, `HdrMarkerName`, `HdrMarkerNumber`, `HdrMarkerXYZ`, `KinStart`, `LogDoppler`, `LogSNR`, `LogOnlyGPS`, `AddMSecOffsets`, `AddRcvrOffsets`, `AdjDisabled`, `AdjObservations` | Excluded: reads a local file |

## 16. Trimble eCognition command-line tools, excluded

Sources (verified 2026-09-29):

- https://docs.ecognition.com/eCognition_documentation/Other%20Resources/eCognition%20SDK/Automation%20API/1.5%20Introduction%20to%20Remote%20Automation.htm ("Introduction to Remote Automation", eCognition 10.5). It says: "Provided are the DIACmdClient.exe, the DIACmdEngine.exe and DIAMkWksp tools. While these are not a programming interface, but command line tools, they do utilize the SOAP protocol interface".
- Docker image `ecognition/linux_cle`, "Trimble eCognition Command-line engine for Linux": https://hub.docker.com/r/ecognition/linux_cle (updated 2026-09-01).
- The other seven images of the official `ecognition` Docker Hub namespace (https://hub.docker.com/v2/repositories/ecognition/ lists eight; verified 2026-10-01): https://hub.docker.com/r/ecognition/win_cle, https://hub.docker.com/r/ecognition/linux_js, https://hub.docker.com/r/ecognition/win_js, https://hub.docker.com/r/ecognition/linux_es, https://hub.docker.com/r/ecognition/win_es, https://hub.docker.com/r/ecognition/linux_es_cuda and https://hub.docker.com/r/ecognition/linux_ls.

**Why it is excluded:** these tools run image-analysis rule sets, which are code, on local or server engines. They read and write local image, project and workspace files, and submit jobs to an eCognition Server. `DIACmdClient -db user[:pwd]@storage` puts a password on the command line, where other local processes can see it. The bridge runs no local programs, has no filesystem tool and handles no credentials. The SOAP interface these tools use is a separate API and is not covered here.

`DIACmdEngine`, the engine's command-line interface:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `DIACmdEngine image=<path> [image=<pathN>...] [thematic=<path>] ruleset=<path> [options]` | Analyses raster or point-cloud files (`.tif`, `.asc` and others), with optional thematic data (`.shp`, `gdb` and others), using a rule set (`.dcp`) | Excluded: runs a rule set on local files |
| `DIACmdEngine image-dir=<path> import-connector=<name> [import-connector-file=<path>] [image=<extra>] [thematic=<extra>] ruleset=<path> [options]` | Analyses data imported with a predefined or custom (`.xml`) import connector. The parameter list calls the root folder `import-dir` | Excluded: same |
| `DIACmdEngine dpr=<path> ruleset=<path> [options]` | Analyses an existing project (`.dpr`) | Excluded: same |
| `DIACmdEngine image-dir=<path> scene-xml=<path> ruleset=<path> [options]` | Analyses several scenes from a scene file list in one run | Excluded: same |
| `DIACmdEngine --update-ruleset <input_ruleset_path> <output_ruleset_path>` | Re-saves a rule set so that it uses the latest algorithm versions | Excluded: writes local files |
| `param:<name>=<value>` | Sets a scene variable of the rule set; repeatable | Excluded with the engine |
| `array-param:<name>=<value1>,<value2>,...` | Sets a rule-set array; repeatable | Excluded with the engine |
| `output-dir=<path>` | Output folder for exports | Excluded: writes local files |
| `license-token=<json>` | Additional licence information, in JSON | Excluded: licence data on the command line |
| `save-dpr[=<path>]` | Saves the project file, by default to `{:Workspc.OutputRoot}\dpr\{:Project.Name}.v{:Project.Ver}.dpr` | Excluded: writes local files |
| `--pause` | Pauses the application when done | Excluded with the engine |
| `--map <path1>=<path2>` | Maps a local drive to a network path | Excluded with the engine |
| `--log-file=<file>` | Log file path (default from `config/eCognition.cfg`) | Excluded: writes local files |

`DIACmdClient`, which submits workspaces to the eCognition Job Scheduler and monitors them. Syntax: `DIACmdClient action [options] workspace_file [ruleset_file] [scene_name]`, or `DIACmdClient action [options] -db user[:pwd]@storage workspace_id [ruleset_id]`. The page's examples write the action with a leading dash, for example `DIACmdClient -sw test1.dpj fastrule.dcp`.

| Command/Switch | What it does | Disposition |
|---|---|---|
| action `s` | Submits the workspace for analysis | Excluded: submits server jobs |
| action `p` | Submits the workspace for stitching | Excluded: same |
| action `w` | Waits for the workspace to finish | Excluded with the client |
| action `t` | Tests the state of the analysis | Excluded with the client |
| action `r` | Rolls back the workspace and deletes results | Excluded: deletes data |
| action `d` | Deletes a single run, with its results | Excluded: deletes data |
| action `sw` | Submits for analysis and waits | Excluded: submits server jobs |
| action `pw` | Submits for stitching and waits | Excluded: same |
| `-p` | Analyses tiles only | Excluded with the client |
| `-top` | Analyses top scenes only | Excluded with the client |
| `-u <url>` | Job Scheduler URL | Excluded with the client |
| `-t <sec>` | Maximum wait for the Job Scheduler to start | Excluded with the client |
| `-run <name>` | Run name | Excluded with the client |
| `-fsr` | Forces submission even if other runs have not finished | Excluded with the client |
| `-scn <file>` | Submits only the scenes listed in the file | Excluded: reads a local file |
| `-ro` | Read only: does not modify the workspace | Excluded with the client |
| `-db` | Opens the workspace from Data Management storage instead of a file, with `user[:pwd]@storage` and IDs in place of file names | Excluded: password on the command line |
| `-auth <url>` | Authentication Server URL, which checks the `-db` user name and password | Excluded: credential handling |

`DIAMkWksp` and `DIAClient.exe`:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `DIAMkWksp wksp_file [input_fldr] [import_tmplt_name] [export_tmplt_file] [add_tmplt_fldr]` | Creates a workspace (`.dpj`), importing every image file found recursively in `input_fldr`, with an optional import template (defined in `Default.scm`), export template file and folder of extra import templates | Excluded: reads and writes local files |
| `DIAClient.exe /image <file>` | Starts the client with an image loaded | Excluded: launches a desktop application |
| `DIAClient.exe /ruleset <file>` | Starts the client with a rule set loaded | Excluded: same |
| `DIAClient.exe /project <dpr>` | Starts the client with a project loaded | Excluded: same |
| `DIAClient.exe /product <name>` | Product to start, for example `"eCognition Developer"` | Excluded: same |
| `DIAClient.exe /portal <name>` | Portal to start, for example `"Rule Set Mode"` | Excluded: same |
| `DIAClient.exe /workspace <dpj>` | Starts the client with a workspace loaded | Excluded: same |

Docker image `ecognition/linux_cle` (it also contains `DIAMkWksp` and `DIACmdClient`, for use with the Job Scheduler image `ecognition/linux_js`):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `docker run -it -e "LM_LICENSE_FILE=@<licence server address>" -v <data>:/mnt ecognition/linux_cle:10.5.0` | Runs the Linux command-line engine interactively; needs an eCognition Server licence from the eCognition License Server | Excluded: runs containers; licensing |
| `docker run ... ecognition/linux_cle:<tag> ./DIACmdEngine <arguments>` | Runs one analysis with the `DIACmdEngine` syntax above; returns 0 on success and -1 otherwise | Excluded: runs a rule set on local files |
| `-e ECOG_CONFIG_logging="log path=/mnt/logs;trace level=Detailed"` | Logging configuration | Excluded with the container |

The other eCognition Docker images. Every image reads the licence server from `LM_LICENSE_FILE=@<licence server address>`, and accepts any `config/eCognition.cfg` setting as `ECOG_CONFIG_{config-group}="key1=value1;key2=value2;.."`. The GRID images (Job Scheduler and Engine Server) are needed only for rule sets that submit, copy, subset, tile or delete scenes, or read subscene statistics.

**Why these are excluded:** they start long-running network services (the Job Scheduler listens on TCP 8184 with a web interface), run rule sets, which are code, or, for the licence server, run a privileged container that activates licences at Trimble's activation server from an activation ID and returns them on stop. A `docker container kill` loses the activated licences permanently. The bridge runs no containers or local programs and never changes licensing.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `docker run -it -e "LM_LICENSE_FILE=@<licence server address>" -v C:\path\to\mydata:c:\mnt ecognition/win_cle:10.4.0` | Windows command-line engine (Windows Server 2019 base) run interactively; `DIACmdEngine.exe` is available inside | Excluded: runs containers; licensing |
| `docker run -e "LM_LICENSE_FILE=@..." -v ... ecognition/win_cle:10.4.0 DIACmdEngine.exe image-dir=... import-connector=... [import-connector-file=...] ruleset=... output-dir=...` | Runs one analysis with the `DIACmdEngine` syntax above (the readme writes the save option as `--save-dpr`) | Excluded: runs a rule set on local files |
| `docker run --rm -d --name ecog_linux_js -p 8184:8184/tcp --ip=172.28.5.1 -e LM_LICENSE_FILE=@<licence IP> --network=grid_network -m 8g --cpus=2 ecognition/linux_js:10.5.0` | Starts the GRID Job Scheduler, which manages the job queue and serves a web interface on port 8184. One per GRID | Excluded: starts a network service |
| `-e ECOG_CONFIG_DiaJobScheduler_general="database folder=<folder>;database file prefix=<prefix>"` | Places the Job Scheduler's file-based job database on a shared volume | Excluded: writes files; service configuration |
| `docker network create -d bridge --subnet=172.28.0.0/16 --gateway=172.28.5.254 grid_network` (the Engine Server readme writes `-d nat`) | Creates the Docker network that the readmes assume | Excluded: changes host networking |
| `ecognition/win_js` | The Job Scheduler on a Windows Server 2019 base; its readme refers to the Linux readme | Excluded: starts a network service |
| `docker run --rm -d --name ecog_linux_es -e JOB_SCHEDULER=<Job Scheduler IP>:8184 -e LM_LICENSE_FILE=@<licence IP> --network=grid_network -m 8g --cpus=2 ecognition/linux_es:10.5.0` | Starts a GRID Engine Server, which connects to the Job Scheduler and processes its jobs (no point clouds on Linux) | Excluded: starts a processing service that runs rule sets |
| `-v /myShare:/mnt -e ECOG_CONFIG_logging="log path=/mnt/logs;trace level=Detailed"` | Optional shared folder and logging for the Job Scheduler and Engine Server | Excluded with the container |
| `ecognition/win_es` | The Engine Server on a Windows Server 2019 base; it can also process point clouds. Its readme refers to the Linux readme | Excluded: same |
| `ecognition/linux_es_cuda` | The Linux Engine Server with the CUDA build of TensorFlow; its readme refers to the Linux readme | Excluded: same |
| `docker run --privileged -d -e container=docker -e VENDOR_PORT=<vendor port> -e ECOGNITION_ACTIVATION_ID=<activation ID> -e NR_LIC=<number of hybrid licences> -p <vendor port>:<vendor port> -p 27000-27009:27000-27009 -v /sys/fs/cgroup:/sys/fs/cgroup ecognition/linux_ls:10.5.0` | Starts the eCognition License Server. On start it activates `NR_LIC` licences of the activation ID over the internet; `--privileged` is required for activation and return | Excluded: licence activation; privileged container; network service |
| `-p 8090:8090` | Exposes the licence server's web console | Excluded: network service |
| `docker container stop <container>` | Stops the licence server and returns (deactivates) its licences | Excluded: licence return |
| `docker logs <container>` (optionally after `--name <name>` on `docker run`) | Shows whether activation or return succeeded | Excluded with the container |

## 17. PC\*MILER BatchPro, Rail-BatchPro, installer, Connect TCP/IP and AS/400 server, excluded

Sources (official Trimble Maps PC\*MILER support articles):

- https://support.pcmiler.com/en/support/solutions/articles/19000076345-using-batchpro-from-the-command-line, also in the Trimble Transportation Learning Center help: https://learn.transportation.trimble.com/wp-content/uploads/tte/ebcbe19c93c746dd320c/olhlp/1a04aee3fa04/docs/Current/PCMilerBatchPro/3.13.10-UsingBatchProfromthe.html.
- https://support.pcmiler.com/en/support/solutions/articles/19000087469-using-batchpro-rail-from-the-command-line.
- https://support.pcmiler.com/en/support/solutions/articles/19000053641-is-there-a-command-line-option-available-for-pc-miler-rail-batchpro-.
- https://support.pcmiler.com/en/support/solutions/articles/19000053111-installing-pc-miler-silent-installation-.

**Not read directly.** On 2026-09-29, support.pcmiler.com and www.pcmiler.com returned HTTP 403 to automated clients, and learn.transportation.trimble.com redirected to its home page. The BatchPro, Rail-BatchPro and installer entries in the first table below were confirmed from search-engine extracts of these official pages. They need re-reading from a network that can open them, and the switch list may be incomplete until then.

**Why it is excluded:** BatchPro and Rail-BatchPro are local batch programs that read input and configuration files and write output reports. `setup.exe` installs software (a host change). The bridge runs no local programs and has no filesystem tool. PC\*MILER Web Services is classified separately in [capability-matrix.md](capability-matrix.md).

| Command/Switch | What it does | Disposition |
|---|---|---|
| `batchw32.exe` | Runs BatchPro without its user interface, processing the input file with the settings in `pcmbatch.cfg`; can be called from a scheduler, another application or a batch file | Excluded: local program; reads and writes files |
| `CommandLine=1` in `pcmbatch.cfg` | Turns on command-line mode (the default is 0). The `.cfg` file is by default in the same folder as `batchw32.exe` | Excluded with the program |
| `batchw32.exe <file>.cfg` (a file name or a full path) | Runs BatchPro with an alternate configuration file | Excluded: same |
| `batchrailcmd.exe "-input:<path>\railbatch.in" "-config:<path>\railbatch.cfg"` (one space between the arguments; in the PC\*MILER Rail `App` folder) | Runs Rail-BatchPro without a user interface. It writes the `.OUT` file next to the `.IN` file, and reports the number of entries processed or an error | Excluded: same |
| `"-output:<name>"` | Optional base name of the output file, without `.OUT` | Excluded: writes local files |
| `setup.exe -r` | Runs the installer and records the choices in `setup.iss`, in the Windows folder (copy it next to `setup.exe`) | Excluded: installs software |
| `setup.exe /s` | Installs silently with the recorded choices | Excluded: same |

PC\*MILER Connect TCP/IP interface and the PC\*MILER-AS/400 Distance Server.

Sources (verified 2026-10-01; both pages could be read directly):

- https://developer.trimblemaps.com/pcmiler/connect/workflows/the-tcp-ip-interface/ ("The TCP/IP Interface").
- https://developer.trimblemaps.com/pcmiler/connect/as400/tolls/as-400-tolls-integration-technical-implications/ ("AS/400 Tolls: Technical Implications").

`pcmsock.exe`, the TCP/IP interface program, and `tcpsvc.exe`, its Windows-service form, are installed in `C:\ALK Technologies\PCMILERXX\tcpip`. They need PC\*MILER Connect. Syntax, as documented: `pcmsock [product code] [port number] [dataset code]`, with an optional thread count. The same parameters are the service's start parameters. The interface is plain text: a client that connects receives a prompt ending in `READY`, and can then call every applicable PC\*MILER Connect function.

**Why these are excluded:** `pcmsock.exe` and `tcpsvc.exe` open a TCP port on the host that exposes the PC\*MILER Connect routing engine to the network, and `SRV32.exe` is a local server program that services AS/400 data queues. Starting listeners or services changes the host. The bridge runs no local programs. PC\*MILER Connect is a local DLL product, not a web API.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `pcmsock PC_MILER <port>`, for example `pcmsock PC_MILER 2001` | Starts the TCP/IP server for PC\*MILER Connect (product code `PC_MILER`), listening on a unique port. The port is required | Excluded: starts a network listener; local program |
| `[dataset code]`, for example `pcmsock PC_MILER 2001 NA` | Optional region: `NA` North America, `AF` Africa, `AS` Asia, `EU` Europe, `ME` Middle East, `OC` Oceania, `SA` South America. It overrides the default region in `pcmserve.ini` | Excluded with the server |
| `[threads]`, for example `pcmsock PC_MILER 2001 4` or `pcmsock PC_MILER 2001 NA 4` | Number of threads (default 64) | Excluded with the server |
| `tcpsvc.exe` with the same start parameters | Runs the TCP/IP interface as a Windows service | Excluded: installs and runs a network service |
| `Pcmsock25.exe PC_MILER 8250 LOG_BASIC .\V25log.txt` (also `Pcmsock20.exe PC_MILER 8200`) | A logging form with versioned executable names. **Not read directly:** it comes only from search-engine extracts of the "MVS for Windows" support article 19000077351 on support.developer.trimblemaps.com, which returned HTTP 403 on 2026-10-01 | Excluded: same; writes a log file |
| `tcptest.exe` (with the sample trip `trip.txt`) | Test client that sends commands to a running server | Excluded: local program; network client |
| `simple.pl` | Sample Perl 5 client for Unix, VMS or other hosts | Excluded: same |
| `telnet 127.0.0.1 <port>` | Connects to the text interface to test the installation | Excluded: network client |
| `SRV32_hwy.exe 2`, for example `C:\ALK Technologies\pcmiler31\as400\SRV32_hwy.exe 2`, set as the command-line parameter of a shortcut to `Srv32.exe` | Runs a Standard Highway and a Tolls Distance Server side by side, after `pcmserve.ini` and `pcmsrv32.dll` are copied to `pmwssrv.ini` and `pmwssrv.dll` and the copy's `Library=` is switched between `ALKWIN` and `ALKTLL` | Excluded: local server program that services AS/400 data queues |

## 18. Viewpoint Vista client installer, excluded

Sources (verified 2026-09-29):

- Trimble Help: https://help.trimble.com/doc/vista/vista/on-premises-deployments/installations/major-release-installation/vista-client-application/install-the-vista-client-on-a-workstation/silent-installation. It was read with a browser-like fetch; plain `curl` receives HTTP 202 with an empty body.
- Trimble's ERP Cloud FAQ site for Vista and Spectrum: https://sites.google.com/trimble.com/vista-cloud-faq/home/vrl-explained/pre-configure-vista-vrl-client (updated 21 August 2024) and https://sites.google.com/trimble.com/vista-cloud-faq/home/vrl-explained/vista-command-line-silent-uninstall (updated 4 May 2021).

**Why it is excluded:** these commands install or remove the Vista client and its prerequisites on workstations, which is a host change needing administrator rights (safety Level 5).

| Command/Switch | What it does | Disposition |
|---|---|---|
| `Vista_Client_2#.##.##.msi /qn` | Silent client installation (Trimble Help) | Excluded: installs software |
| `msiexec.exe /S /v /qn /i "C:\Temp\VistaClient.23.1.0.883.msi" INSTALLINGBITNESS=0` | Silent 64-bit client installation. The FAQ explains `/s` as silent, `/v` as verbose and `/qn` as "additional options" | Excluded: same |
| `msiexec.exe /S /v /qn /i "C:\Temp\VistaClient.23.1.0.883.msi"` | Silent 32-bit client installation (no `INSTALLINGBITNESS`) | Excluded: same |
| `ndp48-x86-x64-allos-enu.exe /quiet /norestart /AcceptEULA` (`/passive` is also accepted) | .NET Framework 4.8 prerequisite, `.exe` version | Excluded: same |
| `msiexec /i ndp48-x86-x64-allos-enu.msi /quiet /norestart /log "<file>"` | .NET Framework 4.8 prerequisite, Microsoft's `.msi` version | Excluded: same |
| `msiexec /i CRRuntime_64bit_13_0_21.msi /qn` | SAP Crystal Reports Runtime 13 prerequisite | Excluded: same |
| Silent uninstall with `msiexec` and the product GUIDs | The FAQ finds the Vista and Crystal Runtime GUIDs with `get-wmiobject Win32_Product \| Sort-Object -Property Name \|Format-Table IdentifyingNumber, Name, LocalPackage -AutoSize`, then says to run an `msiexec` command for each. The published page does not show that command | Excluded: removes software. The command is not shown, so it is not reconstructed |

## 19. CoPilot URL launch (not a command line), excluded

Sources (verified 2026-09-29):

- https://developer.trimblemaps.com/copilot-navigation/feature-guide/advanced-features/url-launch2/ ("URL Launch (Single URL)", CoPilot 10.26.1.345 and later).
- https://developer.trimblemaps.com/copilot-navigation/feature-guide/advanced-features/url-launch/ ("URL Launch (Multiple URLs)", last updated 9 February 2026).

CoPilot's URL launch lets another application on the same device "launch and manage CoPilot". Like `trimbleconnect:`, it is a URL scheme rather than a command line, and it is listed because it starts and controls a local program.

**Why it is excluded:** it controls a truck-navigation app on a driver's device. It activates licences, sets the vehicle and driver identity, changes configuration and restarts CoPilot, selects the vehicle routing profile that CoPilot uses "to generate a safe and legal route", and replaces the trip's stops. A wrong profile or stop changes a moving vehicle's route, so a person responsible for the dispatch must send it (safety: vehicle guidance). The bridge also cannot reach a driver's device, and a product key in a URL is a credential.

Single URL (`copilot://options?type=TASKS&JSON=<Base64-encoded JSON>`), with its documented JSON fields:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `copilot://options?type=TASKS&JSON=<Base64 JSON>` | Runs several tasks from one Base64-encoded JSON object | Excluded: device-local app control |
| `Ams`: `AssetID` and `CompanyID` (direct customers), or `AssetID`, `ExternalAccountID` and `PartnerID` (partners reselling CoPilot) | Activates an Account Manager licence | Excluded: licence activation |
| `FleetPortal`: `DeviceID`, `PartnerID`, `DriverID`, `EnableCompliance`, `EnableFleetPortal`, `ForceSync` | Connects to Account Manager Fleet Settings; `ForceSync` syncs them without a restart | Excluded: changes device configuration |
| `Trip.RoutingProfile` (`Name`) | Sets the active vehicle routing profile | Excluded: vehicle routing (safety) |
| `Trip.Stops[]`: `Location` (`Label`; `Address` with `StreetAddress`, `City`, `State`, `Zip`, `County`, `Country`; `Coords` with `Lat`, `Lon`; `CustomPlaceID`), `StopType` (`Waypoint`), `PlannedDuration`, `EarliestArrivalTime`, `LatestArrivalTime`, `AtRiskThreshold` | Sends one or more stops. The current GPS location is inserted as, or replaces, the first stop | Excluded: changes a driver's route (safety) |
| `Trip.ExternalTripId` | The caller's trip identifier, for RouteReporter compliance review | Excluded with the trip |
| `ShowConfirmation` | Shows a confirmation message in CoPilot | Excluded with the URL |

Multiple URLs, one task per URL:

| Command/Switch | What it does | Disposition |
|---|---|---|
| `copilot://options?type=CONFIG&CompanyID=<id>&AssetID=<id>&showconfirmation=true` | Activates a licence (direct customers) | Excluded: licence activation |
| `copilot://options?type=CONFIG&PartnerID=<id>&ExternalAccountID=<id>&AssetID=<id>&showconfirmation=true` | Activates a licence (partners reselling CoPilot) | Excluded: same |
| `copilot://options?type=Activation&ProductKey=<key>` | Activates a licence with a product key; must be sent on its own | Excluded: licence activation; a product key in a URL |
| `copilot://options?type=CONFIG&VehicleID=<id>&DriverID=<id>` | Identifies the vehicle and driver | Excluded: changes device configuration |
| `copilot://options?type=CONFIG&<ConfigName>=<ConfigValue>&...` | Sets any `user.cfg` or CPIK configuration setting, for example `FlowTrafficEnabled` or `FLOW_TRAFFIC_AVAILABILITY`, `DownloadDataOnSDCard` and `SayWelcome` | Excluded: same |
| `showconfirmation=true` or `SETTINGS_CONFIRMATION_DIALOG=true` | Confirms that the settings were applied | Excluded with the URL |
| `RestartCoPilot=true` or `RESTART_COPILOT=true` | Restarts CoPilot after the settings are applied | Excluded: restarts a navigation app |
| `EnableCustomButton=true` and `AppLaunchBundleID=<bundle id>` | Adds a button that returns to the launching app (Android) | Excluded: changes device configuration |
| `copilot://options?type=VehicleProfile&SelectByName=<name>` | Selects the vehicle routing profile; must be sent on its own | Excluded: vehicle routing (safety) |
| `copilot://options?type=STOPS&stop=name\|address\|city\|zip\|juris\|state\|lat\|long&stop=...` (optional `&ExternalTripId=<id>`) | Replaces the trip's stops with fielded stops | Excluded: changes a driver's route (safety) |
| `copilot://options?type=STOPS&query=<name>\|<search string>` | Adds stops found by a single-string search, which uses a web service | Excluded: same |
| `copilot://options?type=STOPS&query=<customPlaceID>&IncludeCustomPlaceIdOnly=true` | Adds stops by the company's Place ID | Excluded: same |
| `copilot://mydestination?type=LOCATION&action=<VIEW\|GOTO\|ADDNEXTSTOP>&lat=...&long=...&name=...&address=...&city=...&juris=...&state=...&zip=...` (also `customPlaceID` and `trimblePlaceID`; `copilotv9://` for CoPilot 9) | Shows a location on the map (`VIEW`), makes it the only destination after clearing the trip (`GOTO`), or adds it as the next stop (`ADDNEXTSTOP`) | Excluded: same |
| `http://www.copilotlive.com/copilot/android?type=LOCATION&...` | The legacy Android form of the same | Excluded: same |
| `https://copilotgps.com/copilotgps?<query>` (or `http://`) | The clickable form required on Android 12 and later | Excluded: same |
| `geo:<lat>,<long>?q=<address>`, `geo:0,0?q=<lat>,<long>@<address>`, `geo:0,0?q=<lat>,<long>(<label>)` | Geo URI stops on Android | Excluded: same |
| `copilot://options?type=FLEETPORTAL&action=Set&DriverID=<id>&DeviceID=<id>&PartnerID=<id>&EnableCompliance=<bool>&EnableFleetPortal=<bool>&showconfirmation=<bool>&ShowCompliancePopup=<bool>` | Connects to Account Manager Fleet Settings (`ShowCompliancePopup` is deprecated since 10.19.3.48) | Excluded: changes device configuration |

## 20. Trimble Mobile Manager launch interfaces (not a command line), excluded

Sources (verified 2026-09-29):

- Android intents: https://developer.trimble.com/docs/mobile-manager/reference/android-intents/
- iOS URL schemes: https://developer.trimble.com/docs/mobile-manager/reference/ios-url-schema/
- Windows URI scheme: https://developer.trimble.com/docs/mobile-manager/reference/windows-request-schema/

Trimble Mobile Manager (TMM) runs on a field device and connects its GNSS receiver. Other apps on the same device use these interfaces to open TMM pages, sign in, register, and find the ports of TMM's local REST and WebSocket APIs. That local REST API is catalogued in [endpoints/mobile-manager.md](endpoints/mobile-manager.md).

**Why it is excluded:** these interfaces work only on the field device, which the bridge cannot reach. Several return Trimble ID tokens to the calling app (the ID token and access token, and on Android `LOGIN` also the refresh token), and the bridge never handles credentials. Those that open the correction source, configuration, antenna height, receiver selection, tilt, IMU or pole-bias calibration pages lead to changes of GNSS receiver, correction, antenna or calibration settings. They are **safety-excluded**, as the matching Mobile Manager operations are in the API catalogue: positioning that guides survey or machine work must be configured by a qualified person on the device.

Android intents (`Intent` actions):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `com.trimble.tmm.CORRECTIONSOURCESETTINGS` | Opens the correction source settings page | Excluded (safety): correction settings |
| `com.trimble.tmm.EBUBBLECALIBRATION` | Opens the eBubble tilt calibration page | Excluded (safety): calibration |
| `com.trimble.tmm.GNSS_STATUS` | Opens the GNSS Status page | Excluded: device-local user interface |
| `com.trimble.tmm.HERE` | Opens Point Measurement (inputs `distanceUnit`, `name`, `comment`, `antennaHeight`) and returns the measured point: latitude, longitude, height, elevation, antenna height, precisions, sample count, optional EPSG code and account | Excluded (safety): sets the antenna height of a survey measurement and returns a precise position |
| `com.trimble.tmm.IMUBIASCALIBRATION` | Opens the IMU bias calibration page | Excluded (safety): calibration |
| `com.trimble.tmm.LOGIN` | Downloads licences for the user and app (input `applicationID`) and returns the account name, ID token, access token, refresh token and Trimble Precision SDK version | Excluded: returns credentials |
| `com.trimble.tmm.ONDEMAND` | Opens the On-Demand licensing page; returns the current claim and the time remaining | Excluded: licensing |
| `com.trimble.tmm.OPENANTENNAHEIGHT` | Opens the Antenna Height page | Excluded (safety): antenna settings |
| `com.trimble.tmm.OPENLASEROFFSET` | Opens the Laser Offset data-collection workflow; returns the geometry and ESRI-compatible feature attributes | Excluded: field data collection on the device |
| `com.trimble.tmm.OPEN_TO_CONFIGURATION` | Opens the Configuration page | Excluded (safety): receiver and correction configuration |
| `com.trimble.tmm.OPEN_TO_LOGIN` | Starts sign-in (input `applicationID`); returns the account name, TID, ID token, access token and Trimble Precision SDK version | Excluded: returns credentials |
| `com.trimble.tmm.POLEBIASADJUSTMENT` | Opens the Pole Bias Adjustment page | Excluded (safety): calibration |
| `com.trimble.tmm.PseudoSecureSocketServerPort` | Starts the pseudo-secure socket server activity (input `applicationID`); returns its port and the REST API and WebSocket ports | Excluded: device-local |
| `com.trimble.tmm.RECEIVERSELECTION` | Starts receiver selection | Excluded (safety): changes the GNSS receiver |
| `com.trimble.tmm.RefreshUserToken` | Refreshes the user token (input `applicationID`); returns `info` or `error` | Excluded: credential handling |
| `com.trimble.tmm.REGISTER` | Registers the calling app for the REST API (input `applicationID`); returns the result and the REST API and WebSocket ports | Excluded: device-local registration |
| `com.trimble.tmm.SKYPLOT` | Opens the Skyplot page | Excluded: device-local user interface |
| `com.trimble.tmm.SocketServerPort` | Starts the socket server port activity (input `applicationID`); returns its port and the REST API and WebSocket ports | Excluded: device-local |
| `com.trimble.tmm.STATUS` | Returns the account status (input `applicationID`): name, TID, ID token, access token, refresh recommendation, subscription name, expiry and precision thresholds, distance unit, whether the Catalyst service must be installed, and the device's hardware support level | Excluded: returns credentials |

iOS URL schemes (parameters are a Base64-encoded JSON object; responses go to the caller's `returl`):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `tmm://` | Pings TMM to check that it is available | Excluded: device-local |
| `tmmconnect://?<Base64 JSON>` (`application_id`, `returl`) | Signs in to Trimble ID through TMM; returns `UserTID`, `DeviceID` and `TPSDKVersion` | Excluded: sign-in |
| `tmmlogin` (its documented request URL is `tmmconnect://?<Base64 JSON>`) | The same client login | Excluded: sign-in |
| `tmmcorrectionsettings://` | Opens the configuration page to edit the correction settings | Excluded (safety): correction settings |
| `tmmfilelocations://?<Base64 JSON>` (`returl`) | Returns the base location, correction-settings path and licence-report path | Excluded: device-local files |
| `tmmondemand://?<Base64 JSON>` (`application_id`, `returl`) | Returns the current On-Demand claim and countdown | Excluded: licensing |
| `TmmOpenLaserOffset://?<Base64 JSON>` (`returl`) | Opens the Laser Offset workflow; returns URL-encoded geometry and feature attributes | Excluded: field data collection on the device |
| `TmmOpenToAntennaHeight://` | Opens the Antenna Height page | Excluded (safety): antenna settings |
| `tmmopentoconfiguration://` | Opens the Configuration page | Excluded (safety): receiver and correction configuration |
| `tmmopentologinpage://?<Base64 JSON>` (`application_id`, `returl`) | Opens the login page | Excluded: sign-in |
| `tmmopentoreceiverselection://` | Opens receiver selection | Excluded (safety): changes the GNSS receiver |
| `TmmOpenToSkyplot://` | Opens the Skyplot page | Excluded: device-local user interface |
| `tmmpseudosecuresocketserver://?<Base64 JSON>` (`returl`) | Starts the pseudo-secure socket server; returns the REST API and WebSocket ports | Excluded: device-local |
| `tmmrefreshusertoken://?<Base64 JSON>` (`application_id`, `returl`) | Refreshes the user token; returns a result code from 0 to 4 and a message | Excluded: credential handling |
| `tmmregister://?<Base64 JSON>` (`application_id`, `returl`) | Registers the app; returns the result and the REST API and WebSocket ports | Excluded: device-local registration |
| `tmmsocketserverport://?<Base64 JSON>` (`returl`) | Returns the REST API and WebSocket ports | Excluded: device-local |

Windows URI scheme (`trimblemobilemanager://request/<request>?callback=<uri>`; TMM answers on the callback URI with `id`, `status` and `message`):

| Command/Switch | What it does | Disposition |
|---|---|---|
| `tmmCorrectionSettings` | Opens the configuration page to edit the correction settings | Excluded (safety): correction settings |
| `tmmOnDemand` | Opens the On Demand page; returns the current claim and countdown (its description on the page repeats the receiver-selection text) | Excluded: licensing |
| `tmmOpenLaserOffset` | Opens the Laser Offset workflow; returns the geometry and feature attributes | Excluded: field data collection on the device |
| `tmmOpenToAntennaHeight` | Opens the Antenna Height page | Excluded (safety): antenna settings |
| `tmmOpenToConfiguration` | Opens the Configuration page | Excluded (safety): receiver and correction configuration |
| `tmmOpenToLoginPage` (`applicationId`) | Opens the login page; returns the user's TID and TPSDK version | Excluded: sign-in |
| `tmmOpenToReceiverSelection` | Opens receiver selection | Excluded (safety): changes the GNSS receiver |
| `tmmOpenToSkyplot` | Opens the Skyplot page | Excluded: device-local user interface |
| `tmmRegister` (`applicationId`) | Registers the app; returns the result and the REST API and WebSocket ports | Excluded: device-local registration |
| `tmmSocketServerPort` | Returns the REST API and WebSocket ports | Excluded: device-local |

## 21. Trimble-published npm executables, excluded

Source: the npm organisation `trimble-oss` (https://www.npmjs.com/org/trimble-oss; package list https://registry.npmjs.org/-/org/trimble-oss/package), and each package's registry record and readme (verified 2026-09-29). Of the organisation's 18 packages, these four declare executables; the other 14 declare none.

**Disposition:** design-system developer tooling, not a Trimble product interface. These tools run in a developer's project, install packages and write files, or start local MCP servers for coding assistants. None of them reaches a Trimble product's data.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `modus-wc init` (package `@trimble-oss/modus-wc-cli` 0.0.3, executable `modus-wc`) | Sets up Modus Web Components in a project: installs the framework wrapper and Tailwind CSS, edits `index.html`, `vite.config.ts` and styles, and can configure an MCP server and Cursor rules | Excluded: design-system developer tooling; installs packages and writes files |
| `modus-wc add [ids...]`, with `--all` and `--pattern` | Adds UI templates or patterns (102 patterns) to the project | Excluded: same |
| `modus-wc help add [--pattern]` | Lists the available templates or patterns | Excluded: same |
| `modus-wc setup mcp` | Writes `.cursor/mcp.json`, `.cursor/rules/modus-wc.mdc` and `.cursor/skills/modus-wc/SKILL.md` | Excluded: same |
| `modus-wc info` | Shows the project's setup status | Excluded: same |
| `modus-wc docs [component]`, with `--open` | Shows component documentation; `--open` opens Storybook | Excluded: same |
| `modus-mcp-server` (package `@trimble-oss/modus-mcp-server` 1.0.2, author "Trimble") | A stdio MCP server with Modus component documentation and icon search (tools `getting_started_guidelines`, `get_list_of_all_modus_components`, `get_component_details`, `get_modus_icons_by_char`) | Excluded: design-system developer tooling |
| `moduswebcomponents-mcp` (package `@trimble-oss/moduswebcomponents-mcp` 1.19.0) | "MCP server for Modus Web Components documentation". The package has no readme, so no usage is documented | Excluded: same |
| `cem analyze` or `custom-elements-manifest analyze` (package `@trimble-oss/custom-elements-manifest-analyzer` 0.0.4, a Trimble-maintained fork of `@custom-elements-manifest/analyzer`) | Generates a custom elements manifest. Options: `--config`, `--globs`, `--exclude`, `--outdir`, `--dependencies`, `--packagejson`, `--watch`, `--dev`, `--quiet`, `--litelement`, `--fast`, `--stencil`, `--catalyst`, `--catalyst-major-2` | Excluded: same; writes files |

The `@trimble-oss/modus-mcp-server` readme installs and runs the unscoped name `modus-mcp-server` (`npm install -g modus-mcp-server`, `npx -y modus-mcp-server@latest`). npm shows that unscoped name as unpublished since 2025-07-14, so those commands do not fetch Trimble's scoped package.

## 22. Trimble Connect Object Manager (Quadri) `quadri.exe`, excluded

Sources (verified 2026-10-01):

- https://help.trimble.com/en/trimble-connect/trimble-connect/object-manager/functions/configuration-tools/scheduled-batch-operations ("Scheduled Batch Operations", modified 1 Oct 2026; it redirects to the same path under https://help.trimble.com/doc/). Its sub-page "Run a Batch File Automatically" covers only scheduling the batch file in Windows Task Scheduler.
- https://help.trimble.com/doc/quadri/quadri/quadri-connectors/quadri-fully-integrated-novapoint: Novapoint includes the Quadri desktop client, so the same `quadri.exe` command line applies to Novapoint installations. No separate Novapoint command line was checked.

Trimble Connect Object Manager, formerly Quadri, works on a shared object model on a server. Its desktop client `quadri.exe` takes switches, and Trimble's example batch file (`.cmd`, placed in the folder that holds `quadri.exe`) runs it with `START /wait` to receive, share and run tasks without a user. Example lines, as documented:

- `START /wait quadri.exe -bkgr -exit:save -p:"%Workset%" -serverop:receive -wait:true`
- `START /wait quadri.exe -bkgr -exit:save -p:"%Workset%" -serverop:share -sharedesc:"<description>" -wait:true`
- `START /wait quadri.exe -bkgr -exit:save -p:"%Workset%" -batchtask:<task GUID> -wait:true`

**Why it is excluded:** it is a local desktop program. `-serverop:share` and `-batchtask` change the shared cloud model that the whole project team works on, and `-license` changes the licensing regime. The bridge runs no local programs, and the Object Manager model is not reached through any catalogued API.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `-f` | Batch file name | Excluded: local file path |
| `-p` | Project: the local model (workset) file | Excluded: local file path |
| `-c` | Configuration file | Excluded: local file path |
| `-l` | Layout file | Excluded: local file path |
| `-e` | Extensions file | Excluded: local file path |
| `-log` | Log configuration file | Excluded: local file path |
| `-a` | Performs an action: `new`, `load` or `load_prev` | Excluded: local program |
| `-nsp` | No splash screen | Excluded with the program |
| `-bkgr` | Runs in silent mode | Excluded: unattended run of the actions below |
| `-splash` | Sets the splash screen file | Excluded with the program |
| `-noserver` | Runs without a server | Excluded with the program |
| `-noautocad` | Runs without AutoCAD | Excluded with the program |
| `-license` | Sets the licensing regime | Excluded: changes licensing |
| `-language` | Sets the language | Excluded with the program |
| `-config` | Sets the configuration to use | Excluded with the program |
| `-userconfig` | Sets the user configuration to use | Excluded with the program |
| `-resetuserconfig` | Listed without a description | Excluded; its effect is undocumented |
| `-stopshareonvalidationerror` | Listed without a description | Excluded; its effect is undocumented |
| `-batchtask:<GUID>` | Runs a task in silent mode, for example producing TrimBIM files | Excluded: runs tasks; writes files |
| `-serverop:receive` or `-serverop:share` | Receives changes from, or shares changes to, the server; used with silent mode | Excluded: `share` changes the shared cloud model |
| `-sharetask` | Shares a given task | Excluded: changes the shared cloud model |
| `-sharedesc:"<text>"` | Description shown on the timeline when sharing | Excluded with `-serverop:share` |
| `-exit:save` or `-exit:discard` | Saves or discards the local model on exit | Excluded: writes local files |
| `-unit` | Sets the units, for example imperial | Excluded with the program |
| `-wait:true` | In every example line, but not in the parameter table | Excluded; its effect is undocumented |

## 23. Trimble Inpho command-line tools, excluded

Source: "Release Notes for Inpho 15" (Trimble Inpho 15.0.0 to 15.1.1, July 2025; contact imaging_support@trimble.com). The copy found is hosted by a reseller, not by Trimble: https://www.terraspatium.gr/uploads/2/4/2/3/24237397/releasenotes_trimblephotogrammetry__english__15.pdf (verified 2026-10-01). As with Convert to RINEX in section 15, a third-party copy of a Trimble document is used, so current versions may differ. The release notes name these command-line tools and options but give no syntax, and no public Trimble page documents the full command lines of the Inpho modules.

**Why it is excluded:** these are local photogrammetry programs that read and write image, terrain and point files on the workstation. The bridge runs no local programs and has no filesystem tool. Inpho's UASMaster installers are in section 14.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `make_pyr.exe` | Described as "the command line tool make_pyr.exe". Versions up to 14.1.0, with Image Commander (`imgcom`), can convert pseudo 16-bit JPEG TIFF images, which version 15 no longer reads | Excluded: local image processing; writes files |
| MATCH-T DSM `-TpixDtmAlignX`, `-TpixDtmAlignY` | Command-line options (15.1.0) that align the DTM TPIX tile grid to a coordinate definition, as in MATCH-3DX | Excluded: local processing |
| MATCH-T DSM `-TpixPtsAlignX`, `-TpixPtsAlignY` | The same for the point TPIX tiles | Excluded: same |
| MATCH-T DSM `-TpixXYNameSequence` | Swaps the XY index or coordinate order in TPIX file names (YX to XY), to match MATCH-3DX names | Excluded: same |
| DTMToolkit batch mode with an options file | Batch processing driven by an options file, which can define a name pattern (fixed in 15.0.0). The options-file format is not public | Excluded: reads and writes local files |

## 24. Applanix POSPac MMS `POSPacBatch.exe`, excluded

Source: "POSPac MMS 8.1 Release Notes" (Applanix, a Trimble company, June 2017), hosted by a reseller, not by Trimble or Applanix: https://assets.toyo.co.jp/files/user/POSPac_MMS_8.1_-_Release_Notes.pdf (verified 2026-10-01). Under POSPac MMS 7.2 it says "Added POSPacBatch.exe as part of the installer to run a command line version of POSPac". Under 8.1 it lists batch features marked "command line only": IMU data-gap scanning, multiple export and EO outputs, tags to turn DMI or GAMS on or off, and Single Base fallback when SmartBase fails. Batch projects are XML batch files (`.posbat`). No public page documents the switch list of `POSPacBatch.exe`. The auditor's `-qc` option does not appear in this document and was not found elsewhere, so it is not listed.

**Why it is excluded:** it is local GNSS-inertial post-processing that reads raw survey data and writes position and orientation solutions. The results feed survey and mapping work that a qualified person must check. The bridge runs no local programs and has no filesystem tool. TBC's installer for POSPac Command-Line TBC Subscription is in section 14.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `POSPacBatch.exe` | The command-line version of POSPac, installed with POSPac MMS since 7.2; runs batch projects | Excluded: local GNSS-inertial processing |
| `-m UAV` | Named once, in an 8.1 bug fix: "using POSPac UAV in command line fails when using the command line option –m UAV". Its full meaning is not documented | Excluded with the program |
| `.posbat` batch file | XML batch project file, with fields such as multipath settings, lever arms, mounting angles and GAMS lever arms | Excluded: reads a local file |

## 25. TruckMate program auto-login parameters, excluded

Sources: the TruckMate Online Help on Trimble's Transportation Learning Center: "Logging into TruckMate", https://learn.transportation.trimble.com/wp-content/uploads/tte/ebcbe19c93c746dd320c/olhlp/a9739f17da68/docs/Current/TMStdFunct/SF-02-LoggingIn.html, and "Communications Manager procedures", https://learn.transportation.trimble.com/wp-content/uploads/tte/ebcbe19c93c746dd320c/olhlp/66a8653bcc5d/docs/Current/MobileCommMgr/MCM-02-Procs.html. The help-path hash varies between search results (another is `.../olhlp/66280698c978/...`).

**Not read directly.** On 2026-10-01, learn.transportation.trimble.com redirected these pages to its home page, as in section 17. The entries below were confirmed from search-engine extracts of these official pages.

Every TruckMate program accepts these start-up parameters for automatic sign-in, for example `C:\Program Files (x86)\[TruckMate install folder]\MILESERV.EXE \U:USERNAME \P:Password \C:Schema \I:CompanyID`, or `COMMGR.EXE '\U:USER' '\P:password'` for Communications Manager. Either slash (`\` or `/`) is accepted.

**Why it is excluded:** `\P:` puts the user's TruckMate password on the command line, in shortcuts and scripts, where other local processes and users can read it. The parameters also launch desktop programs. The bridge handles no credentials and runs no local programs. The TruckMate REST API is catalogued separately.

| Command/Switch | What it does | Disposition |
|---|---|---|
| `\U:<user>` (or `/U:`) | TruckMate user name | Excluded: credential on the command line |
| `\P:<password>` (or `/P:`) | The user's password, in clear text | Excluded: password on the command line |
| `\C:<schema>` (or `/C:`) | Database schema; must be used with `\I:` | Excluded: launches a desktop program |
| `\I:<company ID>` (or `/I:`) | Company ID (an integer) for multi-company sign-in; must be used with `\C:` | Excluded: same |

## 26. Checked, with no official command line

- **Trimble Connect Sync:** GUI only (section 3).
- **Only installer command lines:** Tekla Structural Designer, Tedds for Word, Portal Frame Designer and Connection Designer (section 12). Otherwise they are automated through SDKs or APIs. Tekla PowerFab has installer command lines and a documented command-prompt database repair (`mysqlcheck`), both in section 12.
- **Also automated through SDKs or APIs:** besides the command lines above, SketchUp, Tekla Structures, Tedds, CoPilot and PC\*MILER have SDKs or APIs. Trimble Access has no documented command line and is automated through its SDK. See [capability-matrix.md](capability-matrix.md).
- **POSPac:** POSPac MMS's command-line program `POSPacBatch.exe` is publicly named in Applanix release notes (section 24), but no public page documents its switches. POSPac Command-line TBC Subscription is installed by TBC (section 14); no public page documents its own command line.
- **Spectrum Command Prompt** (https://help.trimble.com/doc/spectrum/spectrum/system-administration/administrator-utilities/command-prompt, verified 2026-10-01): opened inside Spectrum with Ctrl + Break, it takes a Spectrum function name to go to that screen, or `PA` to return to the Site Map. Like TBC's CAD command line (section 14), it is an in-application navigation prompt, not an operating-system command line. It belongs to the desktop user interface, which the bridge does not drive, so it is outside this page.
- **Inpho modules:** apart from the tools and options in section 23, no public page documents a command line for the Inpho modules.
- **TSEP Manifest Generator** (https://github.com/TrimbleSolutionsCorporation/TSEPManifestGenerator): a GUI tool; its readme documents no command line.
- **Not product command lines:** the Trimble-published npm executables (section 21), the open-source tools in the `trimble-oss` GitHub organisation (for example DBA Dash), and `CxxSonarQubeRunner` (section 13). These are not Trimble product interfaces.
- **Needs re-reading:** the PC\*MILER BatchPro, Rail-BatchPro and installer entries and the `LOG_BASIC` form of `pcmsock` (section 17), from a network that can open support.pcmiler.com and support.developer.trimblemaps.com; and the TruckMate auto-login parameters (section 25), from a network that can open learn.transportation.trimble.com.

Command-line access to Trimble APIs in general goes through `trimblectl` (section 5). `trimblectl api operations` searches every catalogued operation of every product. `trimblectl api read` calls Trimble Connect reads, and `trimblectl api plan` prepares dry-run requests for Trimble Connect changes and other products' reference operations.
