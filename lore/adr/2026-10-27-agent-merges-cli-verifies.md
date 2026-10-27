---
id: 2026-10-27-agent-merges-cli-verifies
status: accepted
zones: [changes, workflows]
tldr: "Deltas use the OpenSpec format and are merged into zones by the agent; the CLI never merges, it only checks before the merge and verifies the result after it, in two separate commands."
rules:
  - "DO: merge deltas into zones yourself (/lore-archive), then show the owner the diff, then run lore change archive --finish"
  - "DON'T: add a CLI command that rewrites zone docs from deltas"
  - "DON'T: weaken --finish to checking only that headings exist"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# The agent merges deltas, the CLI verifies

## Context
A change carries deltas to zones. Someone has to apply them to the zone docs at the archive gate, and something has to make sure they were applied faithfully.

## Decision
Deltas use the OpenSpec format (ADDED / MODIFIED / REMOVED / RENAMED Requirements, plus MODIFIED Sections). The agent merges them into the zones. The CLI checks in two phases: `lore change archive` before the merge (the deltas fit the current zones), and `lore change archive --finish` after the merge and the owner's yes on the diff (the zones now hold exactly what the deltas say).

## Why
The requirement heading already works as an id, the format is familiar, and agent-driven merging has proven reliable in OpenSpec. The merge has to be verified line by line: checking only that a heading exists is not enough for MODIFIED — the first version of lore archived a change that had not been merged at all.
