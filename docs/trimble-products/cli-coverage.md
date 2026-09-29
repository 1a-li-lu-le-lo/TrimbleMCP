# Command-line coverage

Every command-line interface Trimble documents for Trimble Connect, and every `trimblectl` command, is accounted for here. `internal/skilltest` checks that this page covers every view, every panel and every `trimblectl` command.

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

## 6. Other Trimble products

See [capability-matrix.md](capability-matrix.md) for command-line tools of other Trimble products and their dispositions.
