---
id: decisions-model
tldr: "Picks knowledge for a text with a decisions model (OpenRouter, pplx-decider): every zone (tldr with its invariants, boundaries and contracts) and every accepted ADR scored in one selection; decisions as content, chosen zones whole within a token budget, the rest as pointers; delivered on demand by lore ctx and change status; lore's own API key; every call logged."
---
# Decisions model

## Purpose
Brings the right knowledge to the agent when it asks for it. A small decisions model answers "does this relate to the work?" with a probability for every zone and every accepted ADR, and the zone turns those answers into a `[lore]` block: the decisions that apply, then the related zones whole. Also owns the model client itself: the key, the questions, the call log.

## Boundaries
- What the fact check at the gate asks and how it reports → `changes` (it uses this zone's client, questions and thresholds).
- The rest of `change status` (next task, deltas, decisions of the touched zones) → `changes`.
- Zone and ADR content that becomes candidates → `docs`, `adrs`.
- When the agent calls `lore ctx` → `workflows` and the `AGENTS.md` protocol block (`install`).
- Reading `lore.yaml` and its defaults → `cli`.

## Requirements

### Requirement: Lore's own API key
The system SHALL read the key from the environment variable named by `decisions_api_key_env` (default `LORE_OPENROUTER_API_KEY`), else from that name in the project's `lore/.env`, else in `~/.config/lore/.env` (`$XDG_CONFIG_HOME/lore/.env` when set). The application's own `.env` SHALL never be read.

#### Scenario: No key anywhere
- **WHEN** none of the three places has the key
- **THEN** commands that need the model fail with a message naming where to put it; status shows everything else

#### Scenario: Key written into lore.yaml
- **WHEN** `decisions_api_key_env` holds something that is not an env var name, or looks like a key (`sk…`, longer than 30 characters)
- **THEN** every command that uses the model and `lore lint` fail with a message to move and rotate the key, without echoing the value

### Requirement: Model call
The system SHALL send one state (the text) with a set of yes/no questions to the Decisions API endpoint and get back a probability per question, using the configured model and provider order with fallbacks disabled.

#### Scenario: Many questions
- **WHEN** there are more than 50 questions
- **THEN** they are split into batches of 50 sent in parallel; if any batch fails, the selection (or the fact check) fails as a whole

#### Scenario: Error response
- **WHEN** the API returns an error object or a non-2xx status
- **THEN** the call fails with the HTTP status and the message, truncated to 200 characters

### Requirement: Question texts
The question texts SHALL come from the file named in `prompts` of `lore.yaml`, else `lore/prompts/<name>.txt`, else the built-in text; `{kind}`, `{name}` and `{text}` are filled in. There are two questions: `relates` (does the work relate to this candidate) and `contradicts` (does this fact contradict this decision).

#### Scenario: Empty prompt file
- **WHEN** the configured prompt file is missing or empty
- **THEN** the next source is used

### Requirement: Selection by zones and decisions
The system SHALL pick knowledge by scoring, in one selection, every zone and every accepted ADR (zone and global) with the `relates` question. A zone is asked about with its tldr followed by its `## Invariants`, `## Boundaries` and `## Contracts`; an ADR with its tldr and rules. Requirements and other sections are not scored on their own.

#### Scenario: Zone related through an invariant
- **WHEN** the text relates to a zone only through one of its invariants, not through its tldr
- **THEN** the zone is still scored on that invariant and picked if it scores high enough

#### Scenario: ADR of several zones
- **WHEN** an accepted ADR names several zones
- **THEN** it is asked about once

#### Scenario: Selection fails
- **WHEN** a model call of the selection times out or errors
- **THEN** the selection fails as a whole; `lore ctx` reports the error, `change status` shows everything else

### Requirement: Content, pointers and budget
The selection SHALL be printed as two lists. Decisions: every ADR at or above `include_content` as content (its tldr and rules), with no budget; an ADR between `include_pointer` and `include_content` is mentioned by name only under "maybe relevant". Zones: every zone at or above the `zones` threshold, highest score first, printed whole while it fits `ctx.budget_tokens` (about 4 characters per token); a zone that doesn't fit is listed by name with a hint to open it with `lore show`. `change status` lists the chosen zones by name only.

#### Scenario: Budget exhausted
- **WHEN** a chosen zone no longer fits the remaining budget
- **THEN** it is listed by name with the `lore show` hint instead of being printed

#### Scenario: Many decisions
- **WHEN** many ADRs score at or above `include_content` and the zone budget is already used up
- **THEN** every one of those ADRs is still shown as content

### Requirement: Knowledge for a text
`lore ctx "<text>"` SHALL print the selection for any text, or say that nothing relevant was found. The text is the agent's own description of the work or question; the model judges which of the project's knowledge relates to it.

### Requirement: Call log
Every selection and fact check SHALL be appended to `lore/.local/log.jsonl`: time, source (`ctx`, `status`, `check`), the state (truncated to 500 characters), all scores, what was shown, duration and error. Logging failures are ignored.

## Invariants
- The model sees only text: zone tldrs with their invariants, boundaries and contracts, ADR tldrs and rules, facts. Code is never sent to it.
- Nothing is put into the agent's context unless the agent runs a command.
- Only `accepted` ADRs are candidates; proposed and superseded ones never reach the context.
- The key never lives in `lore.yaml` and is never printed.
- Results are probabilities, not verdicts: contradictions are questions for the owner.

## Contracts
- Gives `changes`: `Select` for the next task (status), the client with the `relates` and `contradicts` questions and the thresholds (check), and the log.
- Gives the agent: the output of `lore ctx`.
- Takes from `docs`: zones with tldr, their `##` sections and whole text.
- Takes from `adrs`: accepted ADRs with zones, tldr and rules.
- Takes from `cli`: `lore.yaml` values — model, endpoint, provider, key variable name, `ctx.budget_tokens`, `thresholds.*`, `prompts.*`.

## Entry points
- `internal/lore/decisions.go` — client, key lookup, question texts, log
- `internal/lore/select.go` — candidates, selection, rendering
- `internal/lore/ctx.go` — `lore ctx`
- `internal/assets/files/prompts/relates.txt` — the built-in `relates` question

## Gotchas
- A zone's text for the model (tldr, Invariants, Boundaries, Contracts) is cut to 3,000 characters; an ADR's to 700.
- Timeouts differ by caller: `lore ctx` 60 s, `change status` 30 s, `change check` 2 min.
- A zone whose invariants, boundaries and contracts are empty is matched on its tldr alone.
- `lore/prompts/*.txt` are managed files: `lore upgrade` updates them unless they were edited locally.
- Thresholds in `lore.yaml` are starting values to be tuned on a pilot; the log exists to tune them.
- The `zones` threshold decides both which zones `lore ctx` prints and which zones the fact check says a fact touches.
