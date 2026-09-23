# ADR-0005: TypeSafe skill evaluated; no runtime dependency adopted

- **Status:** Accepted, 2026-09-23.
- **Context:** The project owner asked for the TypeSafe skill (typesafe-ai/skills) to be installed and used. Installing the plugin was blocked by the session's permission policy, so the skill was read from its published SKILL.md. TypeSafe provides hosted "System One" models that return typed judgments (Choice, Noul, Score) for semantic decisions.
- **Evaluation:** Every decision in this release is deterministic policy that must be enforced in code: authorization, tenant binding, ID validation, CRS requirements, pagination. Following the skill's own guidance ("keep known rules, calculations, exact lookups, and execution in code"), none of these should become a model judgment.
- **Candidate future uses:**
  - Scoring skill activation (Choice: activate / no_activate / clarify) against `evals/activation.yaml`.
  - Flagging likely prompt-injection text in upstream names (Noul).
  - Both would send Trimble customer text to a third-party service, which privacy rule TRM-PRIV requires to be authorized, minimized, and disclosed.
- **Decision:** Adopt no runtime dependency. Revisit for offline evaluation tooling, using only synthetic eval prompts, once the owner approves the data flow (open question Q-6).
