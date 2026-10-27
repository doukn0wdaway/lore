---
id: 2026-10-27-read-freely-edit-asks
status: accepted
zones: [install, workflows]
tldr: "The agent may read lore docs freely; every edit of zones, ADRs, product.md and architecture.md asks the owner through Claude Code's permissions.ask — there is no hard ban on editing them."
rules:
  - "DO: read lore/ whenever useful, without asking"
  - "DO: expect a permission prompt when editing zones, adr/, product.md, architecture.md"
  - "DON'T: turn the ask rules into deny rules"
  - "DON'T: work around the prompt by editing through another tool"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# Read freely, edit with the owner's permission

## Context
The law docs must not change behind the owner's back, but the agent writes them in several workflows (bootstrap, zone, adr, product, archive).

## Decision
`lore init` allows `Bash(lore:*)` and adds `permissions.ask` rules for `Edit` of `zones/**`, `adr/**`, `product.md` and `architecture.md` in the docs folder. Reading is not restricted; `changes/` is not guarded.

## Why
A hard ban, as in the old kit, breeds workarounds and extra steps.

## Rejected alternatives
### Deny edits of the law docs
Why not: breeds workarounds and extra steps.
