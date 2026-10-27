---
id: 2026-10-27-upgrade-merges-config-as-text
status: accepted
zones: [install]
tldr: "lore upgrade adds missing lore.yaml keys by inserting their template lines as text; it never re-encodes the file through a YAML encoder."
rules:
  - "DO: insert new keys as text next to their template neighbour, keeping every existing line byte for byte"
  - "DON'T: unmarshal and re-marshal lore.yaml to add keys"
  - "DON'T: touch lore.yaml if the merged text would not parse — say what to add by hand"
supersedes: []
superseded_by: null
inferred: false
ref: "design.md §15"
---
# Upgrade merges lore.yaml as text

## Context
Newer lore versions add keys to `lore.yaml`. The simplest way to add them would be to load the YAML, fill in the defaults and write it back.

## Decision
`lore upgrade` finds the keys the project's file lacks and inserts the template's lines for them (with their comments) as text, after the neighbouring key from the template. Existing lines stay byte for byte. If the result would not parse, the file is left alone and the missing keys are named.

## Why
The config is edited by hand; rebuilding it through a YAML encoder breaks the formatting and the alignment of comments.

## Rejected alternatives
### Re-encode lore.yaml with a YAML library
Why not: breaks formatting and comment alignment.
