---
id: 2026-10-27-one-gate-for-both-modes
status: accepted
zones: [changes, workflows]
tldr: "Spec-first and code-first changes end at the same archive gate with the same checks; the only difference is that a code-first change has no tasks."
rules:
  - "DO: send code-first work through /lore-document and then /lore-archive"
  - "DON'T: add a separate, lighter path for updating zones after coding"
  - "DON'T: skip the fact check or the owner's diff review for code-first changes"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# One gate for spec-first and code-first

## Context
Docs can be updated after planning (spec-first: proposal → code) or after coding in real time (code-first: the change is created from the git diff).

## Decision
Both modes end at the same gate: reconcile deltas with the code, check facts against decisions, settle ADR drafts, merge, the owner's yes on the diff, `--finish`.

## Why
Two different ways of updating the docs would drift apart in their checking rules over time.
