# Open questions

These need answers from the project owner or from Trimble before the pilot.

1. **Q-1: Target products and use cases.** The brief left products, use cases, and tenant as placeholders. This build chose Trimble Connect read-only as the first slice. Confirm, or name the intended products (Maps, ProjectSight, Vista, fleet, ...).
2. **Q-2: Full Connect OpenAPI.** The retrieved SwaggerHub definition was truncated. Obtain the complete definition to verify `GET /projects/{id}`, `GET /users/me`, and the v2.0 collection fields.
3. **Q-3: Credentials and callback.** Is there a Trimble Connect sandbox/partner account? The loopback redirect URI (`http://127.0.0.1:8765/callback`) must be registered via connect-support@trimble.com.
4. **Q-4: Staging identity.** `stage.id.trimble.com` could not be resolved during verification. Which issuer pairs with the staging Connect hosts?
5. **Q-5: Remote authorization server.** Which IdP should issue tokens for remote MCP clients (Claude app, Perplexity)? This is needed for the OAuth 2.1 resource-server work.
6. **Q-6: Data classification.** What classification applies to project names, file names, and documents, and may they be sent to third-party model providers?
7. **Q-7: Rate limits.** Trimble does not publish Connect rate limits. Ask Trimble for tenant limits so the client-side 5 req/s default can be tuned.
