## MODIFIED Requirements

### Requirement: Product scenarios
`product.md` SHALL describe what the application does for whom (`## What it is`, `## Who it is for`) and list user-visible scenarios under `## Scenarios`, one bullet each with a status (`[live]` or `[planned]`) and a `zones:` line naming the zones the scenario passes through. A `zones:` line belongs to the scenario bullet above it. In a `[planned]` scenario, a zone name ending in `?` (`receipts?`) is a future zone — one the scenario will need but that doesn't exist yet.

#### Scenario: Unknown zone in a scenario
- **WHEN** a `zones:` line names a zone without `?` that doesn't exist
- **THEN** lint reports an error at that line, whatever the scenario's status

#### Scenario: Future zone in a planned scenario
- **WHEN** a `[planned]` scenario names `receipts?` and no zone `receipts` exists
- **THEN** lint accepts it

#### Scenario: Future zone now exists
- **WHEN** a `[planned]` scenario names `receipts?` and the zone `receipts` exists
- **THEN** lint reports a warning at that line: drop the `?`

#### Scenario: Future zone outside a planned scenario
- **WHEN** a zone name ending in `?` appears in a scenario that is not `[planned]` (a `[live]` one, or one without a status)
- **THEN** lint reports an error at that line: a live scenario goes only through existing zones

#### Scenario: Template example
- **WHEN** scenarios sit inside an HTML comment (as in the template)
- **THEN** they are ignored

#### Scenario: Missing product
- **WHEN** `product.md` doesn't exist
- **THEN** lint reports an error

## MODIFIED Sections

### Section: Gotchas
- Headings inside fenced code blocks are ignored, so an example zone inside ```` ```markdown ```` does not create requirements.
- Sections are matched by title case-insensitively, and the first heading with that title wins.
- A section runs until the next heading of the same or a higher level, so `#### Scenario:` blocks belong to their `### Requirement:`.
- Lint doesn't check `architecture.md` at all. In `product.md` it checks only the `zones:` lines; it reads a scenario's status to decide whether `?` is allowed, but doesn't validate the status values themselves.
- `INDEX.md` must match byte for byte; any hand edit makes lint fail until `lore index` is run.
- A zone without an `id` falls back to its file name as the key, so a broken zone still shows up in the index and in errors.
