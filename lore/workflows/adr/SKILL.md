---
name: lore-adr
description: Record an architectural decision as a lore ADR — check that it deserves one, find conflicting decisions, ask the owner why, write it, and accept it only after the owner's yes; also replaces (supersedes) an existing ADR. Use when the owner says "record/remember this decision", "write an ADR", "we decided X", or when a fork in the work was resolved by the owner and the answer should outlive the current task.
---

# lore adr — record a decision

## 1. Does it deserve an ADR?

Both must hold:

1. the decision has a **non-obvious "why"**;
2. a future agent looking at the code **might want to "fix" it** — and that would be a mistake.

Whether alternatives were considered doesn't matter. If it fails the test, say so and offer the right place instead:

- an obvious rule ("dates in UTC") → `## Invariants` of the zone (`/lore-zone`);
- style / UI conventions → the design-system zone;
- a local oddity in one function → a code comment next to it.

## 2. Find what it touches

- Pick the zones from `lore/INDEX.md`; `global` if it is project-wide.
- `lore adr list --zone <zone>` for each, read the full text of every ADR that may overlap.
- **Conflict with an accepted ADR?** Ask the owner: *"This contradicts ADR `<id>` (<tldr>). Is this a new decision that replaces it, or should we keep the old one?"* Never decide yourself.
  - Replaces → the new ADR **supersedes the old one entirely**: restate in the new ADR every rule of the old one that still holds (no partial replacement).
  - Keep the old one → stop; nothing to record.

## 3. Ask why — never invent it

Ask the owner, one question at a time, only what you don't know yet:

- Why this way?
- Were other options considered? Why not those? (Only record alternatives that were really considered.)
- What do we knowingly pay for it? When should it be revisited?

If the owner doesn't know the reason: `## Why` = `Unknown (inferred from code).`, `inferred: true`.

## 4. Write it

```sh
lore adr new <slug> --zones a,b [--title "Human title"] [--supersedes <old-id>]
```

Fill the created file (English). Quote frontmatter values that contain `: `:

- `tldr` — one concrete sentence, the decision itself (it is what agents see first and what the decisions model matches against);
- `rules` — operational `DO:` / `DON'T:` lines an agent can follow without reading the body;
- `## Context`, `## Decision`, `## Why` — required, faithful to the owner's words;
- `## Rejected alternatives`, `## Cost`, `## Revisit if` — only if there is content; delete empty ones.

Run `lore lint` and fix every error for this file.

## 5. Owner's yes → accept

Show the owner the `tldr`, `rules` and `Why`. Only after an explicit yes:

```sh
lore adr accept <id>
```

It sets `accepted` and marks superseded ADRs (`status: superseded`, `superseded_by`). Never run it without the owner's yes.

## 6. Anchors

If the rule is easy to break at a specific place in code, propose an anchor comment there and add it after the owner agrees:

```go
// ADR <id>: <the rule in a few words>
```

If the new ADR supersedes an old one, find anchors of the old one (`grep -rn "ADR <old-id>"`) and repoint them — `lore lint` fails on anchors to superseded ADRs.

Finish with `lore lint` clean.
