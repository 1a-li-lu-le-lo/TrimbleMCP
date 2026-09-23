# Privacy

| Data | Classification | Purpose | Sent to model? | Retention in the bridge |
|---|---|---|---|---|
| Project names and IDs | Confidential | Resolve scope | Yes (minimised, cleaned, labelled untrusted) | Not stored; the audit log keeps hashes only |
| File names and metadata | Confidential | Locate documents | Yes (as above) | Not stored |
| File content | Confidential or restricted | Not accessible in this release | No | n/a |
| Trimble Identity tokens | Secret | Upstream auth | Never | Encrypted token store |
| Caller bearer tokens | Secret | Remote auth | Never | SHA-256 only |
| Audit records | Internal | Accountability | No | Append-only JSONL; retention set by the operator |
| Location, fleet, farm, worker data | Restricted | Not configured | No | n/a |

Principles:

- Least data: one page at a time, read-only, no content download.
- The project grant filter never reveals names of projects outside the grant.
- Model training: whether the chosen client providers train on this data must be confirmed by the operator (open question Q-6). The bridge sends no data to any model provider itself.
