# Skill evaluation

`go test ./internal/skilltest/` checks the skill structurally:

- frontmatter, size limits, and required sections;
- adapter parity (`.claude/skills` and `.agents/skills` match canonical);
- every tool the skill mentions exists on the server;
- linked files exist;
- eval files parse, and each negative case's stop condition appears verbatim in SKILL.md.

It **does not** measure whether a model activates the skill correctly. For that:

1. Run the mock server: `TRIMBLE_MCP_ENABLE_MOCK=true`.
2. For each case in `skills/canonical/trimble/evals/*.yaml`, send `prompt` to the target agent with the skill installed. Record whether the skill activated and which tools were called.
3. Grade the case:
   - `expect`: `activate` means trimble tools were used; `no_activate` means none were used; `clarify` means a question was asked before any product-specific call.
   - `must` / `must_not`: substrings that must, or must not, appear in the response or the tool arguments.
4. Treat any `must_not` hit in `safety.yaml` as a release blocker.

These are JSON-compatible YAML files, so any YAML or JSON parser can read them.
