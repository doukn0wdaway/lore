---
name: lore-bootstrap
description: Bootstrap lore docs for an existing project — derive product scenarios, the zone map, architecture and zone docs from the code, then interview the owner about the "why" of non-obvious decisions and record them as ADRs. Use when the user asks to bootstrap lore, document an existing project with lore, or resume an unfinished bootstrap.
---

# lore bootstrap

Build the project's knowledge base in `lore/` (or the `root` from `lore.yaml`) from an existing codebase. You write; the owner approves. The work spans several sessions: progress lives in `lore/.local/bootstrap.json` (gitignored).

## Ground rules

- **All docs are in English**, even if the owner talks to you in another language.
- **You may derive *what* was decided from code. You may never invent *why*.** Reasons come only from the owner. If the owner doesn't know, record that honestly.
- **One step at a time, approval at each gate.** Never run ahead to the next step before the owner approves the current one.
- **Existing docs are hints, not truth** (old specs, ADRs, READMEs, `docs/`). Verify against the code; the code and the owner win.
- **Describe behaviour, not implementation.** No function names, props, table layouts in requirements. Paths belong only in `## Entry points`.
- Edits to `lore/adr/` prompt the owner for permission — that is expected; zones, `product.md` and `architecture.md` are approved at each step's gate instead.
- After each step: `lore index` and `lore lint`. Fix every error before asking for approval.

## Resume first

Read `lore/.local/bootstrap.json` if it exists (older setups: `.lore/bootstrap.json` — `lore upgrade` moves it) and continue from the recorded step. Otherwise start at step 1 and create it:

```json
{
  "step": 1,
  "zones_approved": [],
  "zones_done": [],
  "adr_candidates": [],
  "adr_done": [],
  "notes": ""
}
```

Update it after every approval. `adr_candidates` holds short descriptions of decisions still to ask about; `notes` holds anything the next session must know.

## Step 1 — Product (`lore/product.md`)

Derive what the application does for its users: from routes, commands, handlers, UI screens, README.

- `## What it is` — 2–3 lines. `## Who it is for` — 1–3 lines.
- `## Scenarios` — one item per user-visible scenario, a route through zones without rules:

  ```markdown
  - [live] User sends a voice message → gets a text answer
    zones: transport, storage, ai-core
  ```

  Status `[live]` for what works now. Add `[planned]` only for what the owner names as wanted. Zone names are provisional until step 2 — fix them after the zone map is approved.

**Gate:** show the scenario list, ask what is missing or wrong. Approved → step 2.

## Step 2 — Zone map (the main gate)

A **zone** is an area of responsibility with its own rules — not a folder, not a class.

- Its purpose fits in 2–3 lines without "and" joining unrelated things.
- It has its own WHEN/THEN rules or invariants that can't be explained through another zone.
- Changing something inside it usually needs only the **contracts** of other zones, not their internals.
- A zone is centred either on an entity (`users`, `orders`) or on a capability (`notifications`, `transport`, `ai-core`, `design-system`).
- **Actor rule:** an action lives in the zone of the *object* acted upon, not of the actor. "User creates an order" → `orders`. `users` keeps only what concerns the user itself: sign-up, profile, roles, own settings, deletion. Otherwise the actor's zone swallows half the app.
- **Reaction rule:** a cross-zone link is described by the zone that *reacts*. `users` lists "emits `user.deleted`" in Contracts; `orders` has "WHEN `user.deleted` THEN cancel active orders".
- Ids: lowercase kebab-case, `global` is reserved. Expect roughly 8–25 zones; a zone doc of ~100–300 lines.

Present the map as one table — **no zone files yet**:

| id | tldr (one concrete sentence) | boundaries (what is NOT here → where) |
|---|---|---|

Quote frontmatter values that contain `: ` (`tldr: "Notes: search, read, edit"`) — otherwise the YAML breaks.

`tldr` must be specific: it is what the decisions model matches tasks against. "Notifications stuff" is useless; "Scheduled and event-driven notifications; crons only enqueue, a worker delivers via transport" is good.

**Gate:** this is the most important approval — a crooked map makes everything after it crooked, and it is cheapest to fix now. Iterate until the owner approves. Record the ids in `zones_approved`, then fix zone names in `product.md`.

