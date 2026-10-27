package lore

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// keywordDecider says "yes" (0.9) when the question mentions a word from the
// state, "no" (0.1) otherwise.
type keywordDecider struct{ calls int }

func (k *keywordDecider) Noul(_ context.Context, state string, qs map[string]Question) (map[string]float64, error) {
	k.calls++
	out := map[string]float64{}
	for id, q := range qs {
		out[id] = 0.1
		for _, w := range strings.Fields(strings.ToLower(state)) {
			if len(w) > 3 && strings.Contains(strings.ToLower(q.Instructions), w) {
				out[id] = 0.9
			}
		}
	}
	return out, nil
}

func knowledgeProject(t *testing.T) *Project {
	t.Helper()
	p := newProject(t)
	write(t, p.Path("zones/finance.md"), `---
id: finance
tldr: Expenses, recurring payments and payment reminders in the hledger journal.
---
# Finance

## Purpose
Money.

## Requirements

### Requirement: Payment reminder
The system SHALL remind about a recurring payment before the charge.
#### Scenario: due soon
- **WHEN** a payment is due tomorrow
- **THEN** remind

## Invariants
- Reminders never go out on weekends.

## Gotchas
- hledger amounts are strings.
`)
	write(t, p.Path("zones/vault.md"), "---\nid: vault\ntldr: Obsidian notes: read, create, edit and move notes and folders.\n---\n# Vault\n\n## Purpose\nNotes.\n\n### Requirement: Soft delete\nDeleted notes go to the inbox.\n")
	if _, err := p.WriteIndex(); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for _, a := range []struct{ slug, zone, tldr string }{
		{"reminders-enqueue", "finance", "Payment reminders are enqueued, never sent from the cron."},
		{"utc", "global", "All stored times are in UTC."},
	} {
		path, err := p.NewADR(a.slug, []string{a.zone}, "", nil, "", now)
		if err != nil {
			t.Fatal(err)
		}
		fillADR(t, p, path)
		b, _ := os.ReadFile(filepath.Join(p.Root, path))
		write(t, filepath.Join(p.Root, path), strings.Replace(string(b), "Files are stored by content hash, never by name.", a.tldr, 1))
		if _, err := p.AcceptADR(now.Format("2006-01-02") + "-" + a.slug); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

// remindersADR is the fixture ADR id; built apart from "ADR " so lint doesn't read it as a code anchor.
const remindersADR = "2026-10-01-reminders-enqueue"

func TestSelectZonesAndDecisions(t *testing.T) {
	p := knowledgeProject(t)
	sel, err := p.Select(context.Background(), &keywordDecider{}, "change the payment reminder to three days", 2500)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Zones) != 1 || sel.Zones[0].Zone != "finance" || len(sel.Named) != 0 {
		t.Fatalf("zones = %+v named = %+v", sel.Zones, sel.Named)
	}
	if len(sel.Decisions) != 1 || sel.Decisions[0].Name != "2026-10-01-reminders-enqueue" {
		t.Fatalf("decisions = %+v", sel.Decisions)
	}
	// every zone and every accepted ADR is asked, nothing else
	want := []string{"zone:finance", "zone:vault", "adr:2026-10-01-reminders-enqueue", "adr:2026-10-01-utc"}
	if len(sel.Scores) != len(want) {
		t.Fatalf("scores = %v", sel.Scores)
	}
	for _, k := range want {
		if _, ok := sel.Scores[k]; !ok {
			t.Errorf("%s not scored", k)
		}
	}
	out := sel.Render(ctxHeader)
	for _, want := range []string{
		"ADR " + remindersADR,
		"=== zone finance",
		"### Requirement: Payment reminder", // the zone is printed whole
		"- hledger amounts are strings.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render misses %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "ADR "+remindersADR) > strings.Index(out, "=== zone finance") {
		t.Errorf("decisions must come before zones:\n%s", out)
	}
	if strings.Contains(out, "Obsidian") {
		t.Errorf("vault was not chosen:\n%s", out)
	}
	if again := sel.Render(ctxHeader); again != out {
		t.Fatal("render is not deterministic")
	}
}

// A zone is judged by its invariants, boundaries and contracts too, not only by its tldr.
func TestSelectZoneThroughInvariant(t *testing.T) {
	p := knowledgeProject(t)
	sel, err := p.Select(context.Background(), &keywordDecider{}, "send everything on weekends", 2500)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Zones) != 1 || sel.Zones[0].Zone != "finance" {
		t.Fatalf("zones = %+v", sel.Zones)
	}
}

func TestSelectADROfSeveralZonesAskedOnce(t *testing.T) {
	p := knowledgeProject(t)
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	path, err := p.NewADR("shared", []string{"finance", "vault"}, "", nil, "", now)
	if err != nil {
		t.Fatal(err)
	}
	fillADR(t, p, path)
	if _, err := p.AcceptADR("2026-10-02-shared"); err != nil {
		t.Fatal(err)
	}
	asked := map[string]int{}
	d := deciderFunc(func(_ string, qs map[string]Question) map[string]float64 {
		out := map[string]float64{}
		for id, q := range qs {
			asked[q.Instructions]++
			out[id] = 0.1
		}
		return out
	})
	if _, err := p.Select(context.Background(), d, "anything", 2500); err != nil {
		t.Fatal(err)
	}
	for q, n := range asked {
		if n != 1 {
			t.Errorf("asked %d times: %s", n, q)
		}
	}
	if len(asked) != 5 {
		t.Errorf("questions = %d, want 2 zones + 3 ADRs", len(asked))
	}
}

