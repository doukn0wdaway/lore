---
id: docs
tldr: "The law documents other than ADRs: zone docs (id/tldr frontmatter, required ## Purpose, ### Requirement names unique per zone, entry points that exist), product.md scenarios that name existing zones, architecture.md, the generated INDEX.md, and lore show <zone>."
---
# Docs

## Purpose
Owns the documents that describe what the project is and how it behaves today: zone docs, `product.md`, `architecture.md`, and the generated `INDEX.md` that lists the zones for the agent. Defines their format and how they are read, printed and checked.

## Boundaries
- ADR files, their format and statuses → `adrs`. This zone only shows ADRs next to a zone.
- Changing these docs during normal work (deltas, the archive gate) → `changes`.
- Deciding what a zone is, the actor and reaction rules, writing zone docs and scenarios → `workflows` (`/lore-zone`, `/lore-product`, bootstrap).
- Templates being copied into a new project → `install`; their content belongs here.
- Picking zones and requirements for the agent's context → `decisions-model`.

## Requirements

### Requirement: Zone doc format
A zone doc SHALL be `zones/<id>.md` with frontmatter `id` (equal to the file name, lowercase kebab-case, not `global`) and `tldr`, and a non-empty `## Purpose`. All other sections are optional and free-form; `## Requirements`, `## Invariants`, `## Contracts`, `## Entry points` and `## Gotchas` are the recommended ones.

#### Scenario: Reserved id
- **WHEN** a zone's id is `global`
- **THEN** lint reports an error: `global` is reserved for project-wide ADRs

#### Scenario: Missing purpose
- **WHEN** `## Purpose` is absent or holds only comments
- **THEN** lint reports an error

#### Scenario: Unparsable frontmatter
- **WHEN** a frontmatter value contains `: ` without quotes
- **THEN** lint reports the YAML error with a hint to quote the value

### Requirement: Requirements inside a zone
A requirement SHALL be a `### Requirement: <name>` heading; the name is its id and SHALL be unique and non-empty within the zone. Scenarios are `#### Scenario: <name>` with WHEN/THEN bullets.

#### Scenario: Duplicate requirement
- **WHEN** two requirements in one zone have the same name
- **THEN** lint reports an error pointing at both lines

### Requirement: Entry points exist
Every file path listed as a backticked bullet under `## Entry points` SHALL exist relative to the project root.

#### Scenario: Stale path
- **WHEN** an entry point names a file that was moved or deleted
- **THEN** lint reports an error at that line

#### Scenario: Symbols and globs
- **WHEN** an entry point is a symbol (`pkg.Func()`), a glob, or has no `/` and no `.`
- **THEN** it is not checked; a `:line` or `#anchor` suffix is cut before checking

### Requirement: Zone size guide
Lint SHALL warn when a zone doc is longer than 300 lines (consider splitting) or shorter than 50 lines (maybe a section of a neighbouring zone).

### Requirement: Product scenarios
`product.md` SHALL describe what the application does for whom (`## What it is`, `## Who it is for`) and list user-visible scenarios under `## Scenarios`, one bullet each with a status (`[live]` or `[planned]`) and a `zones:` line naming the zones the scenario passes through.

#### Scenario: Unknown zone in a scenario
- **WHEN** a `zones:` line names a zone that doesn't exist
- **THEN** lint reports an error at that line

#### Scenario: Template example
- **WHEN** scenarios sit inside an HTML comment (as in the template)
- **THEN** they are ignored

#### Scenario: Missing product
- **WHEN** `product.md` doesn't exist
- **THEN** lint reports an error

### Requirement: Generated index
`INDEX.md` SHALL list every zone with its `tldr`, sorted by id, under a "generated — do not edit" header, and nothing else. `lore index` regenerates it and reports whether it changed.

#### Scenario: Stale index
- **WHEN** a zone is added, removed or its `tldr` changes and `lore index` was not run
- **THEN** lint reports `INDEX.md` as out of date

#### Scenario: No zones
- **WHEN** there are no zones
- **THEN** the index says "No zones yet."

### Requirement: Show a zone
`lore show <zone>` SHALL print the zone doc followed by its active decisions — accepted ADRs of that zone and global ones — and then, separately, proposed ADRs marked as not binding yet.

#### Scenario: Unknown zone
- **WHEN** the zone doesn't exist
- **THEN** the command fails and lists the existing zone ids

## Invariants
- Every fact lives in one place: an action is described in the zone of the object acted upon, a reaction in the zone that reacts.
- Zone docs, `product.md` and `architecture.md` describe the present; history lives in archived changes, not in these docs. `product.md` has no `[dropped]` status — a scenario that is no longer wanted is deleted.
- Docs describe behaviour, not implementation: file paths appear only under `## Entry points`.
- All docs are in English.
- `INDEX.md` lists zones only, never ADRs.

## Contracts
- Gives everyone: the list of zones (id, tldr, file) and lookup by id.
- Gives `changes`: zone text parsed into `##` sections and `### Requirement:` blocks, for delta checks and merge verification; `INDEX.md` regeneration.
- Gives `decisions-model`: zones with their tldr, their `## Invariants`, `## Boundaries` and `## Contracts`, and the whole zone text.
- Gives `adrs`: existing zone ids for checking `zones` of an ADR.
- Gives `cli` (lint): all checks above.
- Takes from `adrs`: the ADRs that apply to a zone and their statuses, for `lore show`.

## Entry points
- `internal/lore/project.go` — loading zones, the zone and ADR models
- `internal/lore/markdown.go` — frontmatter, headings, sections, requirements
- `internal/lore/index.go` — `INDEX.md`
- `internal/lore/show.go` — `lore show`
- `internal/lore/lint.go` — zone, entry point, index and product checks

## Gotchas
- Headings inside fenced code blocks are ignored, so an example zone inside ```` ```markdown ```` does not create requirements.
- Sections are matched by title case-insensitively, and the first heading with that title wins.
- A section runs until the next heading of the same or a higher level, so `#### Scenario:` blocks belong to their `### Requirement:`.
- Lint doesn't check `architecture.md` at all, and doesn't check the `[live]` / `[planned]` status values in `product.md` — only the `zones:` lines.
- `INDEX.md` must match byte for byte; any hand edit makes lint fail until `lore index` is run.
- A zone without an `id` falls back to its file name as the key, so a broken zone still shows up in the index and in errors.
