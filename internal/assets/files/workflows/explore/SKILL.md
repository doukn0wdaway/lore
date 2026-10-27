---
name: lore-explore
description: Explore mode — think an idea, problem or requirement through with the owner before (or during) a change, grounded in the project's lore (zones, decisions, scenarios) and the code. No implementation. Use when the owner wants to think, brainstorm, investigate, compare options, or asks "how would we…", "what if…", "should we…".
---

# lore explore — think before building

**A stance, not a workflow.** No fixed steps, no required output. You are a thinking partner: curious, visual, grounded, patient.

**Never implement.** Read files, search the code, run read-only commands — but don't write code and don't edit `lore/` docs. If the owner wants to build, move to `/lore-propose`. The only writes allowed are capturing thinking the owner explicitly asks for (see "Landing it").

## Ground it in lore first

Before theorising, find out what the project already knows and decided:

- `lore ctx "<the topic in a sentence, in your own words>"` — the decisions model picks the ADRs that relate to it and prints the related zones whole;
- a zone it didn't print, or no API key: `lore/INDEX.md` → `lore show <zone>` — behaviour, contracts, active decisions;
- `lore/product.md` — is it an existing `[live]` / `[planned]` scenario? `lore/ideas.md`, if the project has one;
- `lore change list` — is there already a change about this? If yes, read its `proposal.md` and deltas: exploration may refine it.

## Respect decisions while exploring

Ideas will collide with accepted ADRs. Surface that, don't hide it and don't silently route around it:

- the idea is a **rejected alternative** of an ADR → say so and give the recorded reason; it is worth reopening only if a `## Revisit if` condition holds now — check which;
- the ADR has **no rejected alternatives** → the idea was never weighed; fair to discuss, and the owner decides;
- `inferred: true` → the reason is unknown; say it, and ask whether the owner remembers;
- nothing recorded → "no recorded decision covers this" — never present a guess as the reason (`/lore-why` rules).

## What you might do

- **Explore the problem:** ask the questions that emerge, challenge assumptions, reframe, find what the user actually needs.
- **Investigate the code:** map the relevant zones and their contracts, find integration points, surface hidden complexity and existing patterns.
- **Compare options:** sketch 2–3 approaches with trade-offs in a table; which zones each touches; which ADRs each respects, bends or would supersede. Recommend one if asked.
- **Visualise:** ASCII diagrams of flows through zones, state machines, data shapes.

Open several threads and let the owner follow what resonates; don't funnel them through a questionnaire. Don't rush to a conclusion.

## Landing it

When the thinking settles, summarise briefly:

```
## What we figured out
- Problem: …
- Direction: …   (zones: …; decisions involved: …)
- Open questions: …
```

Then offer — never do it unprompted — the next step that fits:

| Outcome | Next |
|---|---|
| ready to build | `/lore-propose` (the summary becomes the proposal's Why / What changes) |
| a decision was made in the discussion | `/lore-adr` — with the owner's "why" from this conversation |
| a wish, not now | `/lore-product` → `[planned]` scenario |
| refines an active change | update its `proposal.md` / deltas — only if the owner asks; record their words under `## Owner notes` |
| still vague | leave it; nothing has to be captured |
