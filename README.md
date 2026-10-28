# lore

Project knowledge for AI coding agents: **what** the system does, **why** it is built this way, and **what** is being changed right now — kept in the repo, delivered to the agent on demand with one command, changed only through a reviewed gate.

lore is a rethink of [OpenSpec](https://github.com/Fission-AI/OpenSpec): it keeps OpenSpec's requirement and delta format and adds first-class decisions (ADRs), context selection by a decisions model, a code-first mode and bootstrap of existing projects.

The current behaviour and the decisions behind it live in [`lore/`](lore/INDEX.md) (zones and ADRs). [`design.md`](design.md) (in Russian) is historical: the original design as of 2026-10-27 and its decision log, no longer updated.

## Why

- **Agents "fix" deliberate decisions.** They see odd code, don't know why it is that way, and simplify it back into a bug. Or they get lost in their own earlier decisions: old and new contradict each other.
- **Context is expensive.** Grepping and reading half the project on every task fills the context and multiplies the number of steps.
- **Docs rot.** Specs written before the code describe the plan, not what was built.

lore answers with three rules: every fact lives in one place; the "law" (zones and decisions) changes only through a gate where the owner sees the diff; the agent gets the knowledge that relates to its work with one command instead of grepping for it.

## How it works

```
lore/
  INDEX.md            generated — zone list, always in the agent's context
  product.md          what the app does and for whom: scenarios [live] / [planned]
  architecture.md     zones and how they depend on each other
  zones/<id>.md       how an area behaves: purpose, requirements (WHEN/THEN), invariants, contracts
  adr/<date>-<slug>.md  why it is built this way: decision, why, rejected alternatives
  changes/<name>/     work in progress: proposal, deltas to zones, tasks, ADR drafts
  workflows/          instructions for the agent (linked as Claude Code skills)
```

- **Zone** — an area of responsibility with its own rules (`notifications`, `finance`, `users`, `design-system`), not a folder. An action lives in the zone of the object it acts on; a reaction lives in the zone that reacts.
- **ADR** — a decision with a non-obvious "why" that a future agent might want to "fix". `## Why` is required; the reason comes from the owner and is never invented.
- **Change** — a feature in progress. **Spec-first:** proposal → owner's approval → deltas and tasks → code. **Code-first:** code in real time, then a change is created from the git diff. Both end at the same **gate**: reconcile with the code, check facts against decisions, merge deltas into zones, owner approves the diff, archive.
- **Decisions model** — a small model that answers "does this relate / contradict?" with probabilities. When the agent needs knowledge, `lore ctx` asks it which decisions and zones relate to the work; at the gate it flags facts that may contradict accepted decisions.

## Install

Requires Nix (or Go 1.23+).

```sh
nix profile install path:/path/to/lore      # puts `lore` on PATH
nix profile upgrade lore                    # later, after pulling changes
```

Without Nix: `CGO_ENABLED=0 go build -o ~/bin/lore ./cmd/lore`. The binary is static (no libc), so it can be copied to any machine of the same OS/arch.

## Quick start

```sh
cd your-project
lore init
```

`init` is idempotent and creates:

- `lore.yaml` and the `lore/` skeleton;
- a lore block in `AGENTS.md` (between `<!-- lore:begin -->` markers) and `@AGENTS.md` in `CLAUDE.md`;
- `.claude/settings.json`: `lore` may run freely, edits of ADRs ask the owner (zones, product and architecture are approved at the gate); an old `lore ctx --hook` prompt hook is removed;
- `.claude/skills/lore-*` symlinks to `lore/workflows/*`;
- `lore/.env` and `lore/.local/` in `.gitignore`.

Then give lore its key for the decisions model (an [OpenRouter](https://openrouter.ai) key, separate from your app's env):

```sh
mkdir -p ~/.config/lore
echo 'LORE_OPENROUTER_API_KEY=sk-or-...' >> ~/.config/lore/.env
chmod 600 ~/.config/lore/.env
```

Everything except `lore ctx`, `change check` and the model part of `change status --next` works without a key. Check the setup with `lore version` and `lore ctx "some task in this project"`.

**Existing project:** open Claude Code in it and run `/lore-bootstrap`. The agent derives product scenarios and a zone map from the code, writes the zones one by one, then interviews you about non-obvious decisions and records them as ADRs. You approve every step; progress survives between sessions.

**New project:** start describing features with `/lore-propose`; zones and decisions grow with the code.

## Working with lore

You mostly talk to the agent; the slash commands are Claude Code skills.

| You want to… | Run |
|---|---|
| document an existing project | `/lore-bootstrap` |
| think an idea through before building | `/lore-explore` |
| plan a feature (spec-first) | `/lore-propose` → approve → `/lore-apply` |
| document what you just coded (code-first) | `/lore-document` |
| finish a change | `/lore-archive` |
| record a decision | `/lore-adr` |
| ask why something is the way it is | `/lore-why` |
| add / update / split a zone | `/lore-zone` |
| add a wish to the backlog, mark a scenario live | `/lore-product` |
| check docs against the code | `/lore-audit` |

What the owner does: approves proposals and ADR drafts, answers "why?" and "new decision or bug?", approves the diff of `lore/` at the gate. The agent writes everything else; the CLI does the mechanical parts (statuses, superseding, merge verification, archiving).

## Guide

### Set up an existing project

1. `lore init`, put the key into `~/.config/lore/.env` (see Quick start).
2. In Claude Code: `/lore-bootstrap`. Six steps, each waits for your approval: product scenarios → zone map (the important one — iterate until it is right) → architecture → zones one by one → an interview about the "why" of non-obvious decisions (they become ADRs) → anchor comments in code. You can stop at any point; the next `/lore-bootstrap` resumes from `lore/.local/bootstrap.json`.
3. Commit `lore/`, `lore.yaml`, `AGENTS.md`, `CLAUDE.md` and `.claude/`.
4. If `lore lint` complains about anchors in files that only show examples (docs, tests), list those files under `anchors_ignore` in `lore.yaml`.

### Everyday work

Just talk to the agent. `INDEX.md` is always in its context; when you ask it to look something up, or it decides it needs more, it runs `lore ctx "<the work in its own words>"` and gets the related decisions and zones. Not your message verbatim: what matters is which of the project's knowledge relates to the work.

**Planning a feature (spec-first).**

1. `/lore-propose voice messages in Telegram` — the agent asks until it understands what and why, then writes `lore/changes/<name>/`: `proposal.md` (Why, What changes as one behavioural fact per bullet, Zones, Scenarios), `deltas/<zone>.md`, `tasks.md`, ADR drafts if a real decision is needed. It runs `lore change check` and asks you about every possible conflict with an accepted ADR.
2. You approve the proposal and the drafts.
3. `/lore-apply` — task by task; at the start the agent runs `lore change status` (next task, deltas, decisions of the touched zones, chosen zones by name), and before each following task `lore change status --next` (only the next task and what the model picks for it). At a fork it stops and asks; your answer goes to `## Owner notes`.
4. `/lore-archive` — the gate: deltas are reconciled with what the code really does, facts checked against ADRs, drafts settled, deltas merged into the zones, `product.md` updated. You see the diff of `lore/` and say yes; the agent runs `lore change archive --finish`.

**Documenting what you already coded (code-first).** Code as usual, then `/lore-document`. The agent creates a change from the git diff (base = merge-base with `main_branch`; if you worked on that branch itself, it asks where the work started), writes the facts, checks them against decisions, asks which of your choices are worth an ADR, writes deltas and continues with `/lore-archive`.

**Other everyday moves.**

| Situation | Say / run |
|---|---|
| "We decided X" | `/lore-adr` — the agent asks why, writes the ADR, accepts it after your yes |
| "Why is it done this way?" | `/lore-why` — answers from ADRs, or says plainly that nothing is recorded |
| Thinking about an idea | `/lore-explore` — no code, grounded in zones and decisions |
| "I want the app to …, later" | `/lore-product` — a `[planned]` scenario |
| A zone got too big / is wrong | `/lore-zone` |
| Docs feel stale | `/lore-audit` — a report, nothing is fixed without you |

### Deltas in short

```markdown
## ADDED Requirements
### Requirement: <new name>
...
## MODIFIED Requirements
### Requirement: <existing name>
<the whole requirement: description and every scenario that still holds>
## REMOVED Requirements
### Requirement: <existing name>
## RENAMED Requirements
- FROM: `### Requirement: Old`
- TO: `### Requirement: New`
## MODIFIED Sections
### Section: <title of a ## section of the zone>
<the whole new section>
```

- A MODIFIED block replaces the zone's block, so copy unchanged scenarios too; `lore change archive` warns about every scenario a MODIFIED drops.
- A new zone is a delta file with frontmatter holding the whole zone doc.
- `--finish` archives only when every merged block equals its delta (no missing and no left-over lines) and the whole project is lint-clean.

### What the agent is held to

- Accepted ADRs are binding. A contradiction is always your call: new decision (a new ADR supersedes the old one entirely) or a bug.
- The "why" of a decision is never invented; unknown reasons are recorded as unknown.
- Zones, ADRs, `product.md` and `architecture.md` change only at the gate or in a workflow you started. Edits of ADRs ask your permission; the rest you approve as one diff at the gate.
- ADR statuses change only through `lore adr accept`, after your yes.

### Keeping lore up to date

```sh
nix profile upgrade lore         # the binary
lore version
lore upgrade                     # in each project: workflows, prompts, new lore.yaml keys, AGENTS.md block
```

`upgrade` keeps workflows you edited locally and prints the diff to the new version; `--force` overwrites them.

### Troubleshooting

- **`lore ctx` finds nothing or the wrong things:** check the key; look at the scores of the call in `lore/.local/log.jsonl` (every `ctx`, `status` and `check` call is there, with errors); thresholds are in `lore.yaml`.
- **`lint` or `upgrade` mention a `hook:` block:** it is left over from the old prompt hook and no longer read — delete it from `lore.yaml`.
- **`lore lint` fails on anchors in example files:** add them to `anchors_ignore`.
- **`--finish` refuses:** it names the first missing, different or left-over line of each delta, or the lint errors elsewhere in the project — fix and run it again.
- **The binary seems stale:** `nix profile upgrade lore` after every change to lore's source; `lore version` tells which one you run.

## Decisions model

Configured in `lore.yaml`; default `perplexity/pplx-decider-v1.2` through OpenRouter's Decisions API.

- **`lore ctx "<text>"`** — the agent runs it when it needs knowledge, with its own description of the work or question. One selection scores every zone (its tldr with invariants, boundaries and contracts) and every accepted ADR. Output is two lists: decisions — ADRs above `include_content` as content, with no budget, and ADRs above `include_pointer` named as "maybe relevant"; zones — above the `zones` threshold, highest first, printed whole while they fit `ctx.budget_tokens`, the rest named with a `lore show` hint. Nothing reaches the agent's context unless it runs a command.
- **`lore change status --next`** runs the same selection for the text of the next task; full `change status` names the chosen zones.
- **`lore change check`** takes every fact under `## What changes` of a proposal and reports the zones it touches (and zones without a delta) and the accepted ADRs it may contradict. These are questions for the owner, not verdicts.
- Every call is logged to `lore/.local/log.jsonl`.

Russian texts against English docs work.

**The key** is lore's own: `LORE_OPENROUTER_API_KEY` from the environment, else `lore/.env` of the project (gitignored), else `~/.config/lore/.env`. The project's own `.env` is never read — it belongs to the app. Putting the key itself into `lore.yaml` is refused by every command.

## CLI

`[name]` may be omitted when exactly one change is active. `-h` / `--help` anywhere prints usage.

| Command | What it does |
|---|---|
| `lore init` | set up lore in the current directory (idempotent) |
| `lore index` | regenerate `lore/INDEX.md` |
| `lore lint [--json]` | check everything below; non-zero exit on errors |
| `lore show <zone>` | a zone plus the ADRs that apply to it (own and global) |
| `lore adr new <slug> --zones a,b [--title …] [--supersedes id] [--change name]` | ADR from the template, `status: proposed`; with `--change` a draft inside that change |
| `lore adr accept <id>` | the owner's approval: `accepted`; superseded ADRs are marked (for drafts — at archive) |
| `lore adr list [--zone z] [--status s] [--json]` | list ADRs |
| `lore change new <name> [--from-diff [--base ref]]` | start a change; `--from-diff` = code-first from the merge-base with the main branch |
| `lore change list` | active changes with mode and progress |
| `lore change status [name] [--next]` | next task, the change's deltas, decisions of the touched zones, chosen zones by name; `--next`: only the next task and what the model picks for it |
| `lore change check [name]` | facts vs zones and accepted ADRs |
| `lore change archive [name]` | pre-merge checks (tasks done, deltas fit the current zones, drafts valid; warns when a MODIFIED drops a scenario) |
| `lore change archive [name] --finish` | after merge and the owner's yes: verify every merged block equals its delta, require the whole project lint-clean, move ADR drafts, supersede, archive |
| `lore ctx "<text>"` | decisions and zones related to a text (your own description of the work or question) |
| `lore agents sync` | (re)link workflows into `.claude/skills/lore-*` |
| `lore upgrade [--force]` | update workflows and prompts (local edits kept unless `--force`), add new `lore.yaml` keys, rewrite the `AGENTS.md` block, migrate old layouts |

**`lore lint` checks:** zone and ADR frontmatter (ids match file names, `tldr`, statuses, zone references, supersede chains), required sections (`## Purpose`, `## Context` / `## Decision` / `## Why`), duplicate requirements, entry-point paths exist, `ADR <id>` anchors in code point at existing, non-superseded ADRs (except paths in `anchors_ignore`), `INDEX.md` is fresh, `product.md` scenarios name existing zones, change and delta format, two active changes editing the same requirement (warning), a key put into `lore.yaml`.

## Configuration

`lore.yaml` at the project root (the folder lore looks for, like git looks for `.git`):

```yaml
root: lore                       # docs folder
main_branch: main                # base for `change new --from-diff`

decisions_model: perplexity/pplx-decider-v1.2
decisions_api_key_env: LORE_OPENROUTER_API_KEY   # the NAME of the variable, not the key
decisions_endpoint: https://openrouter.ai/api/alpha/decisions
decisions_provider: [Perplexity]

ctx:
  budget_tokens: 8000            # how much of whole zones `lore ctx` prints

thresholds:
  zones: 0.3                     # the zone is related — low, never lose a zone
  include_content: 0.7           # show the ADR's content
  include_pointer: 0.4           # name the ADR as maybe relevant
  contradiction: 0.5             # ask the owner at the gate

prompts:                         # question texts for the model, editable per project
  relates: lore/prompts/relates.txt
  contradicts: lore/prompts/contradicts.txt

anchors_ignore: []               # paths lint skips when checking ADR anchors: files, folders,
                                 # globs; a pattern without a slash matches the file name anywhere
```

Workflows in `lore/workflows/` may be edited per project; `lore upgrade` keeps local edits and shows the diff of the new version.

## Development

```sh
nix develop                      # go, gopls
go test ./...
CGO_ENABLED=0 go build -o bin/lore ./cmd/lore
nix build path:.                 # → result/bin/lore (static)
```

```
cmd/lore/                CLI: commands and flag parsing
internal/lore/           project, markdown parsing, lint, index, show, adr, changes and deltas,
                         decisions model client, selection, ctx, install / upgrade
internal/assets/files/   embedded: lore.yaml, AGENTS.md block, templates, workflows, prompts
design.md                historical: the original design and decision log
lore/                    lore documents itself: zones, ADRs, product, architecture
```

lore uses lore: read `lore/INDEX.md` and the zones before changing the code; accepted ADRs in `lore/adr/` are binding. The source of the workflows is `internal/assets/files/workflows/` — edit there, rebuild, then `bin/lore upgrade` to update the installed copies in `lore/workflows/`.

## Status

All three planned stages are implemented: format and bootstrap, changes with the gate, decisions model. Next is a pilot on a real project to tune thresholds and check what only real use shows. Deliberately not done yet: quality metrics for context selection (`lore eval`), a docs-drift detector, links between requirements and tests, risk levels, OpenSpec migration, support for agents other than Claude Code.
