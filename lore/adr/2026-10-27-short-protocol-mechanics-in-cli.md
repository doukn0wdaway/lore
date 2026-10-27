---
id: 2026-10-27-short-protocol-mechanics-in-cli
status: accepted
zones: [workflows, install]
tldr: "The AGENTS.md protocol stays about 15 lines; the process lives in one agent workflow per everyday operation (adr, why, zone, product, audit, …), and the mechanics — statuses, superseding, merge verification, archiving — are done by CLI commands."
rules:
  - "DO: add a workflow (skill) for a new everyday operation instead of growing the AGENTS.md block"
  - "DO: put mechanical steps into a lore CLI command"
  - "DON'T: let the agent change statuses or move ADRs by editing files"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# Short protocol, workflows for operations, mechanics in the CLI

## Context
The agent needs to know how to work with lore. The rules could live in the always-loaded protocol, in task-specific workflows, or be enforced by commands.

## Decision
The `AGENTS.md` block is a short protocol (about 15 lines) that names the workflows. Each everyday operation has its own workflow. Everything mechanical is a CLI command: `lore adr accept` changes statuses and supersedes, `lore change archive` checks and archives.

## Why
Without a command for an operation, the agent does it "however it turns out"; editing statuses by hand breaks the supersede chains. Why the protocol itself is kept to about 15 lines is unknown: the owner doesn't remember, and thinks a long one is probably not needed (2026-10-27).
