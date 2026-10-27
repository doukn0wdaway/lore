# Tasks: Ctx on demand

## 1. Selection
- [x] 1.1 One selection over every zone and every accepted ADR (each ADR once); a zone's model text = tldr + Invariants + Boundaries + Contracts, cut at 3,000 characters; the fact check uses the same zone text; tests
- [x] 1.2 Render two lists: decisions as content without a budget plus "maybe relevant"; zones above the `zones` threshold printed whole while they fit the budget, the rest named with the `lore show` hint; tests
- [x] 1.3 Config: `ctx.budget_tokens` (default 8000) in the config struct and the `lore.yaml` template; stop reading `hook.*`; lint warning for a `hook:` block; tests

## 2. No prompt hook
- [x] 2.1 Remove `lore ctx --hook`, per-session dedup and the `hook` log source; update `lore help`
- [x] 2.2 init/upgrade: stop registering the hook, remove an existing `lore ctx --hook` from `.claude/settings.json` (drop an emptied `UserPromptSubmit` entry); drop the hook feature flag; tests
- [x] 2.3 upgrade: print a hint when `lore.yaml` still has a `hook:` block (the block itself is kept); test
- [x] 2.4 `lore change status --next`: only the next task and the selection for it; full status names chosen zones instead of printing them; tests

## 3. Agent texts
- [x] 3.1 `agents-block.md`: run `lore ctx "<the work or question, in your own words>"` when you need knowledge, never the owner's message verbatim; `lore show <zone>` for a whole zone; on a change: full `lore change status` at the start, `--next` before each following task
- [x] 3.2 Workflow sources: propose, explore and why start from `lore ctx`, `lore show` only without a key or for a zone ctx did not print; apply runs full status at the start and `status --next` before each following task
- [x] 3.3 Rebuild, `lore upgrade` in this repo; check that `.claude/settings.json` lost the hook and `AGENTS.md` has the new text

## 4. Verify
- [x] 4.1 `go test ./...` and `lore lint` pass
- [x] 4.2 `lore ctx` on a few sample texts: the relevant ADRs as content, the right zones whole; look at the zone scores in the log
