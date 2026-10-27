---
id: cli
tldr: "The lore binary as a command line: commands and subcommands, flags before or after arguments, -h anywhere prints usage, the project root is the nearest folder with lore.yaml (like git), lore.yaml defaults, and the lint report (errors before warnings, --json, non-zero exit on errors)."
---
# CLI

## Purpose
How the `lore` binary meets the user: finding the project, reading `lore.yaml`, parsing commands and flags, and reporting problems with the right exit code. It owns no domain rules — every command hands the work to the zone that owns the action.

## Boundaries
- What each command does → the zone of its object: `install` (init, upgrade, agents sync), `docs` (index, show), `adrs` (adr …), `changes` (change …), `decisions-model` (ctx, the hook).
- What lint checks → the zone of the checked thing (`docs`, `adrs`, `changes`, `decisions-model` for the key guard). This zone only runs them and reports.
- Adding new keys to an existing `lore.yaml` → `install`.

## Requirements

### Requirement: Find the project
Every command except `init`, `help`, `version` and the prompt hook SHALL find the project by walking up from the current directory to the nearest folder containing `lore.yaml`; that folder is the project root, and the docs folder is `root` from `lore.yaml` under it.

#### Scenario: No project
- **WHEN** no `lore.yaml` exists here or in any parent
- **THEN** the command fails and suggests `lore init`

#### Scenario: Hook outside the project
- **WHEN** the prompt hook runs
- **THEN** it looks for `lore.yaml` from the session's working directory in the hook input, not from the process's current directory

### Requirement: Config defaults
`lore.yaml` SHALL need no keys at all: every missing or zero value gets its default — `root: lore`, `main_branch: main`, the decisions model, endpoint and key variable, `hook.min_words: 4`, `hook.timeout_ms: 6000`, `hook.budget_tokens: 2500`, thresholds `0.3 / 0.7 / 0.4 / 0.5`.

#### Scenario: Broken YAML
- **WHEN** `lore.yaml` doesn't parse
- **THEN** every command that needs the project fails with the YAML error

### Requirement: Commands and help
The binary SHALL print usage when run without arguments, with `help`, or with `-h` / `-help` / `--help` anywhere on the line, fail with a pointer to `lore help` on an unknown command, and fail naming an unknown subcommand. `lore version` prints the version.

### Requirement: Flags
Flags SHALL be accepted before or after positional arguments, as `--name value` or `--name=value` (one or two dashes); each command accepts only its own flags.

#### Scenario: Unknown flag
- **WHEN** a command gets a flag it doesn't know
- **THEN** it fails naming the flag

#### Scenario: Missing value
- **WHEN** a value flag is last on the line without a value
- **THEN** the command fails saying the flag needs a value

### Requirement: Change name may be omitted
Every `lore change` subcommand that works on one change SHALL accept no name when exactly one change is active, and otherwise fail listing the active changes.

### Requirement: Errors and exit codes
A failing command SHALL print `lore: <message>` to stderr and exit 1. Commands that report issues (`lint`, `change archive`, `change archive --finish`) print the issues to stdout and exit 1 when there is at least one error; warnings alone don't fail.

### Requirement: Lint report
`lore lint` SHALL run every check of the project and print one line per issue — `<severity> <file>[:<line>]: <message>` — errors first, then by file and line, followed by `N error(s), M warning(s)`; with `--json` it prints the issues as a JSON array (empty array when clean).

#### Scenario: Clean project
- **WHEN** there are no issues
- **THEN** it prints `0 error(s), 0 warning(s)` and exits 0

#### Scenario: Only warnings
- **WHEN** lint finds warnings and no errors
- **THEN** it exits 0

## Invariants
- The CLI never asks interactive questions; everything comes from arguments, files and the environment, so the agent can run it.
- Read-only commands (`lint`, `show`, `adr list`, `change list`, `change status`, `change check`, `ctx`) never change the docs; only the decisions log and session files in `lore/.local/` are written.
- Every message about a file uses its path relative to the project root.

## Contracts
- Gives every zone: the project (root, docs folder, `lore.yaml` values) and the resolved change.
- Takes from `docs`, `adrs`, `changes`, `decisions-model`: their lint checks; `lint` combines them.
- Gives `workflows` and the owner: the commands, their output and exit codes.

## Entry points
- `cmd/lore/main.go` — command routing, flags, usage, exit codes
- `internal/lore/project.go` — finding the project, `lore.yaml` and defaults
- `internal/lore/lint.go` — the lint run and the issue format

## Gotchas
- A zero value in `lore.yaml` means "use the default": a threshold of `0` or `min_words: 0` can't be set.
- `hook.enabled` has no default: a `lore.yaml` without it has the prompt hook switched off. `lore init` writes `enabled: true`.
- `-h` or `--help` anywhere prints usage, even as a value: `lore ctx "--help"` shows help instead of picking knowledge.
- Single-dash flags work too (`-zones a,b`), since leading dashes are stripped.
- `lore change archive` without `--finish` also exits 1 on errors, so the gate can be scripted.
