---
id: changes
tldr: "Work in progress as lore/changes/<name>/: spec-first or code-first (from the git diff), tasks and the next task, change status, fact check against zones and accepted ADRs, OpenSpec deltas per zone and the two-phase archive gate that verifies the merge line by line."
---
# Changes

## Purpose
A change is the only way the "law" (zones, ADRs, product, architecture) is updated during normal work. It holds what is being changed right now — proposal, tasks, deltas to zones, ADR drafts — and ends at the archive gate, where the deltas are merged into the zones, the merge is verified and the change moves to the archive.

## Boundaries
- How the agent writes a proposal, implements tasks, reconciles deltas with the code and merges them → `workflows`. The CLI here only does the bookkeeping and the checks.
- ADR format, `lore adr accept` and superseding → `adrs`. This zone only decides *when* drafts of a change are moved and supersede.
- Zone doc format (frontmatter, `## Purpose`, requirement headings) → `docs`.
- How knowledge is picked for a task or a fact, thresholds and the model call → `decisions-model`.
- Resolving the change name on the command line and printing issues → `cli`.

## Requirements

### Requirement: Start a change
The system SHALL create `changes/<name>/` with a proposal, an empty `deltas/` and `adr/`, and a `.meta.yaml` recording the mode, the creation date and the base commit.

#### Scenario: Spec-first change
- **WHEN** the owner runs `lore change new <name>`
- **THEN** the change gets `proposal.md` and `tasks.md` from templates, mode `spec`, and the current `HEAD` as base when the project is in git

#### Scenario: Code-first change from the diff
- **WHEN** the owner runs `lore change new <name> --from-diff`
- **THEN** the change gets mode `code`, only `proposal.md` (no `tasks.md`), and as base the merge-base of `HEAD` with `main_branch` — or the commit given by `--base`

#### Scenario: Base can't be found
- **WHEN** `--from-diff` is used and the merge-base can't be computed, or `--base` is not a commit
- **THEN** nothing is created and the error asks for `--base <ref>`

#### Scenario: Bad or taken name
- **WHEN** the name is not lowercase kebab-case, or the change already exists
- **THEN** nothing is created

### Requirement: Tasks and progress
The system SHALL read tasks from the checkbox lines of `tasks.md` (`- [ ]` / `- [x]`); progress is done/total and the next task is the first unchecked one in file order.

#### Scenario: Next task
- **WHEN** `tasks.md` has `1.1` checked and `1.2` unchecked
- **THEN** the next task is `1.2`, with its line number

### Requirement: List active changes
The system SHALL list every active change (everything under `changes/` except `archive/` and hidden folders) with its mode, task progress and the zones its deltas touch.

### Requirement: Change status
`lore change status` SHALL print, for one change: the next task, a one-line summary of every delta operation, the accepted ADRs of the touched zones and global ones, model-picked knowledge for the next task (decisions as content, chosen zones by name), the ADR drafts of the change, and the full text of every delta. With `--next` it SHALL print only the next task and the model-picked knowledge for it.

#### Scenario: All tasks done
- **WHEN** no unchecked task is left (or a code-first change has no tasks)
- **THEN** status points to `lore change archive <name>`

#### Scenario: Spec change without tasks yet
- **WHEN** a spec-first change has no tasks in `tasks.md`
- **THEN** status says to finish the proposal first

#### Scenario: No decisions model
- **WHEN** there is no API key or the model call fails
- **THEN** status still prints everything else and says that no extra context was picked

#### Scenario: Between tasks
- **WHEN** the agent runs `lore change status --next` after finishing task `1.1`
- **THEN** it prints task `1.2` and the decisions and zone names picked for its text — no delta summary, no delta text, no list of the touched zones' ADRs, no drafts

#### Scenario: Next with all tasks done
- **WHEN** `--next` is used and no unchecked task is left
- **THEN** it points to `lore change archive <name>`, as the full status does

### Requirement: Fact check
`lore change check` SHALL take every bullet under `## What changes` of the proposal as a fact and report, per fact, the zones it touches and the accepted ADRs it may contradict, above the configured thresholds. Possible contradictions are questions for the owner, not verdicts.

#### Scenario: Zone touched without a delta
- **WHEN** a fact touches a zone the change has no delta for
- **THEN** the report lists that zone and asks to add a delta or explain why not

#### Scenario: Possible contradiction
- **WHEN** a fact may contradict an accepted ADR
- **THEN** the report shows the ADR's summary and asks the owner: new decision (supersede) or a bug in the code

