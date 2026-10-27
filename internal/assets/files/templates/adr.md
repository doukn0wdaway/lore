---
id: {{.ID}}
status: proposed            # proposed | accepted | superseded
zones: [{{.Zones}}]
tldr: ""                    # one concrete sentence: the decision itself
rules: []                   # e.g. ["DO: enqueue from cron handlers", "DON'T: call transport.Send() from a cron"]
supersedes: [{{.Supersedes}}]  # ids of ADRs this one replaces entirely
superseded_by: null
inferred: false             # true — inferred from code, the owner could not tell why
ref: null                   # optional: ticket / change
---
# {{.Title}}

## Context
<!-- What problem forced the decision. -->

## Decision
<!-- What we do. -->

## Why
<!-- REQUIRED. Why this way. Never invent it: unknown → ask the owner.
     If the owner does not know either: "Unknown (inferred from code)." and inferred: true -->

## Rejected alternatives
<!-- Optional: only alternatives that were really considered. Delete the section if none.
### <alternative>
Why not: ... -->

## Cost
<!-- Optional: what we knowingly accept. Delete if empty. -->

## Revisit if
<!-- Optional: conditions under which the decision may be challenged. Delete if empty. -->