func TestSelectBudgetNamesZones(t *testing.T) {
	p := knowledgeProject(t)
	sel, err := p.Select(context.Background(), &keywordDecider{}, "payment reminder in finance", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Zones) != 0 || len(sel.Named) != 1 || len(sel.Decisions) != 1 {
		t.Fatalf("with no budget zones are named, decisions still shown: zones %d named %d decisions %d", len(sel.Zones), len(sel.Named), len(sel.Decisions))
	}
	if out := sel.Render(ctxHeader); !strings.Contains(out, "zones: finance (0.90) — open with `lore show <zone>`") || !strings.Contains(out, "ADR "+remindersADR) {
		t.Fatalf("render:\n%s", out)
	}
}

func TestCtx(t *testing.T) {
	p := knowledgeProject(t)
	var out strings.Builder
	if err := p.Ctx(&out, "change the payment reminder to three days", &keywordDecider{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "=== zone finance") || !strings.Contains(out.String(), "ADR "+remindersADR) {
		t.Fatalf("ctx:\n%s", out.String())
	}
	logged, _ := os.ReadFile(filepath.Join(p.LocalDir(), "log.jsonl"))
	if !strings.Contains(string(logged), `"source":"ctx"`) {
		t.Fatalf("log:\n%s", logged)
	}
	if _, err := os.Stat(filepath.Join(p.LocalDir(), "sessions")); err == nil {
		t.Fatal("no per-session state any more")
	}
}

func TestCtxFailureIsNotNothingFound(t *testing.T) {
	p := knowledgeProject(t)
	var out strings.Builder
	err := p.Ctx(&out, "anything", failingDecider{})
	if err == nil || strings.Contains(out.String(), "nothing relevant") {
		t.Fatalf("err %v, out %q", err, out.String())
	}
}

type failingDecider struct{}

func (failingDecider) Noul(context.Context, string, map[string]Question) (map[string]float64, error) {
	return nil, errors.New("timeout")
}

func TestCtxBudgetAndOldHookBlock(t *testing.T) {
	p := knowledgeProject(t)
	cfgPath := filepath.Join(p.Root, ConfigFile)
	cfg, _ := os.ReadFile(cfgPath)
	if p.Cfg.Ctx.BudgetTokens != 8000 || !strings.Contains(string(cfg), "ctx:\n  budget_tokens: 8000") || strings.Contains(string(cfg), "hook:") {
		t.Fatalf("budget %d, lore.yaml:\n%s", p.Cfg.Ctx.BudgetTokens, cfg)
	}
	hookWarning := func() *Issue {
		issues, err := p.Lint()
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range issues {
			if strings.Contains(i.Msg, "`hook:` is no longer used") {
				return &i
			}
		}
		return nil
	}
	if w := hookWarning(); w != nil {
		t.Fatalf("fresh project warns: %v", w)
	}
	old := strings.Replace(string(cfg), "ctx:\n  budget_tokens: 8000", "hook:\n  enabled: true\n  budget_tokens: 4000", 1)
	write(t, cfgPath, old)
	p, err := LoadProject(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Cfg.Ctx.BudgetTokens != 8000 {
		t.Fatalf("hook.budget_tokens must not be read: %d", p.Cfg.Ctx.BudgetTokens)
	}
	w := hookWarning()
	if w == nil || w.Severity != SevWarn || w.Line != p.OldHookLine() || w.Line == 0 {
		t.Fatalf("warning = %+v", w)
	}
}

func TestStatusNext(t *testing.T) {
	p := knowledgeProject(t)
	if _, err := p.NewChange("remind", false, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	c, _ := p.FindChange("remind")
	write(t, filepath.Join(c.Dir, "tasks.md"), "## 1. Reminders\n- [x] 1.1 Storage\n- [ ] 1.2 Change the payment reminder to three days\n")
	write(t, filepath.Join(c.Dir, "deltas/finance.md"), "## REMOVED Requirements\n### Requirement: Payment reminder\n")
	c, _ = p.FindChange("remind")
	status := func(onlyNext bool) string {
		var out strings.Builder
		if err := p.Status(&out, c, &keywordDecider{}, onlyNext); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	full := status(false)
	for _, want := range []string{"1.2 Change the payment reminder", "DELTAS IN THIS CHANGE", "DECISIONS FOR THESE ZONES", "--- ", "ADR " + remindersADR, "zones: finance (0.90) — open with `lore show <zone>`"} {
		if !strings.Contains(full, want) {
			t.Errorf("full status misses %q:\n%s", want, full)
		}
	}
	if strings.Contains(full, "=== zone") {
		t.Errorf("status names zones, never prints them:\n%s", full)
	}
	next := status(true)
	for _, want := range []string{"1.2 Change the payment reminder", "ADR " + remindersADR, "zones: finance"} {
		if !strings.Contains(next, want) {
			t.Errorf("--next misses %q:\n%s", want, next)
		}
	}
	for _, not := range []string{"DELTAS IN THIS CHANGE", "DECISIONS FOR THESE ZONES", "--- ", "=== zone"} {
		if strings.Contains(next, not) {
			t.Errorf("--next prints %q:\n%s", not, next)
		}
	}
	write(t, filepath.Join(c.Dir, "tasks.md"), "- [x] 1.1 Storage\n")
	c, _ = p.FindChange("remind")
	if out := status(true); !strings.Contains(out, "lore change archive remind") {
		t.Fatalf("--next with all done:\n%s", out)
	}
}

func TestCheckFacts(t *testing.T) {
	p := knowledgeProject(t)
	if _, err := p.NewChange("remind", true, "", time.Now()); err == nil {
		t.Fatal("code-first without git must fail")
	}
	if _, err := p.NewChange("remind", false, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	c, _ := p.FindChange("remind")
	write(t, filepath.Join(c.Dir, "proposal.md"), "# Remind\n\n## What changes\n- Payment reminders are sent directly from the cron.\n")
	reports, err := p.CheckFacts(c, &keywordDecider{})
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || len(reports[0].Contradictions) == 0 || reports[0].Contradictions[0].Name != "2026-10-01-reminders-enqueue" {
		t.Fatalf("reports %+v", reports)
	}
	var out strings.Builder
	WriteFactReports(&out, c, reports)
	for _, want := range []string{"MAY CONTRADICT 2026-10-01-reminders-enqueue", "without a delta: finance"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report missing %q:\n%s", want, out.String())
		}
	}
}

func TestOpenRouterClient(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			w.WriteHeader(401)
			io.WriteString(w, `{"error":{"message":"API key expired.","code":401}}`)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		answers := map[string]any{}
		for k := range got["questions"].(map[string]any) {
			answers[k] = map[string]any{"type": "noul", "noul": 0.42}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"answers": answers, "usage": map[string]any{"input_tokens": 10}})
	}))
	defer srv.Close()

	qs := map[string]Question{}
	for i := 0; i < maxQuestions+3; i++ {
		qs[string(rune('a'+i/26))+string(rune('a'+i%26))] = Question{Type: "noul", Instructions: "x"}
	}
	o := &openRouter{endpoint: srv.URL, model: "m", key: "k", provider: []string{"Perplexity"}, client: srv.Client()}
	res, err := o.Noul(context.Background(), "state", qs)
	if err != nil || len(res) != len(qs) || res["aa"] != 0.42 {
		t.Fatalf("res %d err %v", len(res), err)
	}
	if got["model"] != "m" || got["provider"] == nil {
		t.Fatalf("request body %v", got)
	}
	o.key = "bad"
	if _, err := o.Noul(context.Background(), "s", map[string]Question{"q": {}}); err == nil || !strings.Contains(err.Error(), "API key expired") {
		t.Fatalf("error not surfaced: %v", err)
	}
}

