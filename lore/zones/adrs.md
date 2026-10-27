---
id: adrs
tldr: "Decision records in lore/adr/<date>-<slug>.md: format with a required ## Why, zones or global, status changed only by lore adr accept (proposed → accepted, replaced ADRs → superseded), drafts inside a change, lore adr list, and ADR <id> anchors in code that must point at live ADRs."
---
# ADRs

## Purpose
An ADR records *why* the project is built a certain way, so that a future agent does not "fix" a deliberate decision. This zone owns the ADR format, its lifecycle (proposed → accepted → superseded), the commands that create, accept and list ADRs, and the code anchors that point at them.

## Boundaries
- When an ADR deserves to exist, how the agent asks the owner "why" and writes it → `workflows` (`/lore-adr`, bootstrap, propose).
- When drafts of a change are moved into `adr/` and supersede old ADRs → `changes` (archive). This zone only provides the move-independent rules and the supersede operation.
- Picking ADRs for the agent's context and checking facts against them → `decisions-model` and `changes`.
- Printing a zone with its ADRs (`lore show`) → `docs`.

## Requirements

### Requirement: ADR format
An ADR SHALL be a markdown file whose name is its id, `<YYYY-MM-DD>-<slug>`, with frontmatter `id`, `status`, `zones`, `tldr` (plus optional `rules`, `supersedes`, `superseded_by`, `inferred`, `ref`) and non-empty `## Context`, `## Decision` and `## Why` sections.

#### Scenario: Missing why
- **WHEN** `## Why` is absent or holds only comments
- **THEN** lint reports an error

#### Scenario: Id and file name differ
- **WHEN** the frontmatter `id` is not the file name, or is not `YYYY-MM-DD-slug`
- **THEN** lint reports an error

#### Scenario: Short tldr
- **WHEN** the `tldr` has fewer than 5 words
- **THEN** lint warns that it is too short to match tasks against

### Requirement: Zones of an ADR
An ADR SHALL name at least one zone in `zones`; each is an existing zone id or `global`. A draft inside a change may also name a zone that the change creates.

#### Scenario: Unknown zone
- **WHEN** `zones` names a zone that doesn't exist (and isn't created by the draft's change)
- **THEN** `lore adr new` refuses, and lint reports an error for an existing file

### Requirement: Create an ADR
`lore adr new <slug> --zones a,b` SHALL create an ADR from the template with status `proposed` and today's date in the id — in `adr/`, or with `--change <name>` as a draft in `changes/<name>/adr/`.

#### Scenario: Title from the slug
- **WHEN** no `--title` is given
- **THEN** the heading is the slug with dashes turned into spaces and the first letter capitalised

#### Scenario: Supersedes something invalid
- **WHEN** `--supersedes` names an ADR that is not in `adr/` (unknown, or only a draft) or is already superseded
- **THEN** nothing is created

#### Scenario: Id taken
- **WHEN** an ADR or draft with the same date and slug exists
- **THEN** nothing is created

### Requirement: Accept an ADR
`lore adr accept <id>` SHALL be the only way an ADR's status changes from `proposed` to `accepted`, and it SHALL refuse while the ADR file has lint errors.

#### Scenario: Accept an ADR in adr/
- **WHEN** the owner accepts an ADR in `adr/` that supersedes others
- **THEN** it becomes `accepted` and each ADR it supersedes gets `status: superseded` and `superseded_by: <id>`

#### Scenario: Accept a draft in a change
- **WHEN** the owner accepts a draft inside a change
- **THEN** only the draft becomes `accepted`; the ADRs it supersedes stay accepted until the change is archived

#### Scenario: Not proposed
- **WHEN** the ADR is already `accepted` or `superseded`
- **THEN** the command fails

#### Scenario: Unfilled ADR
- **WHEN** the ADR still has an empty `## Why` or another lint error
- **THEN** the command fails and lists the errors of that file

### Requirement: Supersede chain
A superseded ADR SHALL have `superseded_by` pointing at an existing ADR, and `superseded_by` SHALL only be set on a superseded ADR. `supersedes` SHALL name ADRs in `adr/`, never drafts. A draft can't be superseded.

### Requirement: List ADRs
`lore adr list` SHALL list the ADRs in `adr/` (not drafts) with id, status, zones and tldr, filtered by `--zone` (the zone's own and global ADRs) and `--status`, as a table or `--json`.

### Requirement: Code anchors
A comment `ADR <id>` anywhere in the project's files SHALL point at an existing ADR that is not superseded.

#### Scenario: Dead anchor
- **WHEN** an anchor names an ADR that doesn't exist
- **THEN** lint reports an error at that file and line

#### Scenario: Anchor to a superseded ADR
- **WHEN** an anchor names a superseded ADR
- **THEN** lint reports an error naming the ADR that replaced it

#### Scenario: Ignored paths
- **WHEN** a file matches `anchors_ignore` in `lore.yaml` (exact path, a folder above it, or a glob; a pattern without `/` matches the file name at any depth)
- **THEN** its anchors are not checked

## Invariants
- ADRs are immutable once accepted, apart from the status fields: a changed decision is a new ADR that supersedes the old one entirely — there is no partial replacement.
- Status fields (`status`, `superseded_by`) are written only by the CLI, never edited by hand; the CLI rewrites just those frontmatter lines and leaves everything else byte for byte.
- `## Why` is never invented: when the owner doesn't know, it says `Unknown (inferred from code).` with `inferred: true`, and lint keeps a warning on it.
- Only `accepted` ADRs are binding; `proposed` ones are shown as "not binding yet", `superseded` ones are not shown.

## Contracts
- Gives `changes`: ADR validation for drafts, the supersede operation, the list of ADRs with statuses.
- Gives `decisions-model`: accepted ADRs (tldr, rules, zones) as candidates for selection and for contradiction checks.
- Gives `docs`: the ADRs that apply to a zone (own and `global`) for `lore show`.
- Gives `cli` (lint): all ADR checks above — format, zones, supersede chain, duplicate ids, the `inferred` warning, code anchors.
- Takes from `docs`: the list of existing zone ids.
- Takes from `cli` (config): `anchors_ignore`, and the docs folder name (its files are never scanned for anchors).

## Entry points
- `internal/lore/adr.go` — new, accept, supersede, list
- `internal/lore/lint.go` — ADR checks and the code-anchor scan
- `internal/assets/files/templates/adr.md` — the ADR template

## Gotchas
- The anchor scan skips the docs folder, `AGENTS.md`, `CLAUDE.md`, gitignored files (when the project is in git), binary files and files over 1 MB. Outside git it walks the whole tree except `.git`, `node_modules`, `vendor`, `.direnv`.
- Anchors to a draft or a `proposed` ADR are valid; they only break when the ADR is superseded or gone.
- The id carries the creation date, not the acceptance date; two ADRs with the same slug on different days are different ADRs.
- `lore adr list` and `lore show` don't see drafts inside changes; `lore change status` shows them.
- A `tldr` or rule containing `: ` must be quoted in YAML, otherwise the frontmatter fails to parse (lint hints at this).
