---
name: lore-document
description: Code-first — document work already done in real time without specs: create a lore change from the git diff, summarise what the code now does as plain facts, check them against existing decisions, ask the owner about decisions worth recording, write deltas, then hand over to the archive gate. Use when the owner says "document this", "update the docs/lore for what we did", or after coding without a lore change.
---

# lore document — code-first

The owner made the decisions while coding; your job is to turn the diff into lore without inventing anything.

## 1. Create the change from the diff

```sh
lore change new <kebab-name> --from-diff [--base <ref>]
```

The base defaults to the merge-base with `main_branch` from `lore.yaml`. If the work is on the main branch itself, ask the owner where it started and pass `--base <commit>`. The base is stored in `.meta.yaml`.

## 2. Read the diff

`git diff <base>` (committed and uncommitted work; the base is in `lore/changes/<name>/.meta.yaml`). Add `git status` for untracked files.

## 3. Summarise as facts

In `proposal.md` → `## What changes`: one **behavioural** fact per bullet, plain language — what the system now does differently, not which files changed.

> - Voice messages in Telegram are accepted and transcribed in the background.
> - Voice messages longer than 5 minutes are rejected.

Fill `## Why` from what the owner said in the session; if you don't know why, ask. Fill `## Zones` and `## Scenarios` — including `[live]` scenarios the diff changed or removed.

## 4. Check facts against decisions

`lore change check [name]` — the decisions model lists, per fact, the touched zones and the accepted ADRs it may contradict; confirm each flagged one by reading it. Without an API key: `lore show <zone>` for every touched zone. For each real contradiction with an accepted ADR (zone or global) ask the owner: **new decision** or **bug in the code**?

- new decision → ADR draft with `--supersedes <old-id>` (step 5);
- bug → it must be fixed before the change can be archived; tell the owner.

## 5. Decision candidates

Look for choices in the diff that pass the ADR test (non-obvious "why", a future agent might "fix" it). Ask the owner one at a time: *"You did X rather than Y — record it as a decision? Why this way?"* On yes:

```sh
lore adr new <slug> --zones a,b --change <name> [--supersedes <old-id>]
```

Fill it with the owner's reasoning (never invented), then `lore adr accept <id>` after their yes. On no — nothing.

## 6. Write deltas

`deltas/<zone>.md` per touched zone, OpenSpec format (ADDED / MODIFIED / REMOVED / RENAMED Requirements, MODIFIED Sections; a new zone = the whole zone doc) — see `/lore-propose` for the format. They describe the behaviour that is now in the code.

No `tasks.md` in code-first.

## 7. Gate

Continue with `/lore-archive`.
