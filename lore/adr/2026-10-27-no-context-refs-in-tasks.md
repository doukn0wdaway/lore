---
id: 2026-10-27-no-context-refs-in-tasks
status: accepted
zones: [changes, workflows]
tldr: "tasks.md stays plain OpenSpec checkboxes — no ctx:, verify: or depends: fields; lore change status gathers the context for the next task, and the numbering gives the order."
rules:
  - "DO: run lore change status before each task to get its context"
  - "DON'T: add ctx:/verify:/depends: annotations or links to docs in tasks.md"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# No context references in tasks

## Context
A task needs the relevant requirements and decisions. They could be listed in the task itself, or gathered when the task starts.

## Decision
Tasks are plain numbered checkboxes. `lore change status` prints the next task with the change's deltas, the decisions of the touched zones and the knowledge the decisions model picks for that task.

## Why
Hand-written references go stale and duplicate what the model picks. And the prompt hook doesn't fire on "continue", so the context has to come from `status` anyway.