func TestDeciderKeySources(t *testing.T) {
	p := newProject(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(p.Cfg.DecisionsAPIKeyEnv, "")
	write(t, filepath.Join(p.Root, ".env"), "LORE_OPENROUTER_API_KEY=app-env-must-be-ignored\n")
	if _, err := p.NewDecider(); err == nil {
		t.Fatal("the project's .env belongs to the app and must not be read")
	}
	p.Cfg.DecisionsAPIKeyEnv = "sk-or-v1-0123456789abcdef0123456789abcdef"
	if _, err := p.NewDecider(); err == nil || strings.Contains(err.Error(), "0123456789") {
		t.Fatalf("a key in decisions_api_key_env must be refused without echoing it: %v", err)
	}
	p.Cfg.DecisionsAPIKeyEnv = "LORE_OPENROUTER_API_KEY"
	write(t, GlobalEnvFile(), "# lore\nexport LORE_OPENROUTER_API_KEY=\"global\"\n")
	if d, err := p.NewDecider(); err != nil || d.(*openRouter).key != "global" {
		t.Fatalf("global key: %v", err)
	}
	write(t, p.EnvFile(), "LORE_OPENROUTER_API_KEY=project\n")
	if d, _ := p.NewDecider(); d.(*openRouter).key != "project" {
		t.Fatal("lore/.env of the project should win over ~/.config/lore/.env")
	}
	t.Setenv("LORE_OPENROUTER_API_KEY", "env")
	if d, _ := p.NewDecider(); d.(*openRouter).key != "env" {
		t.Fatal("the environment should win")
	}
}

type deciderFunc func(state string, qs map[string]Question) map[string]float64

func (f deciderFunc) Noul(_ context.Context, state string, qs map[string]Question) (map[string]float64, error) {
	return f(state, qs), nil
}
