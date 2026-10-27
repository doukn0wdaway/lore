---
id: 2026-10-27-pplx-decider-only
status: accepted
zones: [decisions-model]
tldr: "The decisions model is one model, perplexity/pplx-decider-v1.2 through OpenRouter's Decisions API — no faster flash model, no per-stage or fallback model."
rules:
  - "DO: keep decisions_model at pplx-decider unless the owner decides otherwise"
  - "DON'T: switch to a flash model or add a cheaper model for one of the stages to cut latency"
  - "DON'T: enable provider fallbacks"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# One decisions model: pplx-decider

## Context
The selection asks a decisions model many yes/no questions per prompt. Faster and cheaper models are available, and the two stages could use different ones.

## Decision
One model, `perplexity/pplx-decider-v1.2`, through OpenRouter's Decisions API, with the provider order fixed and fallbacks disabled.

## Why
It had the best recall on the benchmark; the flash model loses half of the ADRs.

## Rejected alternatives
### A flash model (alone or for one stage)
Why not: loses half of the ADRs.
