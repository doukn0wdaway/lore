---
id: 2026-10-27-inject-content-per-requirement
status: superseded
zones: [decisions-model]
tldr: "Context is pushed to the agent on every prompt by the hook as content, not as 'go read X'; the unit of selection is a requirement (or section, or ADR), chosen in two stages — zones first, then their items — with a timeout and no optimisation yet."
rules:
  - "DO: keep injecting the content of selected requirements and ADRs"
  - "DON'T: replace the injected content with a list of files or zones to read"
  - "DON'T: inject whole zones per prompt"
  - "DON'T: optimise the selection (caching, parallel tricks, fewer stages) before the pilot shows it is needed"
supersedes: []
superseded_by: 2026-10-27-agent-pulls-knowledge-with-ctx
inferred: false
ref: "design.md §15"
---
# Push requirement-level content on every prompt

## Context
The agent needs the relevant knowledge for each prompt. It could be told where to look, given whole zones, or given the relevant pieces. The model is slow on many candidates at once.

## Decision
The prompt hook runs on every prompt and injects the content of what the model picked; the unit is a requirement (or a free section, or an ADR), not a zone. Selection runs in two stages — zones by tldr, then the items of the chosen zones plus global ADRs — under a timeout, with no further optimisation for now.

## Why
What costs is the number of agent steps, not the amount read: every extra tool call re-reads the whole accumulated context. So the knowledge has to arrive by itself, as content. A whole zone per prompt is too expensive. pplx-decider is slow on a large number of candidates, so the first stage narrows them down; beyond that — a working flow first, optimisation later.

## Rejected alternatives
### Inject "read X" pointers only
Why not: each pointer costs the agent extra steps.

### Inject whole zones
Why not: too expensive per prompt.
