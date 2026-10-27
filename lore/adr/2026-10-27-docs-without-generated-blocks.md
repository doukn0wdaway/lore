---
id: 2026-10-27-docs-without-generated-blocks
status: accepted
zones: [docs]
tldr: "Zone docs have no generated block listing their ADRs; lore show joins a zone with its decisions on demand, and INDEX.md lists zones only, never ADRs."
rules:
  - "DO: use lore show <zone> to see a zone with its ADRs"
  - "DON'T: generate ADR lists or other derived blocks into zone docs"
  - "DON'T: add ADRs to INDEX.md"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# No generated blocks in zone docs

## Context
An agent working in a zone needs the zone's decisions. They could be written into the zone doc as a generated block, or joined when the zone is read. `INDEX.md` is always in the agent's context and could list ADRs too.

## Decision
Zone docs contain only hand-written (agent-written, owner-approved) content. `lore show <zone>` prints the zone followed by its accepted ADRs and global ones, and proposed ADRs separately. `INDEX.md` lists zones with their tldr and nothing else.

## Why
Once the prompt hook delivers ADRs to the agent, a generated block in the zone duplicates that delivery and adds noise to every diff of `lore/`. ADRs stay out of `INDEX.md` because their number grows without bound; they reach the agent through `lore show` and the hook.

## Rejected alternatives
### A generated "decisions" block inside each zone doc
Why not: duplicates the hook's delivery and is noise in the diff.
