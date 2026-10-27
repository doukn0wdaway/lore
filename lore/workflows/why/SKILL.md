---
name: lore-why
description: Answer "why is it done this way?" about the project from recorded lore decisions (ADRs) and zone docs, citing them — and say plainly when no reason is recorded instead of guessing. Use when the owner or you ask why some code, behaviour or architecture is the way it is, or before "simplifying" something that looks odd.
---

# lore why — explain a decision

## 1. Find the area

From `lore/INDEX.md` pick the zone(s) the question is about. `lore show <zone>` prints the zone and its active decisions (own and global).

## 2. Find the decision

- Read the full text of every ADR whose `tldr` / `rules` may answer the question.
- Also check the zone's `## Invariants`, `## Contracts`, `## Gotchas` — small "whys" live there.
- `lore adr list --status superseded` if the question is about something that *used* to be different.
- Code anchors: `grep -rn "ADR " <path>` near the code in question.

## 3. Answer

**Recorded decision found** → answer from it and cite the id:

- the reason (`## Why`), in the ADR's terms;
- rejected alternatives, if the question is "why not X" and X is among them → X was considered and rejected; don't propose it again unless a `## Revisit if` condition now holds — then say which one;
- no `## Rejected alternatives` section → alternatives were never weighed; you may suggest one, but as a question to the owner;
- `inferred: true` → say the reason is unknown: the decision was deliberate, but nobody remembers why.

**Nothing recorded** → say so plainly: *"No recorded decision covers this."* Then you may:

- look for hints in `git log -S` / `git blame` and commit messages — and label them as hints, not the reason;
- offer to record the reason once the owner explains it: `/lore-adr`.

**Never present a guess as the reason.**
