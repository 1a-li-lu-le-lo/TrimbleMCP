# MCP tools, resources, and prompts

All tools except `trimble_open_in_desktop` are read-only (`readOnlyHint: true`, `destructiveHint: false`) and return the envelope described under "Output envelope" below.

A tool is listed only when two things hold:

- the caller holds its scope, and
- some configured adapter supports it.

| Tool | Scope | Capability | Required input | Notes |
|---|---|---|---|---|
| `trimble_get_capabilities` | `trimble:capabilities:read` | — | none | Call this first |
| `trimble_list_projects` | `trimble:projects:read` | `list_projects` | `product` | `page_size` 1–100, `page_token` |
| `trimble_get_project` | `trimble:projects:read` | `get_project` | `product`, `project_id` | Hidden for Trimble Connect (unverified endpoint) |
| `trimble_list_folder_items` | `trimble:files:read` | `list_folder_items` | `product`, `project_id`, `folder_id` | Covers the brief's `list_folders` and `list_files` |
| `trimble_get_file_metadata` | `trimble:files:read` | `get_file_metadata` | `product`, `project_id`, `file_id` | No content access |
| `trimble_build_desktop_link` | `trimble:projects:read` | `build_desktop_link` | `product`, `project_id`; optional `view`, `panel` | Trimble Connect for Windows command-line link; no side effects |
| `trimble_open_in_desktop` | `trimble:desktop:launch` | `launch_desktop` | `product`, `project_id`, `reason`; optional `view`, `panel`, `dry_run` (default true) | Local operator on Windows only; opt-in; the only tool with `readOnlyHint: false` (still `destructiveHint: false`) |

The brief's other tools — upload, download, create folder or version, views, geospatial, fleet, jobs, and audit lookup — are **not exposed**. They need either verified upstream support or the approval framework (ADR-0004).

## Output envelope

```json
{
  "status": "ok | error",
  "product": "trimble-connect",
  "operation": "trimble_list_projects",
  "resource": "folder/<id>",
  "result": {},
  "pagination": {"next_page_token": "…", "complete": false, "page_size": 50, "returned": 50},
  "units": "size_bytes: bytes",
  "crs": "",
  "source": {"product": "…", "api_version": "…", "region": "…", "source": "GET /2.1/projects", "observed_at": "…"},
  "observed_at": "…",
  "request_id": "req_…",
  "audit_id": "aud_…",
  "data_labels": ["observed_data", "not_certified"],
  "untrusted_fields": ["projects[].name"],
  "warnings": [],
  "next_action": "…",
  "error": {"code": "…", "safe_message": "…", "retryable": false, "remediation": "…", "retry_after_seconds": 0}
}
```

## Resources

- `trimble://capabilities`
- `trimble://{product}/projects/{project_id}`: listed only when an adapter supports `get_project`.

Reads go through the tool handlers, so they get the same authorization and audit.

## Prompts

- `inspect-trimble-project`
- `find-trimble-file`
- `troubleshoot-trimble-auth`
- `audit-trimble-integration`

Prompt arguments are cleaned and JSON-quoted. Prompts grant nothing.

## Protocol

| Mode | Revisions | Behaviour |
|---|---|---|
| Legacy | 2025-11-25, 2025-06-18, 2025-03-26 | `initialize` handshake; HTTP sessions via `Mcp-Session-Id`, bound to subject and tenant |
| Modern | 2026-07-28 | `_meta` protocol version on every request; `server/discover`; `MCP-Protocol-Version`, `Mcp-Method`, and `Mcp-Name` headers checked |

Both modes:

- JSON responses only; no SSE stream. GET returns 405.
- Batches are rejected.
- Maximum message size is 1 MiB.
