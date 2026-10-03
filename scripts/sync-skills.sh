#!/usr/bin/env sh
# Copies the canonical skill into the platform install locations.
# Claude Code: .claude/skills/<name>/ ; OpenAI Codex: .agents/skills/<name>/
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
src="$root/skills/canonical/trimble"
for dst in "$root/.claude/skills/trimble" "$root/.agents/skills/trimble"; do
  rm -rf "$dst"
  mkdir -p "$dst"
  cp -R "$src/SKILL.md" "$src/references" "$src/templates" "$dst/"
done
echo "synced canonical skill to .claude/skills/trimble and .agents/skills/trimble"
