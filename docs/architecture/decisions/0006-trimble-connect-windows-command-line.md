# ADR-0006: Support the Trimble Connect for Windows command line

- **Status:** Accepted, 2026-09-23.
- **Context:** The owner asked that the MCP support the Trimble CLI documented at https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line as well as the API. That page documents one interface: the registered URL scheme `"trimbleconnect:/projects/[project-id]?show=[view parameter],[panel parameter]"`. It can be started from a command line or a browser, and opens Trimble Connect for Windows at a project, a view (projects, data, 3D) and a panel (clashes, models, objects, ToDos, views). Parameters are case-insensitive.
- **Decision:**
  - New adapter `internal/trimble/desktop`, product `trimble-connect-desktop`, with two capabilities:
    - `build_desktop_link` (no side effects);
    - `launch_desktop` (opens the app locally).
  - Tools:
    - `trimble_build_desktop_link`: scope `trimble:projects:read`.
    - `trimble_open_in_desktop`: new scope `trimble:desktop:launch`. It is listed only for the local operator principal (stdio or CLI), only when `TRIMBLE_CONNECT_DESKTOP_LAUNCH=true`, and only on a Windows host. It is a dry run by default, needs a `reason`, is rate-limited (6 per minute per caller, burst 2), and is audited.
  - The URI is built only from a fixed prefix, a project ID matching `^[A-Za-z0-9_-]{1,64}$`, and enumerated view and panel values. Output is emitted in the documented spelling. A panel requires a view, because a panel-only form is undocumented.
  - The project ID is verified through `trimble-connect` `ListProjects` (up to 20 pages of 100) before a link is returned as verified. Launching refuses unverified projects, and a project that is absent from a complete listing is rejected as `project_not_found`.
  - On Windows the link is opened with `ShellExecuteW` (shell32, "open"). No command interpreter is used, so shell metacharacters are inert. Off Windows, launching is unavailable and the capability is not declared.
  - The HTTP transport rejects `trimble:desktop:launch` in the tokens file. Remote principals are never `Local`, so a remote caller can never open an application on the server host.
- **Consequences:**
  - The command line only navigates the application. It exposes no data, and the bridge cannot observe the result.
  - Assumption A-8: the scheme's project ID equals the REST API project ID. Until that is confirmed, the adapter is Provisional.
  - Installer and uninstaller command lines are out of scope, because they modify the host.
