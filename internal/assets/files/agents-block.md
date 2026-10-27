## lore

Project knowledge lives in `{{.Root}}/`. Zone list: @{{.Root}}/INDEX.md

{{if .Hook -}}
- Context for your task is injected automatically (`[lore]` block). Use it.
  Need more: `lore ctx "<what you need>"`; a whole zone: `lore show <zone>`.
{{else -}}
- Before working in a zone, open it with `lore show <zone>`: it prints the
  zone and the decisions (ADRs) that apply to it.
{{end -}}
- Accepted ADRs are binding. If your work contradicts one, stop and ask
  the owner — never work around it, never "fix" it silently.
- ADR without "Rejected alternatives": you may suggest an alternative —
  ask, don't act.
- Never invent the "why" of a decision. Unknown → ask.
- Don't create ADRs during implementation. Hit a fork → ask the owner.
{{if .Changes -}}
- Docs in `{{.Root}}/` (zones, ADRs, product, architecture) are the project's
  law. They change only at the gate (`lore change archive`) or through a lore
  workflow the owner invoked (bootstrap, zone, adr, product) — never on your
  own initiative.
- Working on a change: `lore change status <name>` before each task.
{{else -}}
- Docs in `{{.Root}}/` (zones, ADRs, product, architecture) are the project's
  law: change them only when the owner asks or a lore workflow says so.
{{end -}}
- After editing anything in `{{.Root}}/`, run `lore lint` and `lore index`.
{{if .Workflows -}}
- Workflows: {{.Workflows}}.
{{end -}}
