# Trimble Connect adapter: endpoint trace

Adapter: `internal/trimble/connect`. Every wire field the adapter reads is listed below. Anything not listed is ignored.

| Bridge operation | Upstream call | Fields read | Source |
|---|---|---|---|
| `ListProjects` | `GET {base}/2.1/projects?pageSize=N[&skipToken=T]` | `items[].id`, `name`, `rootId`, `createdAt`, `updatedAt`; `links.next.href` | OpenAPI tcps 2.0 (2.1 path) |
| `ListFolderItems` | `GET {base}/2.1/folders/{folderId}/items?pageSize=N[&skipToken=T]` | `items[].id`, `name`, `type`, `versionId`, `parentId`, `modifiedOn`, `projectId`; `links.next.href` | OpenAPI tcps 2.0 (2.1 path) |
| `GetFileMetadata` | `GET {base}/2.0/files/{fileId}` | `id`, `name`, `type`, `versionId`, `parentId`, `createdOn`, `modifiedOn`, `size`, `projectId`, `revision`, `hash` | OpenAPI tcps 2.0 |

## Safety behaviours (tested in `connect_test.go`)

- **Base URL:** fixed to the documented regional hosts. An override is allowed only with an explicit test flag.
- **Redirects:** never followed, so the bearer token cannot be forwarded to another host.
- **Pagination:** `links.next.href` is parsed, never fetched. Its host must match the base host, and only `skipToken` is extracted. A repeated cursor is an error.
- **Project binding:** `projectId` must be present and equal the authorized project. Otherwise the call fails with `resource_not_found`, or `upstream_malformed_response` when the field is absent.
- **Response size:** capped at 8 MiB.
- **Retries:** up to 3 attempts, only for 429/500/502/503/504 on GET. `Retry-After` is honoured, capped at 30 s. 401 is never retried.
- **Error bodies:** kept only as private causes and never returned to agents.

## Not implemented, and why

| Endpoint | Reason |
|---|---|
| `GET /projects/{id}` | Not verifiable in the retrieved spec |
| `GET /users/me` | Response fields verified only for the Workspace (JS) API |
| Download URL, upload, versions | Deferred to the Level 1 and Level 3 phases, which need the approval framework |
