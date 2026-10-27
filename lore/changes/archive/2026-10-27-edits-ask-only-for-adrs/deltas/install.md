## MODIFIED Requirements

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

## MODIFIED Sections

### Section: Invariants
- Every install step is idempotent: running it again on an up-to-date project changes nothing.
- Owner edits are never overwritten without `--force`: not in managed files, not in `lore.yaml`, not outside the lore block of `AGENTS.md`, not in existing settings. The only things removed from settings are lore's own old entries: the `lore ctx --hook` hook and the `ask` rules for zones, `product.md` and `architecture.md`.
- Workflows exist once, in `lore/workflows/`; the agent sees them through symlinks, never through copies.
- The docs of the "law" are freely readable by the agent; an edit of an ADR asks the owner, edits of zones, `product.md` and `architecture.md` are approved at the gate (the diff of `lore/` before `--finish`); `changes/` is not guarded.
