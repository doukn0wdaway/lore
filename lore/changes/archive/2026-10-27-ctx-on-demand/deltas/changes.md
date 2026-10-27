## MODIFIED Requirements

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

## MODIFIED Sections

### Section: Contracts
- Takes from `docs`: zone docs, their requirements (by `### Requirement:` name) and `##` sections; `INDEX.md` regeneration.
- Takes from `adrs`: the list of ADRs and their statuses, ADR validation (drafts are linted like any ADR; their `zones` may name zones the change creates), superseding an ADR.
- Takes from `decisions-model`: knowledge selection for the next task (status) and per-fact zone and contradiction scores (check), with the configured thresholds; every call is logged.
- Takes from `cli`: the change name may be omitted when exactly one change is active.
- Gives `workflows`: `change new` / `status [--next]` / `check` / `archive` / `archive --finish` with their outputs and exit codes; `--finish` exits non-zero when nothing was archived.
- Gives `cli` (lint): change checks — name is kebab-case, `.meta.yaml` is valid with mode `spec` or `code`, delta file names are zone ids (not `global`), delta format problems, a new-zone delta has a matching `id`, a `tldr` and a non-empty `## Purpose`, and the warning about two changes touching the same requirement.
