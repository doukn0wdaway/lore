---
name: lore-propose
description: Start a feature spec-first with lore — clarify what and why with the owner, then write the change (proposal, deltas to zones, tasks, ADR drafts), check it against existing decisions and get the owner's approval before any code. Use when the owner wants to plan, spec or propose a feature or change before implementing it.
---

# lore propose — spec-first

Nothing is coded in this workflow. It ends with an approved change ready for `/lore-apply`.

## 1. Understand

Ask until you can state in two lines what changes for the user and why. Read `lore/product.md` (is it a `[planned]` scenario?). Then `lore ctx "<what changes and why, in your own words>"` — the accepted ADRs that apply and the related zones, whole; design within those ADRs. A zone it didn't print, or no API key (then pick the zones from `lore/INDEX.md`): `lore show <zone>` — it also prints the zone's active decisions.

## 2. Create the change

```sh
lore change new <kebab-name>
```

It creates `lore/changes/<name>/` with `proposal.md`, `tasks.md`, `deltas/`, `adr/`.

## 3. Write it (English)

**`proposal.md`** — `Why`, `What changes` (one behavioural fact per bullet), `Zones`, `Scenarios` (how `product.md` changes: planned → live, changed, removed — with why), `ADR drafts`.

**`deltas/<zone>.md`** — one file per touched zone, OpenSpec format:

```markdown
## ADDED Requirements
### Requirement: Voice messages are transcribed
The system SHALL transcribe incoming voice messages in the background.
#### Scenario: Voice message received
- **WHEN** a user sends a voice message
- **THEN** the file is stored and a transcription job is enqueued

## MODIFIED Requirements
### Requirement: File upload size limit
The system SHALL reject uploads over the size limit.
#### Scenario: File too big
- **WHEN** a file is larger than 20 MB
- **THEN** it is rejected
#### Scenario: Voice message too long
- **WHEN** a voice message is longer than 5 minutes
- **THEN** it is rejected with a hint to split it

## REMOVED Requirements
### Requirement: Text-only intake

## RENAMED Requirements
- FROM: `### Requirement: Old name`
- TO: `### Requirement: New name`

## MODIFIED Sections
### Section: Components
<full new text of the zone's free-form `## Components` section>
```

- `MODIFIED` carries the **whole** requirement: its description and every scenario that still holds, changed or not. It replaces the zone's block on merge, so a scenario left out is deleted (`lore change archive` warns about each one).
- Requirement names must match the zone exactly (`lore change archive` checks).
- A **new zone** = `deltas/<new-id>.md` holding the whole zone doc (frontmatter `id`, `tldr`, `## Purpose`, …) — see `/lore-zone` for the format and the zone rules.
- Describe behaviour, not implementation.

**`tasks.md`** — OpenSpec checkboxes in numbered groups, in execution order, each small enough to do and check on its own:

```markdown
## 1. Storage
- [ ] 1.1 voice_messages table and migration
- [ ] 1.2 Background transcription
```

**ADR drafts** — only if the feature needs a decision that passes the ADR test (non-obvious "why" + a future agent might "fix" it). Ask the owner *why*; never invent it. Then:

```sh
lore adr new <slug> --zones a,b --change <name> [--supersedes <old-id>]
```

and fill it as in `/lore-adr`. A draft that replaces an accepted ADR restates every rule of the old one that still holds.

## 4. Check against existing decisions

`lore change check [name]` lists, per fact of `What changes`, the zones it touches and the accepted ADRs it may contradict (decisions model; without a key do it by hand with `lore show`). Confirm each flagged one by reading it. For every fact and every draft: does it contradict an accepted ADR of the touched zones (or a global one)? For each contradiction ask the owner: **new decision** (→ draft that supersedes it) or **keep the old one** (→ change the plan). Never decide yourself.

`lore lint` — fix errors in the change.

## 5. Owner's approval

Show the proposal, the deltas and the drafts. On the owner's yes:

- `lore adr accept <id>` for each draft the owner approved (it stays in the change until archive);
- tell the owner the change is ready: `/lore-apply`.
