# Authentication

You never handle credentials. An operator configures access outside the conversation.

## How access works

- **Agent → bridge:** stdio runs as the local operator with read-only scopes. Remote HTTP uses a bearer token held in the client's own secret store, never in chat.
- **Bridge → Trimble Connect:** Trimble Identity OAuth 2.0 authorization code with PKCE. Trimble Connect does not accept client-credentials tokens. The operator signs in with `trimblectl auth login`, and the refresh token is stored encrypted.
- **Scopes:** the bridge uses its own scopes:
  - `trimble:capabilities:read`, `trimble:projects:read` and `trimble:files:read` are the local stdio default.
  - `trimble:audit:read` is also available.
  - `trimble:desktop:launch` is for the local operator only.
  - `trimble:files:write` and `trimble:files:delete` are reserved; configuration rejects them in this release.

## Troubleshooting without secrets

| Symptom | Meaning | What to tell the user |
|---|---|---|
| `authentication_error` | The Trimble Identity session is missing or expired (Connect sessions must refresh at least every 9 days), or the caller's bridge authorization expired | An operator should run `trimblectl auth login`, or reissue the client's bridge token. |
| `authorization_error` | The caller lacks a scope or project grant, or the upstream user lacks access | Request the grant from an administrator. |
| `entitlement_error` | No licence for the product | Check the Trimble subscription. |
| A tool is missing from the list | Scope not granted, or no adapter supports it | Check `trimble_get_capabilities`. |

If a user pastes a token or password, do not repeat it. Tell them to revoke it and use the configured flow.
