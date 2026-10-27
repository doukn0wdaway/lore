# Edits ask only for ADRs

## Why
Every edit of a zone, `product.md` or `architecture.md` asks the owner through Claude Code's `permissions.ask`, so the sync at archive turns into dozens of prompts. They duplicate the gate: the owner already approves the whole diff of `lore/` before `--finish`, and `--finish` verifies the merge line by line. Serves the scenario "Owner archives a change at the gate".

## What changes
- `lore init` and `lore upgrade` add a `permissions.ask` rule only for `Edit` of `adr/**` in the docs folder; zones, `product.md` and `architecture.md` are edited without a prompt.
- `lore init` and `lore upgrade` remove lore's own old `ask` rules for `Edit` of `zones/**`, `product.md` and `architecture.md` from `.claude/settings.json`, keeping every other rule and setting.
- `/lore-bootstrap` and `/lore-archive` no longer tell the agent to expect a permission prompt on every edit of `lore/`; only edits of `lore/adr/` ask.

## Zones
- install

No delta for `workflows`, `adrs`, `docs`: none of their requirements describe permission prompts; only the texts of two workflows change.

## Scenarios
- changed: "Owner sets up lore in a project → docs skeleton, agent protocol, permissions and skills are in place" — the permissions now guard only ADR edits (the line itself doesn't change)

## ADR drafts
- 2026-10-27-edits-ask-only-for-adrs (supersedes 2026-10-27-read-freely-edit-asks)

## Owner notes
> "почему на zones всё надо аппрувать руками? можно починить это как-то?" / "я бы даже сказал не на zones, а на синке в целом" — owner, 2026-10-27

> Option B: drop the per-edit prompt, rely on the gate; keep it for `adr/**` — owner, 2026-10-27

> Do it as a draft in a new change, so the ADR takes effect together with the code — owner, 2026-10-27