#### Scenario: No facts
- **WHEN** `## What changes` has no bullets (HTML comments don't count)
- **THEN** the command fails and says where to write the facts

#### Scenario: No key
- **WHEN** the decisions model is not configured
- **THEN** the command fails (unlike status, it has nothing to show without the model)

### Requirement: Delta format
A delta SHALL be `deltas/<zone-id>.md` and either a whole new zone doc (it has frontmatter) or a set of OpenSpec operations on an existing zone: `## ADDED Requirements`, `## MODIFIED Requirements`, `## REMOVED Requirements` (each with `### Requirement: <name>`), `## RENAMED Requirements` (`FROM:` / `TO:` pairs) and `## MODIFIED Sections` (`### Section: <title>` for a `## <title>` of the zone). An ADDED or MODIFIED requirement and a MODIFIED section carry the whole new block: description and every scenario that still holds, changed or not.

#### Scenario: Unknown section
- **WHEN** a delta has a `##` heading other than the five operations
- **THEN** lint reports it as an error

#### Scenario: Unpaired rename
- **WHEN** a `FROM:` has no following `TO:`, or a `TO:` has no `FROM:`
- **THEN** lint reports it as an error

### Requirement: Pre-merge check
`lore change archive <name>` SHALL check, before the agent merges anything, that the change can be merged: in spec mode every task is done, and every delta fits the current zone.

#### Scenario: Delta doesn't fit
- **WHEN** a MODIFIED or REMOVED requirement, a RENAMED `FROM`, or a MODIFIED section does not exist in the zone, or a RENAMED `TO` already exists
- **THEN** it is an error and the change is not ready to merge

#### Scenario: MODIFIED drops a scenario
- **WHEN** the zone's requirement has a scenario the MODIFIED block doesn't have
- **THEN** it is a warning: the scenario will be deleted on merge — copy it into the delta if it still holds

#### Scenario: ADDED requirement already exists
- **WHEN** an ADDED requirement is already in the zone
- **THEN** it is a warning and the requirement is treated as MODIFIED

#### Scenario: New zone or missing zone
- **WHEN** a delta with frontmatter targets a zone that already exists, or a delta without frontmatter targets a zone that doesn't
- **THEN** it is an error

#### Scenario: Draft not accepted, no deltas
- **WHEN** an ADR draft of the change is not accepted yet, or the change has no deltas
- **THEN** it is a warning, not an error

### Requirement: Finish the archive
`lore change archive <name> --finish` SHALL run after the agent merged the deltas and the owner approved the diff of `lore/`. It verifies the merge, moves accepted ADR drafts into `adr/` (superseding what they replace), regenerates `INDEX.md`, records the archive date and `HEAD`, and moves the change to `changes/archive/<date>-<name>/`.

#### Scenario: Merged block equals the delta
- **WHEN** an ADDED or MODIFIED requirement or a MODIFIED section is checked
- **THEN** its block in the zone must equal the delta's block line for line, in order (comments, blank lines and whitespace ignored); the first missing, different or left-over line is reported and nothing is archived

#### Scenario: Old line left in the zone
- **WHEN** a MODIFIED scenario said "remind 3 days before" and the zone still also says "remind 1 day before"
- **THEN** the left-over line is reported as not in the delta

#### Scenario: Removals and renames verified
- **WHEN** a REMOVED requirement is still in the zone, a rename is not applied, or a new zone file does not exist
- **THEN** it is an error and nothing is archived

#### Scenario: Drafts not ready
- **WHEN** an ADR draft is not accepted, or an ADR with the same id already exists in `adr/`
- **THEN** it is an error and nothing is archived

#### Scenario: Project not clean
- **WHEN** the merge checks pass but `lore lint` finds any error in the project
- **THEN** nothing is archived and the lint errors are shown

## Invariants
- A change never moves to the archive while an ADR draft in it is not accepted.
- ADR drafts supersede existing ADRs only when the change is archived — never at `adr accept` (see `adrs`).
- The pre-merge check is read-only. `--finish` writes nothing until the merge checks pass; then it regenerates `INDEX.md`, and moves files only if the whole project is lint-clean.
- Spec-first and code-first changes end at the same gate with the same checks; the only difference is that code-first has no tasks to complete.
- Two active changes may touch the same requirement; this is a lint warning, never a lock.

## Contracts
- Takes from `docs`: zone docs, their requirements (by `### Requirement:` name) and `##` sections; `INDEX.md` regeneration.
- Takes from `adrs`: the list of ADRs and their statuses, ADR validation (drafts are linted like any ADR; their `zones` may name zones the change creates), superseding an ADR.
- Takes from `decisions-model`: knowledge selection for the next task (status) and per-fact zone and contradiction scores (check), with the configured thresholds; every call is logged.
- Takes from `cli`: the change name may be omitted when exactly one change is active.
- Gives `workflows`: `change new` / `status [--next]` / `check` / `archive` / `archive --finish` with their outputs and exit codes; `--finish` exits non-zero when nothing was archived.
- Gives `cli` (lint): change checks — name is kebab-case, `.meta.yaml` is valid with mode `spec` or `code`, delta file names are zone ids (not `global`), delta format problems, a new-zone delta has a matching `id`, a `tldr` and a non-empty `## Purpose`, and the warning about two changes touching the same requirement.

## Entry points
- `internal/lore/change.go` — new, list, status, archive check and finish
- `internal/lore/delta.go` — delta parsing, fit check, merge verification
- `internal/lore/ctx.go` — fact check (`CheckFacts`, `WriteFactReports`)
- `internal/assets/files/templates/proposal.md` — proposal template, `## What changes` is the fact list

## Gotchas
- `--finish` requires the *whole* project to be lint-clean, not only the change: an unrelated error elsewhere (another change, a stale index, a dead code anchor) blocks the archive.
- In a delta, a modified section is `### Section: <title>`; in the zone it is `## <title>`.
- A whole-zone delta (with frontmatter) is only checked for existence after the merge, not line by line.
- Tasks are any checkbox lines anywhere in `tasks.md`, including inside groups; order is file order, not the numbers.
- The fact check's model calls share one 2-minute timeout for the whole change; status uses 30 seconds for its model part.
