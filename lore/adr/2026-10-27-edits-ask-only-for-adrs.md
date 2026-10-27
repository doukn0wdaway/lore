---
id: 2026-10-27-edits-ask-only-for-adrs
status: accepted
zones: [install, workflows]
tldr: "The agent reads lore docs freely and edits zones, product.md and architecture.md without a permission prompt — they are approved at the gate (the diff of lore/ before --finish); only edits of ADRs in adr/ ask the owner through Claude Code's permissions.ask, and nothing is denied."
rules:
  - "DO: read lore/ whenever useful, without asking"
  - "DO: keep the permissions.ask rule for Edit of adr/** only"
  - "DON'T: add ask rules for zones, product.md or architecture.md — the gate approves them"
  - "DON'T: turn the ask rule into a deny rule"
  - "DON'T: work around the ADR prompt by editing through another tool"
supersedes: [2026-10-27-read-freely-edit-asks]  # ids of ADRs this one replaces entirely
superseded_by: null
inferred: false             # true — inferred from code, the owner could not tell why
ref: "change edits-ask-only-for-adrs"
---
# Only ADR edits ask the owner

## Context
The law docs must not change behind the owner's back, but the agent writes them in several workflows (bootstrap, zone, adr, product, archive). Every edit of zones, ADRs, `product.md` and `architecture.md` used to ask the owner through `permissions.ask`, so the sync at archive became dozens of prompts.

## Decision
`lore init` and `lore upgrade` allow `Bash(lore:*)` and add a `permissions.ask` rule only for `Edit` of `adr/**` in the docs folder, and remove lore's own older `ask` rules for zones, `product.md` and `architecture.md`. Reading is not restricted; `changes/` is not guarded; nothing is denied.

## Why
Approving every edit of zones, `product.md` and `architecture.md` by hand — above all during the sync at archive — duplicates the gate: the owner already approves the whole diff of `lore/` before `--finish`, and `--finish` verifies the merge line by line. ADRs keep the prompt: they are the most binding documents, and direct edits to them are rare (drafts live in `changes/`, statuses change through the CLI). Ask rather than deny still holds from the superseded decision: a hard ban breeds workarounds and extra steps.

## Rejected alternatives
### Ask on every edit of the law docs
Why not: duplicates the gate and turns the sync into dozens of prompts.

### No ask rules at all
Why not: the owner keeps the prompt for ADRs, the most binding documents.

### Deny edits of the law docs
Why not: breeds workarounds and extra steps.
