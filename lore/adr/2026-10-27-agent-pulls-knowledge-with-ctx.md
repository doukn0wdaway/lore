---
id: 2026-10-27-agent-pulls-knowledge-with-ctx
status: accepted
zones: [decisions-model, workflows, install]
tldr: "No prompt hook: the agent runs lore ctx with its own description of the work or question when the owner asks it to look something up or it needs more; one selection scores every zone (tldr with its invariants, boundaries and contracts) and every accepted ADR, and prints the decisions as content, then the chosen zones whole."
rules:
  - "DO: run lore ctx when the owner asks to look something up or when you need more project knowledge"
  - "DO: pass lore ctx your own description of the work or question"
  - "DON'T: pass the owner's message to lore ctx verbatim"
  - "DON'T: bring back a prompt hook or any other automatic injection on the owner's messages"
  - "DO: show the selected ADRs and zones as content, not as a list of files to read"
  - "DO: print a chosen zone whole, not pieces of it"
  - "DO: judge a zone by its tldr together with its invariants, boundaries and contracts"
  - "DON'T: make ADRs share one budget with zones"
  - "DON'T: add a stage that narrows the ADRs by zone, or a cache, before the log shows ctx is too slow or too noisy"
supersedes: [2026-10-27-inject-content-per-requirement]  # ids of ADRs this one replaces entirely
superseded_by: null
inferred: false             # true — inferred from code, the owner could not tell why
ref: "change ctx-on-demand"
---
# The agent pulls knowledge with lore ctx

## Context
Knowledge used to be pushed by a `UserPromptSubmit` hook on every owner message, selected in two stages (zones by tldr, then their requirements, sections and ADRs, plus global ADRs) under one shared budget. The hook fires only on the owner's messages and skips workflow calls (`/…`) and short messages, so it is absent from most of the real work. In a real `lore ctx` run the ADRs that mattered lost the shared budget to zone sections and arrived as names only.

## Decision
There is no prompt hook. The agent runs `lore ctx "<text>"` when the owner asks it to look something up or when it decides it needs more; the text is the agent's own description of the work or question, and the model judges which knowledge relates to it. One selection scores every zone — by its tldr together with its Invariants, Boundaries and Contracts — and every accepted ADR. It prints two lists: the decisions as content (tldr and rules) with no budget, then the chosen zones whole within `ctx.budget_tokens`; the rest is named.

## Why
The hook is not needed on every owner message: knowledge is needed only when the owner asks the agent to look something up, or when the agent decides it needs more information. The owner's message is not the right query: what matters is which of the project's knowledge relates to the question, so the agent describes it in its own words. Every ADR is scored, with no zone stage in front of it, because for now fewer misses matter more; the owner wants to see how it behaves in real use. Decisions get their own list because with one budget they lost to zone sections. Whole zones instead of pieces: it is simpler, and nothing has to be tuned about which pieces to show. A zone is judged with its invariants, boundaries and contracts, not by the tldr alone, so that a zone related to the work only through an invariant or a neighbour is not lost — without making the tldr in `INDEX.md` longer. Content rather than "go read X" pointers still holds from the superseded decision: each pointer costs the agent extra steps.

## Rejected alternatives
### Keep the prompt hook
Why not: it fires only on the owner's messages, skips workflows and short messages, and is not needed on every message.

### Score ADRs only of the zones picked first
Why not: it can drop the ADRs of a zone that scored low; for now fewer misses matter more than speed.

### Pieces of zones (requirements and sections scored one by one)
Why not: harder to get right — broad sections such as Contracts and Boundaries scored high for almost any text — and whole zones are simpler.

### Longer tldrs that name invariants and neighbours
Why not: the tldr is in `INDEX.md` in every session, and it would have to be kept in sync by hand; the model reads the sections themselves instead.

### One shared budget for everything
Why not: ADRs lost the budget to zone sections and arrived as names only.

### Pointers only ("read X")
Why not: each pointer costs the agent extra steps.

## Revisit if
- The call log (`lore/.local/log.jsonl`) shows `lore ctx` too slow or too noisy.
- Zones judged with their invariants, boundaries and contracts turn out badly calibrated (long candidate texts were never benchmarked).
