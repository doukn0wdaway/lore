## ADDED Requirements

### Requirement: Knowledge on demand
The agent SHALL get project knowledge with `lore ctx "<text>"` when the owner asks it to look something up or when it decides it needs more, where the text is its own description of the work or question — never the owner's message quoted verbatim. `/lore-propose`, `/lore-explore` and `/lore-why` SHALL start from `lore ctx` instead of picking zones by hand and opening each with `lore show`.

#### Scenario: Proposing a feature
- **WHEN** the agent has understood what changes and why
- **THEN** it runs `lore ctx` with that description and works from the decisions and zones it prints

#### Scenario: Zone not printed
- **WHEN** the agent needs a zone that `lore ctx` did not print (not chosen, or over the budget)
- **THEN** it opens that zone with `lore show <zone>`

#### Scenario: No API key
- **WHEN** `lore ctx` fails because there is no key
- **THEN** the agent picks the zones from `INDEX.md` and opens them with `lore show <zone>`

## MODIFIED Requirements

### Requirement: Spec-first
A feature planned ahead SHALL go `/lore-propose` (proposal, deltas, tasks, ADR drafts, fact check, owner's approval — no code) → `/lore-apply` → `/lore-archive`.

#### Scenario: Implementing
- **WHEN** the agent applies a change
- **THEN** it runs `lore change status` at the start (and again whenever its context was lost), and `lore change status --next` before each following task; it implements that task only, verifies it, ticks it in `tasks.md`, and never edits zone docs or ADRs in `adr/`

#### Scenario: Scope changes
- **WHEN** a task turns out wrong or new work appears
- **THEN** the agent edits `tasks.md` and tells the owner; deltas may lag behind and are reconciled at the gate

### Requirement: Explore without building
`/lore-explore` SHALL ground the discussion in lore (`lore ctx` for the topic, `lore show` when a whole zone is needed, `product.md`, active changes), surface collisions with decisions, and never write code or lore docs; at the end it only offers the next step.

## MODIFIED Sections

### Section: Boundaries
- What the CLI commands check and do (statuses, delta checks, merge verification, archiving) → `changes`, `adrs`, `docs`.
- Copying workflows into a project, linking them as skills, rendering the `AGENTS.md` block → `install`.
- Picking knowledge for a text → `decisions-model`; workflows only say when to call `lore ctx` / `lore change status`.

### Section: Contracts
- Takes from `changes`: `lore change new / list / status / check / archive [--finish]` and their outputs.
- Takes from `adrs`: `lore adr new / accept / list`; from `docs`: `lore show`, `lore index`, `INDEX.md`; from `cli`: `lore lint`.
- Takes from `decisions-model`: `lore ctx`.
- Gives the agent: skills `/lore-<name>` (via `install`) and the protocol block in `AGENTS.md`.
