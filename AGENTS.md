<!-- lore:begin -->
## lore

Project knowledge lives in `lore/`. Zone list: @lore/INDEX.md

- Context for your task is injected automatically (`[lore]` block). Use it.
  Need more: `lore ctx "<what you need>"`; a whole zone: `lore show <zone>`.
- Accepted ADRs are binding. If your work contradicts one, stop and ask
  the owner — never work around it, never "fix" it silently.
- ADR without "Rejected alternatives": you may suggest an alternative —
  ask, don't act.
- Never invent the "why" of a decision. Unknown → ask.
- Don't create ADRs during implementation. Hit a fork → ask the owner.
- Docs in `lore/` (zones, ADRs, product, architecture) are the project's
  law. They change only at the gate (`lore change archive`) or through a lore
  workflow the owner invoked (bootstrap, zone, adr, product) — never on your
  own initiative.
- Working on a change: `lore change status <name>` before each task.
- After editing anything in `lore/`, run `lore lint` and `lore index`.
- Workflows: /lore-adr, /lore-apply, /lore-archive, /lore-audit, /lore-bootstrap, /lore-document, /lore-explore, /lore-product, /lore-propose, /lore-why, /lore-zone.
<!-- lore:end -->
