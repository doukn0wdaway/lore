package lore

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDocSkipsFencesAndFindsSections(t *testing.T) {
	src := "---\nid: x\n---\n# Title\n\n## Purpose\nText\n```md\n## Not a heading\n```\n## Requirements\n### Requirement: A\n#### Scenario: s\n### Requirement: B\n## Other\n"
	d := parseDoc(src)
	if !d.hasFront || d.front != "id: x" {
		t.Fatalf("front = %q", d.front)
	}
	if d.title() != "Title" {
		t.Fatalf("title = %q", d.title())
	}
	text, line, ok := d.section(2, "purpose")
	if !ok || !strings.Contains(text, "## Not a heading") || line != 6 {
		t.Fatalf("purpose = %q line %d ok %v", text, line, ok)
	}
	reqs := d.requirements()
	if len(reqs) != 2 || reqs[0].name != "A" || reqs[1].name != "B" || reqs[0].line != 12 {
		t.Fatalf("requirements = %+v", reqs)
	}
	if text, _, _ := d.section(2, "Requirements"); !strings.Contains(text, "Requirement: B") {
		t.Fatalf("requirements section cut early: %q", text)
	}
}

func TestIsBlank(t *testing.T) {
	if !isBlank("\n<!-- hint\nmore -->\n  ") {
		t.Fatal("comment-only text should be blank")
	}
	if isBlank("<!-- hint --> real") {
		t.Fatal("text with content is not blank")
	}
}