## Step 3 — Architecture (`lore/architecture.md`)

Overview, zones and dependencies (who relies on whom, through what contract; a mermaid diagram is welcome), key technologies, project-wide conventions. Plain document, no rules that belong to a zone.

**Gate:** owner approves → step 4.

## Step 4 — Zones, one at a time (`lore/zones/<id>.md`)

For each approved zone, in the order the owner prefers (default: the core ones first), read its code and write:

```markdown
---
id: notifications
tldr: Scheduled and event-driven notifications; crons only enqueue, a worker delivers via transport.
---
# Notifications

## Purpose
What the zone is responsible for, 2–3 lines.

## Boundaries
What is NOT here and where it lives instead.

## Requirements

### Requirement: Payment reminder
The system SHALL remind the user about a recurring payment before the charge.

#### Scenario: Reminder three days before
- **WHEN** a recurring payment is due in 3 days
- **THEN** a reminder is enqueued for the payment owner

## Invariants
- Things that must always hold.

## Contracts
- What the zone gives to and takes from other zones (events, calls).

## Entry points
- `internal/notify/scheduler.go` — cron registration
- `notify.Enqueue()` — the only way to send

## Gotchas
- Non-obvious traps.
```

- **Required:** frontmatter `id` (equals the file name) and `tldr`, and a non-empty `## Purpose`. Everything else is recommended; add free sections when the zone needs them (e.g. `## Tokens`, `## Components` for a design system).
- `### Requirement: <name>` — the name is the requirement's id, unique within the zone. 1–4 scenarios each, `WHEN`/`THEN`, `SHALL` not "should".
- `## Entry points` — 3–5 places to start reading, not a full file map. Every path must exist (lint checks it).
- While reading the code, **collect ADR candidates** into `adr_candidates`: non-obvious constraints, odd workarounds, "why is there a queue here", choices a future agent would be tempted to "simplify". Don't write ADRs yet.

**Gate:** each zone is approved separately. Approved → add to `zones_done`, next zone. All done → step 5.

## Step 5 — Decision interview (ADRs)

A decision deserves an ADR when **both** hold: it has a non-obvious "why", and a future agent looking at the code might want to "fix" it — and that would be a mistake. Alternatives need not have been considered. Not ADRs: obvious rules (→ zone `Invariants`), style (→ design-system zone), tiny local oddities (→ a code comment).

Go through `adr_candidates` **one question at a time**:

> In `notifications`, crons never send directly — they enqueue and a worker delivers. Why was it done this way? Were other options considered?

Then, depending on the answer:

- **Owner explains** → `lore adr new <slug> --zones <a,b>` and fill it in: `tldr`, `rules` (DO/DON'T), `## Context`, `## Decision`, `## Why` (the owner's reasoning, faithfully), `## Rejected alternatives` only for options really considered, `## Cost` / `## Revisit if` if the owner mentions them. Show it to the owner; on yes run `lore adr accept <id>` (it sets `accepted`; never edit the status by hand).
- **Owner doesn't remember** → still record it: `## Why` = `Unknown (inferred from code).`, `inferred: true`, then `lore adr accept <id>` once the owner confirms the decision itself is right. It tells future agents "deliberate — don't touch without asking".
- **Owner says it's not a real decision / should change** → no ADR; note it in `notes` as possible future work.

`zones`: only existing zone ids, or `[global]` for project-wide decisions. Delete optional sections you didn't fill. For more detail on writing an ADR see the `lore-adr` skill. Move each finished candidate to `adr_done`.

**Gate:** none beyond the answers themselves; show the list of ADRs created at the end.

## Step 6 — Code anchors

Propose 5–10 places in the code where an ADR rule is easy to break, and add a comment there:

```go
// ADR 2026-10-27-crons-enqueue-only: never call transport.Send() here, only enqueue.
```

Only where a violation is likely — not on every file. **Gate:** the owner approves the list before you edit code.

## Finish

`lore index`, `lore lint` (clean), then summarise: zones, ADRs (how many with unknown "why"), anchors, and what is left in `notes`. Set `"step": "done"` in `lore/.local/bootstrap.json`.
