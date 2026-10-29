## MODIFIED Requirements

### Requirement: The archive gate
`/lore-archive` SHALL be the only way the law changes during normal work: pre-check, reconcile deltas with the real code (what was built, not what was planned), check facts against decisions, settle every ADR draft, merge deltas into zones (MODIFIED replaces the whole block) and update `product.md` and `architecture.md`, repoint code anchors of superseded ADRs, `lore index` and `lore lint`, the owner's yes on the diff, `--finish`.

#### Scenario: Unexplained deviation
- **WHEN** the code differs from the deltas and `## Owner notes` doesn't explain it
- **THEN** the agent asks the owner whether it is intended

#### Scenario: Planned scenario with a future zone goes live
- **WHEN** a `[planned]` scenario naming `receipts?` becomes `[live]` at archive
- **THEN** its `zones:` line is left without `?`: the zone the change created, or the zones the scenario really passes through

### Requirement: Maintaining zones and product
Zones and `product.md` SHALL change outside a change only through `/lore-zone` and `/lore-product` at the owner's request, showing the change and getting a yes first. Requirement names stay stable; a split or merge moves requirements under their names and fixes `zones:` in ADRs and `product.md`.

#### Scenario: Scenario no longer wanted
- **WHEN** a scenario is abandoned
- **THEN** its line is deleted; if the reason is non-obvious and it might be proposed again, the owner is asked to record it as an ADR

#### Scenario: Update reverses a decision
- **WHEN** a zone update would contradict an ADR
- **THEN** the agent stops and goes through `/lore-adr` first

#### Scenario: Wanted scenario needs a zone that doesn't exist
- **WHEN** `/lore-product` adds a `[planned]` scenario that goes through a zone not in the index
- **THEN** the zone is written with `?` (`receipts?`) and the owner is told a new zone will be needed; no zone is created — its boundaries are defined later, in the proposal that builds it

#### Scenario: Future zone is created
- **WHEN** `/lore-zone` creates a zone that `product.md` names as `receipts?`
- **THEN** the `?` is dropped in every scenario that names it
