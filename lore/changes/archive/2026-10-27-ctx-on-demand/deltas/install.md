<!-- At merge, also update the zone tldr: ".claude/settings.json permissions and prompt hook" → ".claude/settings.json permissions (and removal of the old prompt hook)". -->

## MODIFIED Requirements

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
The system SHALL merge into `.claude/settings.json`: allow `Bash(lore:*)`; ask before `Edit` of `zones/**`, `adr/**`, `product.md` and `architecture.md` in the docs folder. It SHALL register no hook, and SHALL remove a `UserPromptSubmit` hook whose command is `lore ctx --hook` left by an older version. Existing settings and rules are otherwise kept; only missing ones are added.

#### Scenario: Old hook registered
- **WHEN** a `UserPromptSubmit` hook with the command `lore ctx --hook` exists
- **THEN** that hook is removed; a `UserPromptSubmit` entry left without hooks is removed too, and so is a `hooks` map left empty; every other hook stays

#### Scenario: Broken settings file
- **WHEN** `.claude/settings.json` is not valid JSON
- **THEN** the command fails and the file is not touched

## MODIFIED Sections

### Section: Purpose
Puts lore into a project and keeps it current: creates the config and the docs skeleton, copies the embedded workflows and prompts, and wires lore into the agent (protocol block, settings, skills). Re-running any of it is safe; owner edits are never overwritten silently.

### Section: Boundaries
- What the workflows, prompts and templates say → `workflows`, `decisions-model`, `docs`, `adrs`, `changes` (the zone that owns each). This zone only copies and links them.
- What the keys of `lore.yaml` mean and their defaults → `cli`.
- Writing the project's knowledge (product, zones, ADRs) → `workflows` (bootstrap).

### Section: Invariants
- Every install step is idempotent: running it again on an up-to-date project changes nothing.
- Owner edits are never overwritten without `--force`: not in managed files, not in `lore.yaml`, not outside the lore block of `AGENTS.md`, not in existing settings. The one thing removed from settings is lore's own old `lore ctx --hook` entry.
- Workflows exist once, in `lore/workflows/`; the agent sees them through symlinks, never through copies.
- The docs of the "law" (zones, ADRs, product, architecture) are freely readable by the agent but every edit asks the owner; `changes/` is not guarded.

### Section: Contracts
- Takes from the binary (embedded assets): `lore.yaml` template, `AGENTS.md` block template, doc templates, workflows, prompts, and the feature flags (changes).
- Takes from `docs`: `INDEX.md` generation.
- Gives `workflows`: their files in `lore/workflows/` and `/lore-<name>` skills in Claude Code.
- Gives `decisions-model`: the prompt files.
- Gives the agent: the protocol block through `CLAUDE.md` → `AGENTS.md`.

### Section: Gotchas
- `.claude/settings.json` is re-encoded when a rule is added or the old hook removed: keys get sorted and formatting normalised. Nothing is lost, but the diff can be larger than the changed lines.
- Upgrade doesn't create missing `product.md`, `architecture.md` or doc folders — only init does.
- Deleting a managed file doesn't opt out of it: the next init or upgrade puts it back. To change a workflow, edit it; upgrade will keep the edit.
- The `AGENTS.md` block is rewritten on every init and upgrade; edits inside the markers are lost. Put project rules outside them.
- Config merging only knows top-level keys and one level of nesting; a new key deeper than that is not added.
