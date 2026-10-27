---
id: 2026-10-27-workflows-linked-not-copied
status: accepted
zones: [install, workflows]
tldr: "Workflows exist once, as files in the repo (lore/workflows/), and reach Claude Code as skills through symlinks in .claude/skills/ — never as copies."
rules:
  - "DO: edit workflows in lore/workflows/ (in the lore repo itself: internal/assets/files/workflows/)"
  - "DON'T: copy workflow text into .claude/skills, CLAUDE.md, AGENTS.md or anywhere else"
  - "DON'T: replace the skill symlinks with copies"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# Workflows are linked, not copied

## Context
The agent's instructions have to be available as Claude Code skills, and they are also referred to from the protocol in `AGENTS.md`.

## Decision
Every workflow is one `SKILL.md` in `lore/workflows/<name>/`. `lore agents sync` links it into `.claude/skills/lore-<name>` with a relative symlink. The `AGENTS.md` block only names the workflows.

## Why
In the previous kit (nexus), copies of the same rules diverged in 11 places.
