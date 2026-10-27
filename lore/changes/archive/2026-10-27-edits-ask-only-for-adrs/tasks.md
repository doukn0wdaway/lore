# Tasks: Edits ask only for ADRs

## 1. Settings
- [x] 1.1 init/upgrade: add the `ask` rule only for `Edit` of `adr/**`; remove lore's old `ask` rules for zones, `product.md` and `architecture.md` (owner's rules stay); repoint the code anchor to the new ADR; tests

## 2. Agent texts
- [x] 2.1 Workflow sources: bootstrap and archive say only edits of `lore/adr/` ask the owner
- [x] 2.2 Rebuild, `lore upgrade` in this repo; check `.claude/settings.json` keeps only the `adr/**` ask rule

## 3. Verify
- [x] 3.1 `go test ./...` and `lore lint` pass
