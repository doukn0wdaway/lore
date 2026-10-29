---
name: lore-archive
description: The lore gate — finish a change (spec-first or code-first): reconcile deltas with what the code really does, check facts against decisions, settle ADR drafts, merge deltas into the zone docs, show the owner the diff and archive. Use when the owner asks to archive, finish, close or merge the current lore change, or when `lore change status` says all work is done.
---

# lore archive — the gate

The only way the project's law (zones, ADRs, product, architecture) changes. What gets in must be **what the code does**, not what was planned.

The change name may be omitted in every command when only one change is active.

## 1. Pre-check

```sh
lore change archive [name]
```

Fix every error it reports (unfinished tasks, deltas pointing at requirements that don't exist, broken drafts). Warnings: read them — another active change touching the same requirement means merging carefully on top of its text.

## 2. Reconcile deltas with the code (spec-first)

The deltas were written before the code. Compare them with `git diff <base>` (base in `.meta.yaml`) and the `## Owner notes` in `proposal.md`:

- behaviour that appeared during the work → add it to the deltas;
- planned behaviour that was dropped → remove it from the deltas;
- deviations the owner asked for (Owner notes) are intended; unexplained deviations → ask the owner.

Update `## What changes` in `proposal.md` to match. (Code-first changes did this in `/lore-document`.)

## 3. Facts vs decisions

```sh
lore change check [name]
```

The decisions model reads every bullet of `## What changes` and reports the zones each fact touches (zones without a delta are listed — add a delta or explain why not) and the accepted ADRs it **may** contradict. Treat its output as candidates: read each flagged ADR and decide whether the conflict is real. Without an API key, do the same by hand: `lore show <zone>` for every touched zone and compare each fact with its decisions.

Every real contradiction with an accepted ADR of the touched zones or a global one is a question to the owner — **new decision** (→ ADR draft with `--supersedes`, see `/lore-adr`) or **bug** (→ fix the code first; the change is not archived until then). Never decide yourself.

## 4. ADR drafts

Every draft in `adr/` of the change must be `accepted` by the owner (`lore adr accept <id>`), or deleted if the owner drops it. Code-first: offer decision candidates now if `/lore-document` didn't.

Run `lore change archive [name]` again — it must be clean.

## 5. Merge

Apply each `deltas/<zone>.md` to `lore/zones/<zone>.md` (no permission prompt here — the owner approves the whole diff in step 6):

| Delta | Merge |
|---|---|
| ADDED | add the requirement under `## Requirements`; if it already exists, replace it |
| MODIFIED | replace the whole requirement block with the delta's block |
| REMOVED | delete the whole requirement block |
| RENAMED | rename the `### Requirement:` heading, keep its body |
| MODIFIED Sections | replace the whole `## <title>` section |
| new zone (frontmatter) | create `lore/zones/<id>.md` from the delta |

Then:

- `lore/product.md` — apply `## Scenarios` of the proposal: `[planned]` → `[live]` (add if missing; a live scenario keeps no future zone `<id>?` — the zone this change created, or the zones it really passes through), rewrite changed scenarios in place, delete removed ones. product.md describes the present; history stays in the archived change and git. If the reason a scenario was dropped is non-obvious and an agent might propose it again, that is an ADR ("we don't do X because …") — ask the owner;
- `lore/architecture.md`: update if zones or dependencies changed;
- if a superseded ADR has code anchors (`grep -rn "ADR <old-id>"`), repoint them to the new id.

The merge is idempotent: running it twice gives the same result.

## 6. Owner's yes

```sh
lore index
lore lint
git diff -- lore/
```

Show the owner the diff of `lore/` (summarise if long, but don't hide anything) and wait for an explicit yes.

## 7. Finish

```sh
lore change archive [name] --finish
```

It verifies the merge (every ADDED/MODIFIED requirement and MODIFIED section equals the delta's block exactly — no line missing, none left over from the old text; REMOVED gone, renames applied, new zones exist), moves accepted ADR drafts to `lore/adr/` and marks the ADRs they supersede, regenerates the index and moves the change to `lore/changes/archive/<date>-<name>/`. If it reports errors, fix and run it again.
