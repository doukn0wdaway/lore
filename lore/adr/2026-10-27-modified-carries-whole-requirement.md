---
id: 2026-10-27-modified-carries-whole-requirement
status: accepted
zones: [changes]
tldr: "A MODIFIED requirement in a delta carries the whole requirement — description and every scenario that still holds — and --finish requires the zone's block to equal the delta's block exactly."
rules:
  - "DO: copy unchanged scenarios of the requirement into a MODIFIED block"
  - "DON'T: write a partial MODIFIED that lists only the changed scenario"
  - "DON'T: relax --finish back to 'the delta's lines are present in the zone'"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15 item 38"
---
# MODIFIED carries the whole requirement

## Context
lore first allowed a partial MODIFIED: a delta listed only the scenarios that changed, the rest of the requirement was kept on merge, and `--finish` checked only that every line of the delta was present in the zone. With that check, lines of the old text left in the zone after the merge (an old THEN next to the new one) went unnoticed.

## Decision
A MODIFIED block carries the whole requirement, as in OpenSpec. On merge it replaces the zone's block; `--finish` requires the merged block (and every ADDED requirement and MODIFIED section) to equal the delta's block line for line, ignoring comments, blank lines and whitespace. The pre-merge check warns about each scenario a MODIFIED block drops.

## Why
With a partial MODIFIED the check could not notice old lines left in the zone after the merge, so a zone could be corrupted silently. OpenSpec works this way today. (Owner, 2026-10-27: "вариант 2, как в openspec".)

## Rejected alternatives
### Partial MODIFIED, exact match per scenario
Why not: still misses a scenario the agent drops by accident while merging.

### Compare the zone before and after the merge through git
Why not: heavy, and `--finish` would need git.

## Cost
Unchanged scenarios are copied into the delta.