func newProject(t *testing.T) *Project {
	t.Helper()
	dir := t.TempDir()
	p, err := Init(dir, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func write(t *testing.T, path, src string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func messages(issues []Issue, sev Severity) string {
	var b strings.Builder
	for _, i := range issues {
		if i.Severity == sev {
			b.WriteString(i.String() + "\n")
		}
	}
	return b.String()
}

func TestInitIsCleanAndIdempotent(t *testing.T) {
	p := newProject(t)
	issues, err := p.Lint()
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("fresh project has issues:\n%s", messages(issues, SevError)+messages(issues, SevWarn))
	}
	agents, _ := os.ReadFile(filepath.Join(p.Root, "AGENTS.md"))
	if _, err := Init(p.Root, io.Discard); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(filepath.Join(p.Root, "AGENTS.md"))
	if string(agents) != string(again) || strings.Count(string(again), blockBegin) != 1 {
		t.Fatalf("second init changed AGENTS.md:\n%s", again)
	}
	if link, err := os.Readlink(filepath.Join(p.Root, ".claude/skills/lore-bootstrap")); err != nil || link != "../../lore/workflows/bootstrap" {
		t.Fatalf("skill link = %q, %v", link, err)
	}
}

func TestMergeConfig(t *testing.T) {
	old := `# my project
root: docs-lore      # custom
main_branch: trunk
decisions_api_key_env: MY_KEY

hook:
  enabled: false     # not yet
  min_words: 6

prompts:
  relates: docs-lore/prompts/relates.txt
`
	tmpl := `root: lore
main_branch: main
# Decisions model
decisions_model: m
decisions_api_key_env: LORE_OPENROUTER_API_KEY
decisions_provider: [Perplexity]  # provider order

hook:
  enabled: true
  min_words: 4
  timeout_ms: 6000

thresholds:
  zones: 0.3

prompts:
  relates: lore/prompts/relates.txt
  contradicts: lore/prompts/contradicts.txt
`
	out, added, err := mergeConfig(old, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(added, ","); got != "decisions_model,decisions_provider,hook.timeout_ms,thresholds,prompts.contradicts" {
		t.Fatalf("added = %s\n%s", got, out)
	}
	if !strings.HasPrefix(out, "# my project\nroot: docs-lore      # custom\nmain_branch: trunk\n") {
		t.Fatalf("existing text changed:\n%s", out)
	}
	for _, want := range []string{"  min_words: 6\n  timeout_ms: 6000\n", "decisions_api_key_env: MY_KEY\ndecisions_provider: [Perplexity]", "enabled: false     # not yet", "# Decisions model\ndecisions_model: m", "decisions_provider: [Perplexity]  # provider order", "  contradicts: lore/prompts/contradicts.txt"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	again, added2, _ := mergeConfig(out, tmpl)
	if len(added2) != 0 || again != out {
		t.Fatalf("second merge changed something: %v", added2)
	}
}

func TestUpgradeMovesLegacyLocalDir(t *testing.T) {
	p := newProject(t)
	write(t, filepath.Join(p.Root, ".lore", "bootstrap.json"), `{"step": 2}`)
	write(t, filepath.Join(p.Root, ".lore", ".env"), "LORE_OPENROUTER_API_KEY=x\n")
	write(t, filepath.Join(p.Root, ".gitignore"), "bin/\n.lore/\n")
	if err := p.Upgrade(io.Discard, false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(p.LocalDir(), "bootstrap.json")); string(b) != `{"step": 2}` {
		t.Fatal("bootstrap progress not moved")
	}
	if _, err := os.Stat(p.EnvFile()); err != nil {
		t.Fatal(".env not moved")
	}
	if _, err := os.Stat(filepath.Join(p.Root, ".lore")); err == nil {
		t.Fatal(".lore/ still exists")
	}
	gi, _ := os.ReadFile(filepath.Join(p.Root, ".gitignore"))
	if strings.Contains(string(gi), ".lore/") || !strings.Contains(string(gi), "lore/.env") || !strings.Contains(string(gi), "lore/.local/") {
		t.Fatalf(".gitignore:\n%s", gi)
	}
}

func TestInitKeepsExistingAgentsAndSettings(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "AGENTS.md"), "# Mine\n\nKeep me.\n")
	write(t, filepath.Join(dir, ".claude/settings.json"), `{"permissions":{"allow":["Bash(go:*)"]},"other":1}`)
	if _, err := Init(dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	agents, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.HasPrefix(string(agents), "# Mine\n\nKeep me.\n\n"+blockBegin) {
		t.Fatalf("AGENTS.md = %q", agents)
	}
	for _, want := range []string{`lore ctx "<the work or question, in your own words>"`, "never the\n  owner's message verbatim", "--next` before each next task"} {
		if !strings.Contains(string(agents), want) {
			t.Fatalf("AGENTS.md block misses %q:\n%s", want, agents)
		}
	}
	if strings.Contains(string(agents), "injected automatically") {
		t.Fatalf("no automatic injection any more:\n%s", agents)
	}
	settings, _ := os.ReadFile(filepath.Join(dir, ".claude/settings.json"))
	for _, want := range []string{`"Bash(go:*)"`, `"Bash(lore:*)"`, `"Edit(/lore/adr/**)"`, `"other": 1`} {
		if !strings.Contains(string(settings), want) {
			t.Fatalf("settings missing %s:\n%s", want, settings)
		}
	}
	if strings.Contains(string(settings), "UserPromptSubmit") {
		t.Fatalf("no prompt hook is registered:\n%s", settings)
	}
}

func TestUpgradeRemovesOldPromptHook(t *testing.T) {
	p := newProject(t)
	path := filepath.Join(p.Root, ".claude/settings.json")
	write(t, path, `{"hooks":{
  "UserPromptSubmit":[
    {"hooks":[{"type":"command","command":"lore ctx --hook","timeout":10}]},
    {"matcher":"x","hooks":[{"type":"command","command":"lore ctx --hook"},{"type":"command","command":"mine"}]}
  ],
  "Stop":[{"hooks":[{"type":"command","command":"other"}]}]
}}`)
	if err := p.Upgrade(io.Discard, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "lore ctx --hook") {
		t.Fatalf("old hook kept:\n%s", got)
	}
	for _, want := range []string{`"command": "mine"`, `"matcher": "x"`, `"command": "other"`, `"Bash(lore:*)"`} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("settings lost %s:\n%s", want, got)
		}
	}
	if strings.Count(string(got), `"hooks": [`) != 2 {
		t.Fatalf("the emptied UserPromptSubmit entry should be gone:\n%s", got)
	}

	write(t, path, `{"hooks":{"UserPromptSubmit":[{"hooks":[{"type":"command","command":"lore ctx --hook"}]}]}}`)
	if err := p.Upgrade(io.Discard, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); strings.Contains(string(got), `"hooks"`) {
		t.Fatalf("an empty hooks map should be dropped:\n%s", got)
	}
}

func TestSettingsAskOnlyForADRs(t *testing.T) {
	p := newProject(t)
	path := filepath.Join(p.Root, ".claude/settings.json")
	write(t, path, `{"permissions":{"ask":["Edit(/lore/zones/**)","Edit(/lore/adr/**)","Edit(/lore/product.md)","Edit(/lore/architecture.md)","Edit(/secrets/**)"]}}`)
	if err := p.Upgrade(io.Discard, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	for _, gone := range []string{"zones/**", "product.md", "architecture.md"} {
		if strings.Contains(string(got), gone) {
			t.Errorf("old ask rule for %s kept:\n%s", gone, got)
		}
	}
	for _, want := range []string{`"Edit(/lore/adr/**)"`, `"Edit(/secrets/**)"`, `"Bash(lore:*)"`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("settings lost %s:\n%s", want, got)
		}
	}
	before := string(got)
	if err := p.Upgrade(io.Discard, false); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(path); string(again) != before {
		t.Fatalf("second upgrade changed settings:\n%s", again)
	}
}

func TestUpgradeHintsAboutOldHookBlock(t *testing.T) {
	p := newProject(t)
	cfgPath := filepath.Join(p.Root, ConfigFile)
	cfg, _ := os.ReadFile(cfgPath)
	old := string(cfg) + "\nhook:\n  enabled: true   # mine\n  min_words: 4\n"
	write(t, cfgPath, old)
	var out strings.Builder
	if err := p.Upgrade(&out, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "`hook:` is no longer used") {
		t.Fatalf("no hint:\n%s", out.String())
	}
	if got, _ := os.ReadFile(cfgPath); !strings.Contains(string(got), "hook:\n  enabled: true   # mine\n  min_words: 4\n") {
		t.Fatalf("the hook block must be kept as is:\n%s", got)
	}
	out.Reset()
	write(t, cfgPath, string(cfg))
	if err := p.Upgrade(&out, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "hook") {
		t.Fatalf("hint without a hook block:\n%s", out.String())
	}
}

func TestLint(t *testing.T) {
	p := newProject(t)
	write(t, p.Path("zones/notifications.md"), `---
id: notifications
tldr: Scheduled and event-driven notifications; crons only enqueue.
---
# Notifications

## Purpose
Reminders.

### Requirement: Reminder
### Requirement: Reminder

## Entry points
- `+"`src/missing.go`"+` — gone
- `+"`notify.Enqueue()`"+` — a symbol, skipped
`)
	write(t, p.Path("zones/x.md"), "---\nid: y\ntldr: t\n---\n# X\n")
	write(t, p.Path("adr/2026-10-27-queue.md"), `---
id: 2026-10-27-queue
status: superseded
zones: [notifications, ghost]
tldr: Crons only enqueue jobs, a worker delivers them.
---
# Queue

## Context
c

## Decision
d

## Why
<!-- todo -->
`)
	write(t, filepath.Join(p.Root, "main.go"), "// ADR 2026-10-27-queue: x\n// ADR 2026-01-01-none: y\n")

	issues, err := p.Lint()
	if err != nil {
		t.Fatal(err)
	}
	errs := messages(issues, SevError)
	for _, want := range []string{
		`duplicate requirement "Reminder"`,
		`entry point "src/missing.go" does not exist`,
		`id "y" does not match the file name "x"`,
		`lore/zones/x.md: missing ## Purpose`,
		`zones: no zone "ghost"`,
		`## Why is empty`,
		`status superseded without superseded_by`,
		`anchor refers to superseded ADR "2026-10-27-queue"`,
		`anchor refers to unknown ADR "2026-01-01-none"`,
		`INDEX.md: out of date`,
	} {
		if !strings.Contains(errs, want) {
			t.Errorf("missing error %q in:\n%s", want, errs)
		}
	}
	if strings.Contains(errs, "Enqueue") {
		t.Errorf("symbol entry point was checked as a path:\n%s", errs)
	}
	if warns := messages(issues, SevWarn); !strings.Contains(warns, `tldr is too short`) {
		t.Errorf("missing short-tldr warning:\n%s", warns)
	}
}

func TestLintAnchorsIgnore(t *testing.T) {
	p := newProject(t)
	for _, f := range []string{"design.md", "docs/a.md", "examples/x.go", "pkg/a/b_test.go", "main.go"} {
		write(t, filepath.Join(p.Root, f), "// ADR 2026-01-01-none: example\n")
	}
	p.Cfg.AnchorsIgnore = []string{"design.md", "docs/", "./examples", "*_test.go"}

	issues, err := p.Lint()
	if err != nil {
		t.Fatal(err)
	}
	errs := messages(issues, SevError)
	if !strings.Contains(errs, "main.go:1: anchor refers to unknown ADR") {
		t.Errorf("anchor outside anchors_ignore not reported:\n%s", errs)
	}
	if n := strings.Count(errs, "anchor refers"); n != 1 {
		t.Errorf("want 1 anchor error, got %d:\n%s", n, errs)
	}
}

func TestNewADRAndShow(t *testing.T) {
	p := newProject(t)
	write(t, p.Path("zones/storage.md"), "---\nid: storage\ntldr: Files, OCR and whisper, dialogue history and facts.\n---\n# Storage\n\n## Purpose\nKeeps things.\n")
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	if _, err := p.NewADR("hash-names", []string{"nope"}, "", nil, "", now); err == nil {
		t.Fatal("unknown zone accepted")
	}
	path, err := p.NewADR("hash-names", []string{"storage"}, "", nil, "", now)
	if err != nil || path != "lore/adr/2026-10-27-hash-names.md" {
		t.Fatalf("path %q err %v", path, err)
	}
	if _, err := p.NewADR("hash-names", []string{"storage"}, "", nil, "", now); err == nil {
		t.Fatal("duplicate ADR accepted")
	}
	src, _ := os.ReadFile(filepath.Join(p.Root, path))
	if !strings.Contains(string(src), "zones: [storage]") || !strings.Contains(string(src), "# Hash names") {
		t.Fatalf("template not filled:\n%s", src)
	}
	var out strings.Builder
	if err := p.Show(&out, "storage"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Proposed, not binding yet") || !strings.Contains(out.String(), "2026-10-27-hash-names") {
		t.Fatalf("show output:\n%s", out.String())
	}
}

func fillADR(t *testing.T, p *Project, rel string) {
	t.Helper()
	path := filepath.Join(p.Root, rel)
	b, _ := os.ReadFile(path)
	s := strings.Replace(string(b), `tldr: ""`, `tldr: "Files are stored by content hash, never by name."`, 1)
	for _, sec := range []string{"Context", "Decision", "Why"} {
		s = strings.Replace(s, "## "+sec+"\n", "## "+sec+"\nFilled.\n", 1)
	}
	write(t, path, s)
}

func TestAcceptAndSupersede(t *testing.T) {
	p := newProject(t)
	write(t, p.Path("zones/storage.md"), "---\nid: storage\ntldr: Files, OCR and whisper, dialogue history and facts.\n---\n# Storage\n\n## Purpose\nKeeps things.\n")
	if _, err := p.WriteIndex(); err != nil {
		t.Fatal(err)
	}
	day1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)

	old, _ := p.NewADR("by-name", []string{"storage"}, "", nil, "", day1)
	if _, err := p.AcceptADR("2026-10-01-by-name"); err == nil || !strings.Contains(err.Error(), "## Why is empty") {
		t.Fatalf("accepted an unfilled ADR: %v", err)
	}
	fillADR(t, p, old)
	if _, err := p.AcceptADR("2026-10-01-by-name"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.NewADR("by-hash", []string{"storage"}, "", []string{"2026-10-02-ghost"}, "", day2); err == nil {
		t.Fatal("superseding an unknown ADR accepted")
	}
	next, err := p.NewADR("by-hash", []string{"storage"}, "", []string{"2026-10-01-by-name"}, "", day2)
	if err != nil {
		t.Fatal(err)
	}
	fillADR(t, p, next)
	changed, err := p.AcceptADR("2026-10-02-by-hash")
	if err != nil || len(changed) != 2 {
		t.Fatalf("changed %v err %v", changed, err)
	}
	adrs, _ := p.LoadADRs()
	o, n := findADR(adrs, "2026-10-01-by-name"), findADR(adrs, "2026-10-02-by-hash")
	if o.Status != "superseded" || o.SupersededBy != n.ID || n.Status != "accepted" {
		t.Fatalf("old %s/%s new %s", o.Status, o.SupersededBy, n.Status)
	}
	issues, _ := p.Lint()
	if e := messages(issues, SevError); e != "" {
		t.Fatalf("lint after supersede:\n%s", e)
	}
	if _, err := p.AcceptADR("2026-10-02-by-hash"); err == nil {
		t.Fatal("accepted twice")
	}
}

func TestChangeLifecycle(t *testing.T) {
	p := newProject(t)
	zone := "---\nid: storage\ntldr: Files, OCR and whisper, dialogue history and facts.\n---\n# Storage\n\n## Purpose\nKeeps things.\n\n## Requirements\n\n### Requirement: Size limit\n#### Scenario: big\n- **WHEN** a file is over 20 MB\n- **THEN** reject\n"
	write(t, p.Path("zones/storage.md"), zone)
	if _, err := p.WriteIndex(); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	oldPath, _ := p.NewADR("by-name", []string{"storage"}, "", nil, "", now.AddDate(0, 0, -30))
	fillADR(t, p, oldPath)
	if _, err := p.AcceptADR("2026-09-27-by-name"); err != nil {
		t.Fatal(err)
	}

	if _, err := p.NewChange("voice", false, "", now); err != nil {
		t.Fatal(err)
	}
	c, err := p.FindChange("") // the only active change
	if err != nil || c.Name != "voice" || c.Meta.Mode != ModeSpec {
		t.Fatalf("change %+v err %v", c, err)
	}
	dir := c.Dir
	write(t, filepath.Join(dir, "tasks.md"), "## 1. Storage\n- [x] 1.1 table\n- [ ] 1.2 transcription\n")
	write(t, filepath.Join(dir, "deltas/storage.md"), "## ADDED Requirements\n### Requirement: Voice\nThe system SHALL transcribe.\n\n## MODIFIED Requirements\n### Requirement: Size limit\n#### Scenario: voice\n- **WHEN** voice > 5 min\n- **THEN** reject\n\n## REMOVED Requirements\n### Requirement: Ghost\n")
	write(t, filepath.Join(dir, "deltas/inbox.md"), "---\nid: inbox\ntldr: Incoming messages queue before they reach the dialogue.\n---\n# Inbox\n\n## Purpose\nQueues messages.\n")

	draft, err := p.NewADR("by-hash", []string{"storage", "inbox"}, "", []string{"2026-09-27-by-name"}, "voice", now)
	if err != nil || !strings.HasPrefix(draft, "lore/changes/voice/adr/") {
		t.Fatalf("draft %q err %v", draft, err)
	}
	fillADR(t, p, draft)

	c, _ = p.FindChange("voice")
	issues, err := p.ArchiveCheck(c)
	if err != nil {
		t.Fatal(err)
	}
	errs := messages(issues, SevError)
	for _, want := range []string{"task not done: 1.2 transcription", `requirement "Ghost" not found`} {
		if !strings.Contains(errs, want) {
			t.Errorf("pre-check missing %q:\n%s", want, errs)
		}
	}
	if !strings.Contains(messages(issues, SevWarn), "must accept it") {
		t.Errorf("pre-check should warn about the unaccepted draft:\n%s", messages(issues, SevWarn))
	}

	write(t, filepath.Join(dir, "tasks.md"), "## 1. Storage\n- [x] 1.1 table\n- [x] 1.2 transcription\n")
	write(t, filepath.Join(dir, "deltas/storage.md"), "## ADDED Requirements\n### Requirement: Voice\nThe system SHALL transcribe.\n")
	if _, err := p.AcceptADR("2026-10-27-by-hash"); err != nil {
		t.Fatal(err)
	}
	all, _ := p.LoadADRs()
	if findADR(all, "2026-09-27-by-name").Status != "accepted" {
		t.Fatal("accepting a draft superseded the old ADR before archive")
	}
	c, _ = p.FindChange("voice")
	if issues, _ := p.ArchiveCheck(c); CountErrors(issues) > 0 {
		t.Fatalf("pre-check after fixes:\n%s", messages(issues, SevError))
	}

	issues, err = p.ArchiveFinish(c, now, io.Discard)
	if err != nil || !strings.Contains(messages(issues, SevError), `not merged: requirement "Voice"`) {
		t.Fatalf("finish before merge: %v\n%s", err, messages(issues, SevError))
	}

	// A merge that drops the delta's text is caught.
	write(t, p.Path("zones/storage.md"), zone+"\n### Requirement: Voice\nSomething else.\n")
	c, _ = p.FindChange("voice")
	issues, _ = p.ArchiveFinish(c, now, io.Discard)
	if !strings.Contains(messages(issues, SevError), `not merged into lore/zones/storage.md › Requirement: Voice: expected "The system SHALL transcribe.", found "Something else."`) {
		t.Fatalf("unfaithful merge not caught:\n%s", messages(issues, SevError))
	}

	// The agent merges.
	write(t, p.Path("zones/storage.md"), zone+"\n### Requirement: Voice\nThe system SHALL transcribe.\n")
	write(t, p.Path("zones/inbox.md"), "---\nid: inbox\ntldr: Incoming messages queue before they reach the dialogue.\n---\n# Inbox\n\n## Purpose\nQueues messages.\n")
	c, _ = p.FindChange("voice")
	issues, err = p.ArchiveFinish(c, now, io.Discard)
	if err != nil || len(issues) > 0 {
		t.Fatalf("finish: %v\n%s", err, messages(issues, SevError))
	}
	if _, err := os.Stat(p.Path("changes/archive/2026-10-27-voice/proposal.md")); err != nil {
		t.Fatal("change not archived")
	}
	all, _ = p.LoadADRs()
	o, n := findADR(all, "2026-09-27-by-name"), findADR(all, "2026-10-27-by-hash")
	if n == nil || n.Status != "accepted" || o.Status != "superseded" || o.SupersededBy != n.ID {
		t.Fatalf("after archive: old %+v new %+v", o, n)
	}
	if issues, _ := p.Lint(); CountErrors(issues) > 0 {
		t.Fatalf("lint after archive:\n%s", messages(issues, SevError))
	}
	if _, err := p.FindChange(""); err == nil {
		t.Fatal("archived change still active")
	}
}

func TestUpgradeKeepsLocalEdits(t *testing.T) {
	p := newProject(t)
	skill := p.Path("workflows/bootstrap/SKILL.md")
	orig, _ := os.ReadFile(skill)
	write(t, skill, string(orig)+"local\n")
	if err := p.Upgrade(io.Discard, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(skill); !strings.HasSuffix(string(got), "local\n") {
		t.Fatal("upgrade overwrote a local edit")
	}
	if err := p.Upgrade(io.Discard, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(skill); string(got) != string(orig) {
		t.Fatal("upgrade --force did not restore the embedded version")
	}
}

// A MODIFIED requirement or section replaces the whole block: lines left over
// from the old text, or scenarios the delta drops, are caught.
func TestMergeIsExact(t *testing.T) {
	p := newProject(t)
	zone := "---\nid: notify\ntldr: Payment reminders before a recurring charge.\n---\n# Notify\n\n## Purpose\nReminds.\n\n## Requirements\n\n### Requirement: Reminder\nThe system SHALL remind.\n\n#### Scenario: before\n- **WHEN** a payment is due\n- **THEN** remind 1 day before\n\n#### Scenario: twice\n- **WHEN** a reminder was sent\n- **THEN** don't send it again\n\n## Gotchas\n- old trap\n"
	write(t, p.Path("zones/notify.md"), zone)
	if _, err := p.WriteIndex(); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	if _, err := p.NewChange("days", false, "", now); err != nil {
		t.Fatal(err)
	}
	c, _ := p.FindChange("days")
	write(t, filepath.Join(c.Dir, "tasks.md"), "- [x] 1.1 three days\n")
	block := "The system SHALL remind.\n\n#### Scenario: before\n- **WHEN** a payment is due\n- **THEN** remind 3 days before\n"
	write(t, filepath.Join(c.Dir, "deltas/notify.md"), "## MODIFIED Requirements\n### Requirement: Reminder\n"+block+"\n## MODIFIED Sections\n### Section: Gotchas\n- new trap\n")

	c, _ = p.FindChange("days")
	issues, _ := p.ArchiveCheck(c)
	if w := messages(issues, SevWarn); !strings.Contains(w, `MODIFIED "Reminder" drops scenario "twice"`) {
		t.Errorf("dropped scenario not warned:\n%s", w)
	}

	merged := func(req, gotchas string) []Issue {
		t.Helper()
		head := zone[:strings.Index(zone, "### Requirement:")]
		write(t, p.Path("zones/notify.md"), head+"### Requirement: Reminder\n"+req+"\n## Gotchas\n"+gotchas)
		c, _ := p.FindChange("days")
		issues, err := p.ArchiveFinish(c, now, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		return issues
	}
	stale := strings.Replace(block, "- **THEN** remind 3 days before\n", "- **THEN** remind 3 days before\n- **THEN** remind 1 day before\n", 1)
	if e := messages(merged(stale, "- new trap\n"), SevError); !strings.Contains(e, `"- **THEN** remind 1 day before" is not in the delta`) {
		t.Errorf("stale line not caught:\n%s", e)
	}
	if e := messages(merged(block, "- new trap\n- old trap\n"), SevError); !strings.Contains(e, `"- old trap" is not in the delta`) {
		t.Errorf("stale section line not caught:\n%s", e)
	}
	if e := messages(merged(block, ""), SevError); !strings.Contains(e, `missing "- new trap"`) {
		t.Errorf("missing section line not caught:\n%s", e)
	}
	// Blank lines, comments and spacing don't matter.
	loose := strings.ReplaceAll(block, "\n", "\n\n") + "<!-- note -->\n"
	if issues := merged(loose, "-   new trap\n"); CountErrors(issues) > 0 {
		t.Fatalf("faithful merge rejected:\n%s", messages(issues, SevError))
	}
}
