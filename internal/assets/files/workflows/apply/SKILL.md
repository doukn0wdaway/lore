---
name: lore-apply
description: Implement an approved lore change task by task — get the context from `lore change status` (then `--next` before each task), implement, tick it off, stop and ask at forks. Use when the owner says to implement, apply, continue or work on a lore change.
---

# lore apply — implement a change

## Loop

1. At the start (and again whenever your context was lost): `lore change status [name]` — the next task, this change's deltas, the decisions (ADRs) of the zones it touches and, if the decisions model is configured, the decisions and zones it picked for this task. The name may be omitted when only one change is active.
2. Implement that task only. Follow the accepted ADRs printed by `status`; the deltas say what behaviour to build. A zone named but not printed: `lore show <zone>`.
3. Verify it (tests / build / run, as the project does).
4. Tick it in `lore/changes/<name>/tasks.md`: `- [ ]` → `- [x]`.
5. `lore change status [name] --next` — only the next task and the decisions and zones picked for it (the rest is already in your context). Back to 2.

When `status` says all work is done → `/lore-archive`.

## Forks and surprises

- **Two reasonable ways and the deltas/ADRs don't decide** → stop and ask the owner. Don't pick and justify later.
- Record the answer in `proposal.md` under `## Owner notes`, quoted with the date. The archive step uses these notes to tell intended deviations from mistakes.
- If the answer should outlive this change (a real decision), propose an ADR draft: `lore adr new <slug> --zones … --change <name>`, fill it with the owner's reasoning, `lore adr accept <id>` after their yes.
- **The work contradicts an accepted ADR** → stop and ask: new decision (→ draft with `--supersedes`) or keep it (→ change the approach).
- Scope grows or a task turns out wrong → edit `tasks.md` (add / reorder / remove tasks) and tell the owner. Deltas may lag behind the code during the work — they are reconciled at the archive step.

Never edit zone docs or ADRs in `lore/adr/` during apply — they change only at the archive gate.
