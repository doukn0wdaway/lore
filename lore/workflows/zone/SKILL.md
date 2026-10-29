---
name: lore-zone
description: Create, update, split or merge a lore zone doc (lore/zones/<id>.md) with the owner's approval — keeps the zone format, the actor/reaction rules, ADR zone references, product.md and the index consistent. Use when the owner asks to add or document a zone, update what a zone says, or restructure zones.
---

# lore zone — create or change a zone

Zones are the project's law: change them only when the owner asks. Show the change, get a yes, then write.

## What a zone is

An area of responsibility with its own rules — not a folder, not a class.

- Purpose fits in 2–3 lines without "and" joining unrelated things.
- It has its own WHEN/THEN rules or invariants that can't be explained through another zone.
- Changes inside it usually need only the **contracts** of other zones.
- **Actor rule:** an action lives in the zone of the *object* acted upon ("user creates an order" → `orders`), not of the actor.
- **Reaction rule:** a cross-zone link is described by the zone that *reacts* (`orders`: "WHEN `user.deleted` THEN cancel active orders"); the emitting zone lists the event under `## Contracts`.
- Size guide: ~100–300 lines. Bigger → split; under ~50 → probably a section of a neighbour.

## Format

```markdown
---
id: <kebab-case, equals file name; "global" is reserved>
tldr: <one concrete sentence — what agents and the decisions model match tasks against>
---
# <Title>

## Purpose            (required, 2–3 lines)
## Boundaries         (what is NOT here → where)
## Requirements
### Requirement: <name — unique in the zone, it is the id>
<The system SHALL ...>
#### Scenario: <name>
- **WHEN** ...
- **THEN** ...
## Invariants
## Contracts
## Entry points       (3–5 starting places; paths must exist)
## Gotchas
```

Quote frontmatter values that contain `: ` (`tldr: "Notes: search, read, edit"`) — otherwise the YAML breaks.

Only frontmatter and `## Purpose` are required; add free sections when the zone needs them (`## Tokens`, `## Components`, ...). Describe behaviour, not implementation — names of functions and files belong only in `## Entry points`.

## Create

1. Check the zone criterion and that no existing zone already owns this (`lore/INDEX.md`). Propose `id`, `tldr`, boundaries → owner approves.
2. Read the code, write the doc.
3. Update `lore/architecture.md` (zones and dependencies) and any `product.md` scenario that goes through the new zone; where it is named as a future zone (`<id>?`), drop the `?`.

## Update

Change only what the owner asked for; keep requirement names stable (other docs and future changes refer to them). If the update reverses something an ADR says, stop and use `/lore-adr` — the decision changes first.

## Split / merge / rename

1. Owner approves the new map (ids, tldrs, boundaries).
2. Move requirements to their new owners, keeping their names.
3. Fix references: `zones:` in every ADR (`lore adr list --zone <old>`), `zones:` lines in `product.md`, `architecture.md`.
4. Delete the old file(s).

## Always finish with

```sh
lore index
lore lint
```

Both clean, then show the owner the diff of `lore/`.
