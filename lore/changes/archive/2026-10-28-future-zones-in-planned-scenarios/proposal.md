# Future zones in planned scenarios

## Why
`/lore-product` told the agent to mark a zone that doesn't exist yet with `?` in a scenario's `zones:` line (`zones: storage, finance, receipts?`), but lint treated `receipts?` as an unknown zone and reported an error: the tool and the workflow contradicted each other. Since `--finish` requires the whole project to be lint-clean (ADR 2026-10-27-finish-requires-clean-lint), every planned scenario with a future zone blocked the archive of any change. Dropping the marker loses the hint about which zones are still to be created; creating stub zones up front fixes zone boundaries before the proposal that should define them.

Serves: "Owner adds, updates, splits or merges a zone, or adds a wanted scenario and later marks it live" and "Owner checks the docs".

## What changes
- Lint reads the status (`[live]`, `[planned]`) of the scenario bullet each `zones:` line belongs to.
- In a `[planned]` scenario, a zone name ending in `?` (`receipts?`) is a future zone: lint accepts it while no zone with that id exists.
- In a `[planned]` scenario, a future zone whose zone now exists is a lint warning: drop the `?`.
- A zone name ending in `?` in a scenario that is not `[planned]` (a `[live]` one, or one without a status) is a lint error.
- A zone name without `?` that doesn't exist is still a lint error, whatever the status.
- `/lore-product` marks a future zone with `?` only in a `[planned]` scenario, tells the owner a new zone will be needed and doesn't create it; marking a scenario `[live]` leaves no `?`.
- `/lore-zone`, when it creates a zone, drops the `?` where `product.md` names it as a future zone.
- `/lore-archive`, when a scenario becomes `[live]`, leaves no `?` in its `zones:` line: the zone the change created, or the zones the scenario really passes through.
- The commented example in the `product.md` template shows and explains the `?` marker.

## Zones
- docs
- workflows

No delta for `cli` (it owns the lint report, not the product check), `install` (it copies the template; its content belongs to `docs`), `changes` (the merge into `product.md` is the agent's step in `/lore-archive`), `adrs` (untouched).

## Owner notes
> Option 4: lint understands `?`; in a `[live]` scenario it is fixed through code (the zone must exist before the scenario is live). — owner, 2026-10-28

> Future zones are marked instead of created as stubs: zone boundaries are better defined in propose. — owner, 2026-10-28

> Code-first. — owner, 2026-10-28
