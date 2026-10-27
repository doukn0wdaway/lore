---
id: 2026-10-27-lore-owns-its-key
status: accepted
zones: [decisions-model]
tldr: "lore uses its own API key, separate from the application's environment: LORE_OPENROUTER_API_KEY from the environment, lore/.env or ~/.config/lore/.env; the project's .env is never read and a key written into lore.yaml is refused."
rules:
  - "DO: put the key into ~/.config/lore/.env or lore/.env"
  - "DON'T: read the application's .env"
  - "DON'T: allow the key itself in lore.yaml — it holds only the variable name"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# lore owns its API key

## Context
lore needs an OpenRouter key for the decisions model. The project being documented usually has its own `.env` with its own keys, and `lore.yaml` is committed.

## Decision
The key is lore's own. It is looked up in the environment variable named by `decisions_api_key_env`, then in the project's `lore/.env` (gitignored), then in `~/.config/lore/.env`. The application's `.env` is never read. A value in `decisions_api_key_env` that looks like a key fails every model command and lint, without echoing it.

## Why
The project's `.env` belongs to the application. A key written into the config leaks into git.
