# Product

## What it is
A CLI plus a set of agent workflows that keep a project's knowledge in the repo: what the system does (zones), why it is built this way (ADRs) and what is being changed now (changes). Relevant knowledge is delivered to the AI agent automatically; the docs change only through a gate the owner approves.

## Who it is for
A developer who builds projects with AI coding agents (Claude Code) and approves what the agent writes. The agent is the second reader of everything lore produces.

## Scenarios

- [live] Owner sets up lore in a project → docs skeleton, agent protocol, permissions, prompt hook and skills are in place
  zones: cli, install
- [live] Owner upgrades lore in a project → workflows and prompts are updated (local edits kept unless forced), new config keys added, old layouts migrated
  zones: cli, install
- [live] Owner bootstraps an existing project → agent derives product, zone map, architecture and zones from the code, then interviews the owner and records ADRs
  zones: workflows, docs, adrs, cli
- [live] Owner sends a prompt → agent gets the relevant requirements and decisions in its context, each only once per session
  zones: decisions-model, docs, adrs
- [live] Owner explores an idea → agent thinks it through against existing zones and decisions, no code
  zones: workflows, decisions-model
- [live] Owner proposes a feature spec-first → proposal, deltas, tasks and ADR drafts are approved before any code
  zones: workflows, changes, adrs
- [live] Agent implements a change task by task → before each task it sees the next task, the deltas and the relevant decisions
  zones: workflows, changes, decisions-model
- [live] Owner asks to document code already written → a change is created from the git diff and its facts are written as deltas
  zones: workflows, changes
- [live] Owner archives a change at the gate → deltas reconciled with the code, facts checked against accepted ADRs, deltas merged into zones, owner approves the diff, the merge is verified and the change archived
  zones: workflows, changes, decisions-model, adrs, docs, cli
- [live] Owner records or replaces a decision → an ADR is written, accepted on the owner's yes, replaced ADRs are marked superseded
  zones: workflows, adrs
- [live] Owner or agent asks why something is built this way → the answer cites ADRs and zones, or says plainly that no reason is recorded
  zones: workflows, adrs, docs
- [live] Owner or agent looks up a zone → the zone plus the ADRs that apply to it (own and global)
  zones: docs, adrs
- [live] Owner adds, updates, splits or merges a zone, or adds a wanted scenario and later marks it live
  zones: workflows, docs
- [live] Owner checks the docs → lint reports format errors, broken references, a stale index and dead code anchors; an audit compares the docs with what the code does
  zones: cli, workflows, docs
