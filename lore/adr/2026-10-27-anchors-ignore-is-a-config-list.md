---
id: 2026-10-27-anchors-ignore-is-a-config-list
status: accepted
zones: [adrs]
tldr: "Paths excluded from the ADR-anchor check are an explicit anchors_ignore list in lore.yaml; lore has no built-in exclusions for tests or markdown."
rules:
  - "DO: add files with example anchors to anchors_ignore"
  - "DON'T: hard-code exclusions (tests, markdown, docs folders) into the anchor scan"
supersedes: []
superseded_by: null
inferred: false
ref: null
---
# anchors_ignore is an explicit list

## Context
In the lore repository itself, docs and tests contain example `ADR <id>` anchors that point at no real ADR, so `lore lint` could never be clean.

## Decision
`lore.yaml` has `anchors_ignore`: files, folders or globs (a pattern without `/` matches the file name at any depth) that the anchor scan skips. Nothing is excluded by default.

## Why
It is more logical and keeps it simple (KISS). (Owner, 2026-10-27.)

## Rejected alternatives
### Skip test files and markdown outside lore/ by default
Why not: less logical and less simple than an explicit list.
