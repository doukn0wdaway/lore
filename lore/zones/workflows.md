---
id: workflows
tldr: "The agent-side process shipped as Claude Code skills (bootstrap, propose, apply, document, archive, explore, adr, why, zone, product, audit) and the protocol block in AGENTS.md: the agent writes, the owner approves at gates; the why is never invented; ADRs only at a proposal or from the owner's answer at a fork; contradictions with ADRs go to the owner."
---
# Workflows

## Purpose
Defines how the agent works with lore: which steps it follows for each kind of task, where it must stop and ask the owner, and what it may write on its own. The CLI does the mechanics; the workflows tell the agent when to call it and what to write between the calls.

## Boundaries
- What the CLI commands check and do (statuses, delta checks, merge verification, archiving) → `changes`, `adrs`, `docs`.
- Copying workflows into a project, linking them as skills, rendering the `AGENTS.md` block → `install`.
- Picking knowledge for a text → `decisions-model`; workflows only say when to call `lore ctx` / `lore change status`.

## Requirements

### Requirement: Agent writes, owner approves
The agent SHALL write every lore document itself and SHALL get the owner's explicit yes at the gates: the proposal with its deltas and ADR drafts, every ADR before `lore adr accept`, the diff of `lore/` before `lore change archive --finish`, and each step of bootstrap.

#### Scenario: Accepting an ADR
- **WHEN** an ADR or a draft is written
- **THEN** the agent shows its `tldr`, `rules` and `Why` and runs `lore adr accept` only after an explicit yes

#### Scenario: Archiving
- **WHEN** the deltas are merged into the zones
- **THEN** the agent shows the owner the diff of `lore/` (summarised if long, nothing hidden) and runs `--finish` only after a yes

### Requirement: The why is never invented
The agent SHALL take the reason behind a decision only from the owner. When the owner doesn't know, the ADR says `Unknown (inferred from code).` with `inferred: true`; when nothing is recorded, the agent says "No recorded decision covers this" and may offer hints labelled as hints.

#### Scenario: Asked why
- **WHEN** the owner or the agent asks why something is built a certain way
- **THEN** the answer cites the ADR id and its `## Why`, or says plainly that no decision is recorded — never a guess presented as the reason

#### Scenario: Why not X
- **WHEN** X is among an ADR's rejected alternatives
- **THEN** the agent gives the recorded reason and does not propose X again unless a `## Revisit if` condition holds now; when the ADR has no rejected alternatives, X may be suggested as a question to the owner

### Requirement: When an ADR is created
An ADR SHALL be created only when the decision has a non-obvious "why" and a future agent might want to "fix" it, and only at a proposal, from the owner's answer at a fork, or in the bootstrap interview — never on the agent's own initiative during implementation. Obvious rules go to a zone's `## Invariants`, local oddities to a code comment.

#### Scenario: Fork during implementation
- **WHEN** two reasonable ways exist and neither the deltas nor the ADRs decide
- **THEN** the agent stops and asks; the answer is recorded under `## Owner notes` of the proposal, and becomes an ADR draft only if it should outlive the change and the owner agrees

### Requirement: Contradiction with a decision
When work, a fact of a change, an idea or the code contradicts an accepted ADR, the agent SHALL stop and ask the owner whether it is a new decision (an ADR that supersedes the old one entirely) or a mistake (keep the decision, change the plan or fix the code). The agent never decides this itself and never works around an ADR.

#### Scenario: Flagged by the fact check
- **WHEN** `lore change check` reports a possible contradiction
- **THEN** the agent reads the ADR, confirms whether the conflict is real, and asks the owner about each real one

#### Scenario: Bug
- **WHEN** the owner says the contradiction is a bug
- **THEN** the change is not archived until the code is fixed

