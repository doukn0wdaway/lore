---
id: decisions-model
tldr: "Picks knowledge for a text with a decisions model (OpenRouter, pplx-decider): stage 1 chooses zones, stage 2 their requirements, sections and accepted ADRs plus global ADRs; content vs pointer thresholds and a token budget; delivered by the prompt hook (once per session, never blocks), lore ctx and change status; lore's own API key; every call logged."
---
# Decisions model

## Purpose
Brings the right knowledge to the agent without the agent going after it. A small decisions model answers "does this relate to the work?" with a probability for every candidate — a zone, a requirement, a free section, an accepted ADR — and the zone turns those answers into a short `[lore]` block. Also owns the model client itself: the key, the questions, the call log.

## Boundaries
- What the fact check at the gate asks and how it reports → `changes` (it uses this zone's client, questions and thresholds).
- The rest of `change status` (next task, deltas, decisions of the touched zones) → `changes`.
- Zone and ADR content that becomes candidates → `docs`, `adrs`.
- Registering the hook in `.claude/settings.json` → `install`.
- Reading `lore.yaml` and its defaults → `cli`.

## Requirements

### Requirement: Lore's own API key
The system SHALL read the key from the environment variable named by `decisions_api_key_env` (default `LORE_OPENROUTER_API_KEY`), else from that name in the project's `lore/.env`, else in `~/.config/lore/.env` (`$XDG_CONFIG_HOME/lore/.env` when set). The application's own `.env` SHALL never be read.

#### Scenario: No key anywhere
- **WHEN** none of the three places has the key
- **THEN** commands that need the model fail with a message naming where to put it; status shows everything else; the hook stays silent

#### Scenario: Key written into lore.yaml
- **WHEN** `decisions_api_key_env` holds something that is not an env var name, or looks like a key (`sk…`, longer than 30 characters)
- **THEN** every command that uses the model and `lore lint` fail with a message to move and rotate the key, without echoing the value

### Requirement: Model call
The system SHALL send one state (the text) with a set of yes/no questions to the Decisions API endpoint and get back a probability per question, using the configured model and provider order with fallbacks disabled.

#### Scenario: Many questions
- **WHEN** there are more than 50 questions
- **THEN** they are split into batches of 50 sent in parallel; if any batch fails, that stage of the selection (or the fact check) fails as a whole

#### Scenario: Error response
- **WHEN** the API returns an error object or a non-2xx status
- **THEN** the call fails with the HTTP status and the message, truncated to 200 characters

### Requirement: Question texts
The question texts SHALL come from the file named in `prompts` of `lore.yaml`, else `lore/prompts/<name>.txt`, else the built-in text; `{kind}`, `{name}` and `{text}` are filled in. There are two questions: `relates` (does the work relate to this candidate) and `contradicts` (does this fact contradict this decision).

#### Scenario: Empty prompt file
- **WHEN** the configured prompt file is missing or empty
- **THEN** the next source is used

### Requirement: Two-stage selection
The system SHALL pick knowledge in two stages: stage 1 scores every zone by its `tldr`; stage 2 scores the candidates of the zones above the `zones` threshold — their requirements, their free sections (all except Purpose, Requirements and Entry points) and their accepted ADRs — plus every accepted global ADR.

#### Scenario: Stage 2 fails
- **WHEN** stage 2 times out or errors
- **THEN** the zones chosen in stage 1 are still delivered

#### Scenario: No zone above threshold
- **WHEN** no zone reaches the `zones` threshold
- **THEN** stage 2 still scores the global ADRs

### Requirement: Content, pointers and budget
A stage-2 candidate at or above `include_content` SHALL be injected as content while it fits the budget (`budget_tokens`, about 4 characters per token); one that doesn't fit, or scores between `include_pointer` and `include_content`, SHALL be mentioned by name only. Chosen zones are listed with a hint to open them with `lore show`.

#### Scenario: Budget exhausted
- **WHEN** a high-scoring requirement no longer fits the remaining budget
- **THEN** it is listed under "maybe relevant" instead of being injected

### Requirement: Prompt hook
On every prompt, the `UserPromptSubmit` hook SHALL inject the selection for the prompt text as additional context, and SHALL never block or fail the prompt.

#### Scenario: Skipped prompts
- **WHEN** the prompt has fewer than `min_words` words, starts with `/`, the hook is disabled, or no `lore.yaml` is found from the session's directory upward
- **THEN** nothing is injected and the model is not called

#### Scenario: Failure or timeout
- **WHEN** the key is missing, the model fails or `timeout_ms` passes
- **THEN** the prompt goes through without context and the error is logged

#### Scenario: Once per session
- **WHEN** an item's content or a zone was already injected earlier in the same Claude Code session
- **THEN** it is not injected again

#### Scenario: Pointer, then content
- **WHEN** an item was only mentioned as a pointer earlier in the session and a later prompt scores it at or above `include_content`
- **THEN** its content is injected; as a pointer it is not repeated

#### Scenario: Size cap
- **WHEN** the rendered block exceeds 9,500 characters
- **THEN** it is cut, because Claude Code caps additional context at 10,000

### Requirement: Knowledge for a text
`lore ctx "<text>"` SHALL print the same `[lore]` block for any text, without session dedup, or say that nothing relevant was found.

### Requirement: Call log
Every selection and fact check SHALL be appended to `lore/.local/log.jsonl`: time, source (`hook`, `ctx`, `status`, `check`), the state (truncated to 500 characters), all scores, what was injected, duration and error. Logging failures are ignored.

## Invariants
- The model sees only text: zone tldrs, requirement and section bodies, ADR tldrs and rules. Code is never sent to it.
- Nothing that talks to the model may block the agent's prompt.
- Only `accepted` ADRs are candidates; proposed and superseded ones never reach the context.
- The key never lives in `lore.yaml` and is never printed.
- Results are probabilities, not verdicts: contradictions are questions for the owner.

## Contracts
- Gives `changes`: `Select` for the next task (status), the client with the `relates` and `contradicts` questions and the thresholds (check), and the log.
- Gives the agent: the `[lore]` block in the hook's `additionalContext`, the output of `lore ctx`.
- Takes from `docs`: zones with tldr, requirement blocks, `##` sections.
- Takes from `adrs`: accepted ADRs with zones, tldr and rules.
- Takes from `cli`: `lore.yaml` values — model, endpoint, provider, key variable name, `hook.*`, `thresholds.*`, `prompts.*`.
- Takes from Claude Code (hook input JSON): `prompt` (or `prompt_text`), `session_id`, `cwd`.

## Entry points
- `internal/lore/decisions.go` — client, key lookup, question texts, log
- `internal/lore/select.go` — candidates, two-stage selection, rendering
- `internal/lore/ctx.go` — `lore ctx`, the hook, per-session dedup
- `internal/assets/files/prompts/relates.txt` — the built-in `relates` question

## Gotchas
- A session id that isn't `[A-Za-z0-9_-]{1,128}` disables dedup for that session rather than failing.
- Requirement and section text is cut to 700 characters for the model and to 1,800 for the agent.
- Timeouts differ by caller: hook `timeout_ms` (6 s by default), `lore ctx` 60 s, `change status` 30 s, `change check` 2 min.
- `lore/prompts/*.txt` are managed files: `lore upgrade` updates them unless they were edited locally.
- Thresholds in `lore.yaml` are starting values to be tuned on a pilot; the log exists to tune them.
