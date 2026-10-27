---
id: 2026-10-27-finish-requires-clean-lint
status: accepted
zones: [changes]
tldr: "lore change archive --finish archives only when the whole project is lint-clean, not just the change being archived."
rules:
  - "DO: fix every lint error in the project before --finish, even ones unrelated to the change"
  - "DON'T: scope the --finish lint run to the change's own files"
supersedes: []
superseded_by: null
inferred: false
ref: null
---
# --finish requires the whole project to be lint-clean

## Context
`--finish` verifies the merge of the change's deltas. Errors may also exist elsewhere in `lore/` — another change, a stale index, a dead code anchor.

## Decision
After the merge checks pass and `INDEX.md` is regenerated, `--finish` runs the full `lore lint` and archives nothing while any error remains anywhere in the project.

## Why
So that the format of the docs always holds: every archive leaves the whole knowledge base valid. (Owner, 2026-10-27.)

## Cost
An unrelated error elsewhere blocks the archive of a finished change.
