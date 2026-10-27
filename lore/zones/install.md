---
id: install
tldr: "lore init / upgrade / agents sync: idempotent scaffold of lore.yaml and lore/, managed workflows and prompts tracked by a hash manifest so local edits survive (unless --force), new lore.yaml keys added as text, the AGENTS.md block and @AGENTS.md in CLAUDE.md, .claude/settings.json permissions (and removal of the old prompt hook), skill symlinks, .gitignore, migration of old layouts."
---
# Install

## Purpose
Puts lore into a project and keeps it current: creates the config and the docs skeleton, copies the embedded workflows and prompts, and wires lore into the agent (protocol block, settings, skills). Re-running any of it is safe; owner edits are never overwritten silently.

## Boundaries
- What the workflows, prompts and templates say → `workflows`, `decisions-model`, `docs`, `adrs`, `changes` (the zone that owns each). This zone only copies and links them.
- What the keys of `lore.yaml` mean and their defaults → `cli`.
- Writing the project's knowledge (product, zones, ADRs) → `workflows` (bootstrap).

## Requirements

### Requirement: Init
`lore init` SHALL set up lore in the current directory and only add what is missing: `lore.yaml` from the template, `zones/`, `adr/`, `changes/archive/` (with `.gitkeep` while empty), `product.md` and `architecture.md` from templates, the managed files, `INDEX.md`, and the agent wiring.

#### Scenario: Run twice
- **WHEN** `lore init` runs in a project that already has lore
- **THEN** no existing file is overwritten and nothing changes unless a piece is missing

#### Scenario: Existing lore.yaml
- **WHEN** `lore.yaml` already exists
- **THEN** it is kept as is and its `root` decides where the docs folder goes

### Requirement: Managed files
Workflows (`lore/workflows/*`) and model prompts (`lore/prompts/*`) SHALL be managed: installed from the binary, recorded in `lore/.manifest.json` with the hash of the installed version, and updated on upgrade only if the local file was not edited since.

#### Scenario: Unedited file, new version
- **WHEN** the local file still matches the hash in the manifest and the binary has a different version
- **THEN** upgrade overwrites it and reports "updated"

#### Scenario: Edited file
- **WHEN** the local file was edited and the binary has a different version
- **THEN** upgrade keeps the local file, prints the unified diff to the new version, and keeps treating the file as edited

#### Scenario: Force
- **WHEN** upgrade runs with `--force`
- **THEN** every managed file is overwritten with the binary's version

#### Scenario: Deleted file
- **WHEN** a managed file is missing locally
- **THEN** it is created again

### Requirement: Upgrade
`lore upgrade` SHALL add new `lore.yaml` keys, update managed files and rewrite the agent wiring, so a project gets what a newer binary brings.

#### Scenario: New config keys
- **WHEN** the template has a top-level key or a nested key the project's `lore.yaml` lacks
- **THEN** it is inserted as text with its comment, after its neighbour from the template; every existing line stays byte for byte

#### Scenario: Merge would break YAML
- **WHEN** the merged `lore.yaml` would not parse
- **THEN** the file is left untouched and upgrade says which keys to add by hand

#### Scenario: Old hook block
- **WHEN** the project's `lore.yaml` still has a `hook:` block
- **THEN** it is kept byte for byte and upgrade prints a hint that the block is no longer used and can be deleted

### Requirement: Agent protocol block
The system SHALL keep a lore block in `AGENTS.md` between `<!-- lore:begin -->` and `<!-- lore:end -->`, rendered from the template with the docs folder, the features this binary has (changes) and the list of installed workflows; and SHALL make sure `CLAUDE.md` has the line `@AGENTS.md`. The block tells the agent to run `lore ctx "<the work or question, in its own words>"` when it needs project knowledge, never with the owner's message quoted verbatim, and `lore show <zone>` for a zone it did not print; and, when working on a change, to run `lore change status` at the start and `lore change status --next` before each following task.

#### Scenario: Block exists
- **WHEN** `AGENTS.md` already has the markers
- **THEN** only the text between them is replaced; the rest of the file is kept

#### Scenario: No block yet
- **WHEN** `AGENTS.md` exists without the markers, or doesn't exist
- **THEN** the block is appended, or the file is created with it

