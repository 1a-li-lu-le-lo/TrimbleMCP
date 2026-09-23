# Trimble Connect for Windows command line

Trimble documents one command-line interface for Trimble Connect for Windows: its registered URL scheme. It can be launched from a command line or from a browser.

```
"trimbleconnect:/projects/[project-id]?show=[view parameter],[panel parameter]"
```

- **Views:** `projects`, `data`, `3D`
- **Panels:** `clashes`, `models`, `objects`, `ToDos`, `views`
- **Case:** parameters are case-insensitive. The bridge emits Trimble's documented spelling.
- **Example:** `trimbleconnect:/projects/rQR1yhTGj9I?show=3D,ToDos` opens the project in the 3D view with the ToDos tab shown.

Source: https://help.trimble.com/doc/trimble-connect/trimble-connect/connect-for-windows/getting-started/using-the-command-line

## Tools

| Tool | Effect | Requirements |
|---|---|---|
| `trimble_build_desktop_link` | Returns the link; opens nothing | `trimble:projects:read`; product `trimble-connect-desktop` configured |
| `trimble_open_in_desktop` | Opens Trimble Connect for Windows on the operator's own machine. **Dry run unless `dry_run: false`** | `trimble:desktop:launch` scope; local stdio session; Windows host; operator set `TRIMBLE_CONNECT_DESKTOP_LAUNCH=true`; a `reason` |

## Rules

1. Resolve the project ID with `trimble_list_projects` (product `trimble-connect`) first. The bridge re-checks the ID against the API and refuses to open an unverified project.
2. The command line only navigates the desktop application. It cannot read, change, export, or sync data, and the bridge cannot see what the application shows. Ask the user to confirm the result.
3. Call `trimble_open_in_desktop` with `dry_run: false` only when the user asked to open the application in this conversation. Otherwise return the link from `trimble_build_desktop_link`.
4. A panel needs a view. The documented form is `show=[view],[panel]`, and the bridge rejects a panel without a view.
5. Remote clients (Claude app, Perplexity, remote HTTP) can build links but can never launch anything. Launching would open an application on the server, not on the user's machine.
6. Do not pass raw URIs, other URL schemes, file paths, or command lines. The tools accept only a project ID, a view, and a panel.
7. Installing, updating, or uninstalling Trimble Connect for Windows is out of scope.
