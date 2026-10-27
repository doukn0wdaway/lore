## ADDED Requirements

### Requirement: Old hook keys
`lore lint` SHALL warn when `lore.yaml` has a `hook:` block: its keys are no longer read, and the budget is `ctx.budget_tokens`.

#### Scenario: Upgraded project with the old block
- **WHEN** `lore.yaml` has `hook:` with `budget_tokens: 4000` and no `ctx:` block
- **THEN** lint warns that `hook:` is unused and that the budget belongs in `ctx.budget_tokens`; the default 8000 is used

## MODIFIED Requirements

### Requirement: Find the project
Every command except `init`, `help` and `version` SHALL find the project by walking up from the current directory to the nearest folder containing `lore.yaml`; that folder is the project root, and the docs folder is `root` from `lore.yaml` under it.

#### Scenario: No project
- **WHEN** no `lore.yaml` exists here or in any parent
- **THEN** the command fails and suggests `lore init`

### Requirement: Config defaults
`lore.yaml` SHALL need no keys at all: every missing or zero value gets its default — `root: lore`, `main_branch: main`, the decisions model, endpoint and key variable, `ctx.budget_tokens: 8000`, thresholds `0.3 / 0.7 / 0.4 / 0.5`.

#### Scenario: Broken YAML
- **WHEN** `lore.yaml` doesn't parse
- **THEN** every command that needs the project fails with the YAML error

## MODIFIED Sections

### Section: Boundaries
- What each command does → the zone of its object: `install` (init, upgrade, agents sync), `docs` (index, show), `adrs` (adr …), `changes` (change …), `decisions-model` (ctx).
- What lint checks → the zone of the checked thing (`docs`, `adrs`, `changes`, `decisions-model` for the key guard). This zone only runs them and reports, and checks `lore.yaml` itself.
- Adding new keys to an existing `lore.yaml` → `install`.

### Section: Invariants
- The CLI never asks interactive questions; everything comes from arguments, files and the environment, so the agent can run it.
- Read-only commands (`lint`, `show`, `adr list`, `change list`, `change status`, `change check`, `ctx`) never change the docs; only the decisions log in `lore/.local/` is written.
- Every message about a file uses its path relative to the project root.

### Section: Gotchas
- A zero value in `lore.yaml` means "use the default": a threshold of `0` or `budget_tokens: 0` can't be set.
- `-h` or `--help` anywhere prints usage, even as a value: `lore ctx "--help"` shows help instead of picking knowledge.
- Single-dash flags work too (`-zones a,b`), since leading dashes are stripped.
- `lore change archive` without `--finish` also exits 1 on errors, so the gate can be scripted.