### Requirement: Claude Code settings
The system SHALL merge into `.claude/settings.json`: allow `Bash(lore:*)`; ask before `Edit` of `adr/**` in the docs folder. It SHALL register no hook, and SHALL remove a `UserPromptSubmit` hook whose command is `lore ctx --hook` left by an older version, and the `ask` rules for `Edit` of `zones/**`, `product.md` and `architecture.md` in the docs folder left by an older version. Existing settings and rules are otherwise kept; only missing ones are added.

#### Scenario: Old hook registered
- **WHEN** a `UserPromptSubmit` hook with the command `lore ctx --hook` exists
- **THEN** that hook is removed; a `UserPromptSubmit` entry left without hooks is removed too, and so is a `hooks` map left empty; every other hook stays

#### Scenario: Old ask rules
- **WHEN** `permissions.ask` has `Edit(/lore/zones/**)`, `Edit(/lore/product.md)` and `Edit(/lore/architecture.md)` next to the owner's own `ask` rules
- **THEN** those three are removed, `Edit(/lore/adr/**)` and the owner's rules stay

#### Scenario: Broken settings file
- **WHEN** `.claude/settings.json` is not valid JSON
- **THEN** the command fails and the file is not touched

### Requirement: Skill links
`lore agents sync` (also run by init and upgrade) SHALL link every workflow folder that has a `SKILL.md` into `.claude/skills/lore-<name>` with a relative symlink, fix links pointing elsewhere, and remove `lore-*` symlinks whose workflow is gone.

#### Scenario: Not a symlink
- **WHEN** `.claude/skills/lore-<name>` exists and is a real folder or file
- **THEN** it is left alone and reported as skipped

### Requirement: Gitignore
The system SHALL make sure `.gitignore` has `<root>/.env` and `<root>/.local/`.

### Requirement: Legacy layout
The system SHALL move an old top-level `.lore/` into the docs folder — `.env` (or `env`) to `lore/.env`, everything else into `lore/.local/` — remove `.lore/` when empty and drop its `.gitignore` line.

#### Scenario: Target exists
- **WHEN** the file already exists at the new place
- **THEN** the old one is kept where it is and reported

## Invariants
- Every install step is idempotent: running it again on an up-to-date project changes nothing.
- Owner edits are never overwritten without `--force`: not in managed files, not in `lore.yaml`, not outside the lore block of `AGENTS.md`, not in existing settings. The only things removed from settings are lore's own old entries: the `lore ctx --hook` hook and the `ask` rules for zones, `product.md` and `architecture.md`.
- Workflows exist once, in `lore/workflows/`; the agent sees them through symlinks, never through copies.
- The docs of the "law" are freely readable by the agent; an edit of an ADR asks the owner, edits of zones, `product.md` and `architecture.md` are approved at the gate (the diff of `lore/` before `--finish`); `changes/` is not guarded.

## Contracts
- Takes from the binary (embedded assets): `lore.yaml` template, `AGENTS.md` block template, doc templates, workflows, prompts, and the feature flags (changes).
- Takes from `docs`: `INDEX.md` generation.
- Gives `workflows`: their files in `lore/workflows/` and `/lore-<name>` skills in Claude Code.
- Gives `decisions-model`: the prompt files.
- Gives the agent: the protocol block through `CLAUDE.md` → `AGENTS.md`.

## Entry points
- `internal/lore/install.go` — init, upgrade, managed files, AGENTS.md, settings, skills, migration
- `internal/lore/configmerge.go` — adding new keys to `lore.yaml` as text
- `internal/assets/assets.go` — embedded files and the managed list
- `internal/assets/files/agents-block.md` — the protocol block template

## Gotchas
- `.claude/settings.json` is re-encoded when a rule is added or the old hook removed: keys get sorted and formatting normalised. Nothing is lost, but the diff can be larger than the changed lines.
- Upgrade doesn't create missing `product.md`, `architecture.md` or doc folders — only init does.
- Deleting a managed file doesn't opt out of it: the next init or upgrade puts it back. To change a workflow, edit it; upgrade will keep the edit.
- The `AGENTS.md` block is rewritten on every init and upgrade; edits inside the markers are lost. Put project rules outside them.
- Config merging only knows top-level keys and one level of nesting; a new key deeper than that is not added.
