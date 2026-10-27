---
name: lore-product
description: Maintain lore/product.md — add a wanted scenario as [planned] (a lightweight backlog), mark a scenario [live] when it works, or update the product description. Use when the owner says "I want the app to ...", "add to the backlog/roadmap", "what is planned", or after a feature ships.
---

# lore product — scenarios and backlog

`lore/product.md` answers "what should the application do and for whom". Each scenario is a **route through zones, without rules** — rules live in zones.

```markdown
- [planned] User sends a photo of a receipt → the expense lands in finance
  zones: transport, storage, finance
```

## Add a wanted scenario

1. Phrase it from the user's point of view: trigger → visible outcome. One line.
2. Pick the zones it goes through from `lore/INDEX.md`. A zone that doesn't exist yet → note it in the line (`zones: storage, finance, receipts?`) and tell the owner a new zone will be needed — don't create zones here (`/lore-zone`).
3. Status `[planned]`. Order of planned items = priority; ask the owner where it goes.

## Mark as live

Only when the behaviour really works in the code. `[planned]` → `[live]`; check the zone list still matches what was built.

## A scenario changed or is no longer relevant

`product.md` describes the present, not the history — there is no `[dropped]` status.

- **Changed** (works differently now, or the wish changed) → rewrite the line in place.
- **No longer relevant** (feature removed, wish abandoned) → delete the line.
- **Why** it was dropped or changed is non-obvious and an agent might propose it again → that is a decision: record it with `/lore-adr` ("we don't do X because …", zones of the scenario or `global`).
- If the code still implements a removed `[live]` scenario, the code change goes through a lore change (`/lore-propose`), and the scenario is removed at its archive step, not before.

History stays in archived changes (`## Scenarios` of their proposals) and git.

## List what is planned

Read the scenarios with `[planned]` and present them in order with their zones.

## Always

Show the change, get the owner's yes, then `lore lint` (it checks that every zone in `zones:` exists).
