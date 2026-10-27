---
name: lore-audit
description: Check lore docs against the actual code — for one zone or all — and report where requirements, invariants, entry points or ADR rules no longer match what the code does. Reports only; fixes go through the owner. Use when the owner asks to audit/verify/check lore docs, suspects docs are stale, or after a period of coding without updating docs.
---

# lore audit — docs vs code

Read-only by default: find discrepancies, report them, change nothing until the owner decides.

## 1. Scope

A zone the owner names, or every zone from `lore/INDEX.md` one by one. Start with `lore lint` — mechanical problems (dead entry points, stale anchors, broken references) come first.

## 2. For each zone

`lore show <zone>`, then read the code from its `## Entry points` outwards. For every claim, decide:

| Verdict | Meaning |
|---|---|
| ok | the code does what the doc says |
| stale | the code does something else now |
| missing | the code has behaviour the doc doesn't mention (worth a requirement) |
| unverifiable | can't tell from the code (config, external service) |

Check `## Requirements` (each scenario), `## Invariants`, `## Contracts`, `## Boundaries` (does the code respect them?).

Also check `lore/product.md`: every `[live]` scenario through this zone still works; a `[planned]` one may already be live.

For each active ADR of the zone: does the code follow its `rules`? A violation is **not** automatically a doc problem — it is either a bug or a decision that silently changed. Only the owner can tell which.

## 3. Report

Group by zone, most serious first:

```
## notifications
- stale — Requirement "Payment reminder": doc says 1 day before, code uses 3 days (internal/notify/rules.go)
- missing — retries with backoff on delivery failure, not described
- ADR 2026-10-27-crons-enqueue-only — violated in internal/jobs/digest.go: cron calls transport.Send()
  → bug or new decision?
```

Cite file paths so the owner can check. Don't pad the report with "ok" lines — just count them.

## 4. Fixing (only on the owner's word)

- stale / missing in a zone → `/lore-zone` (update);
- ADR violated, owner says "bug" → fix the code;
- ADR violated, owner says "new decision" → `/lore-adr` (supersede);
- a `product.md` scenario is stale, removed or already live → `/lore-product`.