### Requirement: Spec-first
A feature planned ahead SHALL go `/lore-propose` (proposal, deltas, tasks, ADR drafts, fact check, owner's approval — no code) → `/lore-apply` → `/lore-archive`.

#### Scenario: Implementing
- **WHEN** the agent applies a change
- **THEN** it runs `lore change status` at the start (and again whenever its context was lost), and `lore change status --next` before each following task; it implements that task only, verifies it, ticks it in `tasks.md`, and never edits zone docs or ADRs in `adr/`

#### Scenario: Scope changes
- **WHEN** a task turns out wrong or new work appears
- **THEN** the agent edits `tasks.md` and tells the owner; deltas may lag behind and are reconciled at the gate

### Requirement: Code-first
Work done without a change SHALL be documented with `/lore-document`: a change from the git diff, one behavioural fact per bullet in `## What changes`, the fact check, decision candidates asked one at a time, deltas describing what the code now does, then `/lore-archive`.

#### Scenario: Work on the main branch
- **WHEN** there is no merge-base to start from
- **THEN** the agent asks the owner where the work started and passes `--base`

### Requirement: The archive gate
`/lore-archive` SHALL be the only way the law changes during normal work: pre-check, reconcile deltas with the real code (what was built, not what was planned), check facts against decisions, settle every ADR draft, merge deltas into zones (MODIFIED replaces the whole block) and update `product.md` and `architecture.md`, repoint code anchors of superseded ADRs, `lore index` and `lore lint`, the owner's yes on the diff, `--finish`.

#### Scenario: Unexplained deviation
- **WHEN** the code differs from the deltas and `## Owner notes` doesn't explain it
- **THEN** the agent asks the owner whether it is intended

#### Scenario: Planned scenario with a future zone goes live
- **WHEN** a `[planned]` scenario naming `receipts?` becomes `[live]` at archive
- **THEN** its `zones:` line is left without `?`: the zone the change created, or the zones the scenario really passes through

### Requirement: Bootstrap
`/lore-bootstrap` SHALL document an existing project in six gated steps — product, zone map, architecture, zones one at a time, decision interview, code anchors — keeping progress in `lore/.local/bootstrap.json` so it resumes in a later session.

#### Scenario: Resume
- **WHEN** bootstrap is invoked and the progress file exists
- **THEN** it continues from the recorded step instead of starting over

#### Scenario: Existing docs
- **WHEN** the project has old specs, READMEs or ADRs
- **THEN** they are treated as hints and verified against the code; the code and the owner win

### Requirement: Maintaining zones and product
Zones and `product.md` SHALL change outside a change only through `/lore-zone` and `/lore-product` at the owner's request, showing the change and getting a yes first. Requirement names stay stable; a split or merge moves requirements under their names and fixes `zones:` in ADRs and `product.md`.

#### Scenario: Scenario no longer wanted
- **WHEN** a scenario is abandoned
- **THEN** its line is deleted; if the reason is non-obvious and it might be proposed again, the owner is asked to record it as an ADR

#### Scenario: Update reverses a decision
- **WHEN** a zone update would contradict an ADR
- **THEN** the agent stops and goes through `/lore-adr` first

#### Scenario: Wanted scenario needs a zone that doesn't exist
- **WHEN** `/lore-product` adds a `[planned]` scenario that goes through a zone not in the index
- **THEN** the zone is written with `?` (`receipts?`) and the owner is told a new zone will be needed; no zone is created — its boundaries are defined later, in the proposal that builds it

#### Scenario: Future zone is created
- **WHEN** `/lore-zone` creates a zone that `product.md` names as `receipts?`
- **THEN** the `?` is dropped in every scenario that names it

### Requirement: Explore without building
`/lore-explore` SHALL ground the discussion in lore (`lore ctx` for the topic, `lore show` when a whole zone is needed, `product.md`, active changes), surface collisions with decisions, and never write code or lore docs; at the end it only offers the next step.

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

### Requirement: Audit reports only
`/lore-audit` SHALL compare docs with the code per zone and report each claim as stale, missing or unverifiable (counting the ok ones), with file paths; an ADR rule the code violates is reported as "bug or new decision?". Nothing is fixed without the owner's word.

## Invariants
- All lore docs are written in English, whatever language the owner speaks.
- The law (zones, ADRs, product, architecture) changes only at the archive gate or in a workflow the owner invoked (bootstrap, zone, adr, product) — never on the agent's initiative.
- Questions to the owner go one at a time.
- `lore adr accept` is never run without the owner's explicit yes; ADR statuses are never edited by hand.
- A superseding ADR restates every rule of the old one that still holds — there is no partial replacement.
- The `AGENTS.md` protocol stays short (about 15 lines); everything else lives in the workflows.

## Contracts
- Takes from `changes`: `lore change new / list / status / check / archive [--finish]` and their outputs.
- Takes from `adrs`: `lore adr new / accept / list`; from `docs`: `lore show`, `lore index`, `INDEX.md`; from `cli`: `lore lint`.
- Takes from `decisions-model`: `lore ctx`.
- Gives the agent: skills `/lore-<name>` (via `install`) and the protocol block in `AGENTS.md`.

## Entry points
- `internal/assets/files/workflows/` — all workflow sources (one `SKILL.md` per folder)
- `internal/assets/files/workflows/archive/SKILL.md` — the gate
- `internal/assets/files/workflows/bootstrap/SKILL.md` — bootstrap of an existing project
- `internal/assets/files/agents-block.md` — the protocol block for `AGENTS.md`

## Gotchas
- In this repository the source of the workflows is `internal/assets/files/workflows/`; `lore/workflows/` holds the installed copies. Edit the source, rebuild, then `lore upgrade` to update the copies — an edit made only in `lore/workflows/` is kept by upgrade as a local edit and diverges from the binary.
- A workflow's frontmatter `name` is `lore-<folder>`, matching its skill link `.claude/skills/lore-<folder>`; nothing checks this, so keep them equal when adding a workflow.
- `/lore-explore` mentions `lore/ideas.md` "if the project has one"; nothing in lore creates or checks it.
