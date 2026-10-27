---
id: 2026-10-27-model-sees-only-text
status: accepted
zones: [decisions-model, changes]
tldr: "The decisions model only ever sees text — zone tldrs, requirement and section bodies, ADR tldrs and rules, prompts, facts; code is never sent to it, the agent turns code into plain behavioural facts first."
rules:
  - "DO: in code-first changes, write what the diff does as plain facts under What changes, then run lore change check"
  - "DON'T: send diffs or source code to the decisions model"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# The decisions model sees only text

## Context
At the gate, code changes have to be checked against accepted decisions, and the model could be given the diff directly.

## Decision
The model gets only text. In a change, the agent translates the code into behavioural facts (bullets under `## What changes`), and `lore change check` asks the model about those facts.

## Why
That is how decisions models work: they answer questions about a textual state.
