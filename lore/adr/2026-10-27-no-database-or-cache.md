---
id: 2026-10-27-no-database-or-cache
status: accepted
zones: [global]
tldr: "lore keeps no database, index cache or daemon: every command parses the markdown files as they are at that moment."
rules:
  - "DO: read the files on every call"
  - "DON'T: add a cache, an index database or a background process to speed lore up"
supersedes: []
superseded_by: null
inferred: false
ref: null
---
# No database or cache

## Context
Every command and every prompt hook call parses zones, ADRs and changes from markdown. A cache or an index database would make that faster.

## Decision
No database and no cache: the files are parsed on every call.

## Why
Freshness: what lore shows is always what the files say right now. (Owner, 2026-10-27: "актуальность".)
