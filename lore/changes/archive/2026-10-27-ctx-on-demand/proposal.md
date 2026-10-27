# Ctx on demand

## Why
The prompt hook fires only on the owner's own messages, and skips every workflow call (`/…`) and every short message, so it is absent from most of the real work, and when it does fire it often brings noise. Knowledge is needed only when the owner asks the agent to look something up, or when the agent decides it needs more — so the agent should call `lore ctx` itself. Inside workflows the agent loads whole zones with `lore show`, and in `lore ctx` the relevant ADRs lose the shared budget to zone sections and arrive as names only. Stage 1 (zones by tldr) can drop the ADRs of a zone that scored low, and picking pieces of zones made relevant ADRs lose to zone sections; the benchmark measured recall on all ADRs at once. Serves the scenarios "Owner sends a prompt…", "Owner explores an idea…", "Owner proposes a feature…".

## What changes
- lore no longer has a prompt hook: nothing is injected into the agent's context on the owner's messages, and there is no per-session dedup.
- `lore init` and `lore upgrade` no longer register a `UserPromptSubmit` hook, and they remove an existing `lore ctx --hook` entry from `.claude/settings.json`, keeping every other setting.
- `lore ctx` has no `--hook` flag any more: in a project not yet upgraded the old hook entry fails with an unknown-flag error on each prompt (exit 1, the prompt still goes through) until `lore upgrade` removes it.
- The agent calls `lore ctx "<text>"` when the owner asks it to look something up or when it decides it needs more knowledge; the text is the agent's own description of the work or question, never the owner's message quoted verbatim.
- Knowledge selection scores, in one selection with the `relates` question, every zone and every accepted ADR (an ADR of several zones once); requirements and sections are no longer scored one by one.
- The model judges a zone by its tldr together with its `## Invariants`, `## Boundaries` and `## Contracts`, so a zone related only through an invariant or a neighbour is still found; `INDEX.md` and the tldr itself don't change.
- The selection prints two lists: decisions — every ADR at or above `include_content` with its tldr and rules, with no budget, and ADRs between `include_pointer` and `include_content` by name under "maybe relevant" — and zones — every zone at or above the `zones` threshold printed whole, highest first, while they fit `ctx.budget_tokens`; a zone that doesn't fit is named with a hint to open it with `lore show`.
- `lore change status` picks knowledge for the next task with the same selection, but names the chosen zones instead of printing them; the fact check judges which zones a fact touches by the same zone text.
- `lore change status --next` prints only the next task and the knowledge picked for it; the full `lore change status` (delta summary, ADRs of the touched zones, drafts, delta text) is for the start of the work and after the agent's context was lost.
- `/lore-apply` runs the full status at the start and `status --next` before each following task.
- The zone budget of `lore ctx` is the `lore.yaml` key `ctx.budget_tokens` (default 8000); the `hook` keys (`enabled`, `min_words`, `timeout_ms`, `budget_tokens`) are no longer read.
- `lore upgrade` does not delete an existing `hook:` block from `lore.yaml`; it prints a hint to remove it, and `lore lint` warns while it is there.
- The `AGENTS.md` protocol block tells the agent to call `lore ctx` with the work or question in its own words when it needs project knowledge, `lore show <zone>` for a whole zone, and on a change the full `lore change status` at the start and `--next` before each following task.
- `/lore-propose`, `/lore-explore` and `/lore-why` get their context with `lore ctx` instead of opening every touched zone with `lore show`; `lore show` stays for when there is no API key and for a zone `lore ctx` did not print.

## Zones
- decisions-model
- install
- cli
- workflows
- docs
- changes

No delta for `adrs`: the selection reads ADRs, but what the zone gives and takes does not change.

## Scenarios
- changed: "Owner sets up lore in a project → docs skeleton, agent protocol, permissions, prompt hook and skills are in place" → "… docs skeleton, agent protocol, permissions and skills are in place"
- changed: "Owner upgrades lore in a project → …" — also removes the old prompt hook from the settings
- changed: "Owner sends a prompt → agent gets the relevant requirements and decisions in its context, each only once per session" → "Owner or agent needs project knowledge → `lore ctx` gives the relevant decisions and requirements for the work or question" — why: the hook is gone, the agent asks when it needs to
- changed: product "What it is": "Relevant knowledge is delivered to the AI agent automatically" → "The agent gets the relevant knowledge with one command"

## ADR drafts
- agent-pulls-knowledge-with-ctx (supersedes 2026-10-27-inject-content-per-requirement)

## Owner notes
> "хук не нужен на каждое мое сообщение, а только если я попрошу агента что-то поискать по этому поводу или он решит что надо доп инфа, и не напрямую меня цитировать при этом в lore ctx" — owner, 2026-10-27

> "в ctx даже не что нужно, а что из нашей инфы может относиться к этому вопросу" — owner, 2026-10-27

> Stage 1 removed: "хочется пока что меньше ошибок и посмотреть че по юзеджу будет" — owner, 2026-10-27

> Two separate lists without a common cap (ADRs and zones); `budget_tokens` stays in the config, renamed to `ctx.budget_tokens` (now a cap on whole zones), the old `hook:` block is not deleted (upgrade hints, lint warns); thresholds stay, incl. `include_pointer` for "maybe relevant"; the zones line is derived from the chosen items; upgrade removes the hook from settings — owner, 2026-10-27

> Whole zones instead of pieces: "а мож зоны полностью показывать всегда и мозга себе не ебать?"; zones picked by tldr, but the model sees invariants and boundaries/contracts too (option B, not longer tldrs): "Б, давай, интересно" — owner, 2026-10-27

> Full status repeats the same change context before every task: `lore change status --next` between tasks, full status at the start and after the context is lost; zones by name in status; in this change — owner, 2026-10-27

> `--hook` is removed, not kept as a no-op: "убираем флаг" — owner, 2026-10-27
