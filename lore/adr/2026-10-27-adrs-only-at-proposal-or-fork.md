---
id: 2026-10-27-adrs-only-at-proposal-or-fork
status: accepted
zones: [workflows, adrs]
tldr: "ADRs are created only at a proposal, from the owner's answer at a fork, or in the bootstrap interview — never on the agent's own initiative during implementation."
rules:
  - "DO: hit a fork → stop and ask the owner; offer an ADR draft only if the answer should outlive the change"
  - "DON'T: write an ADR to justify a choice you made yourself"
  - "DON'T: record obvious rules as ADRs — they go to the zone's Invariants"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# ADRs only at a proposal or a fork

## Context
An ADR is binding for every future agent. If any agent may create one at any time, ADRs multiply and include choices nobody actually made.

## Decision
An ADR is created only when the decision has a non-obvious "why" and a future agent might want to "fix" it — and only at a proposal, from the owner's answer at a fork, or in the bootstrap interview. During implementation the agent asks instead of deciding.

## Why
Otherwise there are too many approvals, and the owner stops reading them. And the agent retroactively presents its own decisions as agreements.
