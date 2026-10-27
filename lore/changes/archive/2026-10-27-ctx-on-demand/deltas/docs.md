## MODIFIED Sections

### Section: Contracts
- Gives everyone: the list of zones (id, tldr, file) and lookup by id.
- Gives `changes`: zone text parsed into `##` sections and `### Requirement:` blocks, for delta checks and merge verification; `INDEX.md` regeneration.
- Gives `decisions-model`: zones with their tldr, their `## Invariants`, `## Boundaries` and `## Contracts`, and the whole zone text.
- Gives `adrs`: existing zone ids for checking `zones` of an ADR.
- Gives `cli` (lint): all checks above.
- Takes from `adrs`: the ADRs that apply to a zone and their statuses, for `lore show`.
