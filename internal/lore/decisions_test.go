package lore

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestSelectTwoStages(t *testing.T) {
	p := knowledgeProject(t)
	sel, err := p.Select(context.Background(), &keywordDecider{}, "change the payment reminder to three days", 2500)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Zones) != 1 || sel.Zones[0].Zone != "finance" {
		t.Fatalf("zones = %+v", sel.Zones)
	}
	keys := strings.Join(sel.Keys(), "\n")
	for _, want := range []string{"req:finance›Payment reminder", "adr:2026-10-01-reminders-enqueue"} {
		if !strings.Contains(keys, want) {
			t.Errorf("missing %s in\n%s", want, keys)
		}
	}
	if strings.Contains(keys, "vault") {
		t.Errorf("vault was not selected in stage 1, its content must not be asked:\n%s", keys)
	}
	if _, ok := sel.Scores["adr:2026-10-01-utc"]; !ok {
		t.Error("global ADRs are always candidates in stage 2")
	}
	out := sel.Render(ctxHeader, nil)
	if !strings.Contains(out, "The system SHALL remind about a recurring payment") || !strings.Contains(out, "zones: finance") {
		t.Fatalf("render:\n%s", out)
	}
	if again := sel.Render(ctxHeader, map[string]bool{}); again != out {
		t.Fatal("render is not deterministic")
	}
}

func TestSelectBudgetDemotesToPointers(t *testing.T) {
	p := knowledgeProject(t)
	sel, err := p.Select(context.Background(), &keywordDecider{}, "payment reminder in finance", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(sel.Content) != 0 || len(sel.Pointers) == 0 {
		t.Fatalf("with no budget everything relevant becomes a pointer: content %d pointers %d", len(sel.Content), len(sel.Pointers))
	}
}

func TestHook(t *testing.T) {
	p := knowledgeProject(t)
	cfg, _ := os.ReadFile(filepath.Join(p.Root, ConfigFile))
	if !strings.Contains(string(cfg), "enabled: true") {
		t.Fatal("hook should be enabled in a fresh lore.yaml")
	}
	d := &keywordDecider{}
	run := func(prompt string) string {
		in, _ := json.Marshal(map[string]string{"prompt": prompt, "session_id": "s1", "cwd": p.Root})
		var out bytes.Buffer
		RunHook(bytes.NewReader(in), &out, func(*Project) (Decider, error) { return d, nil })
		return out.String()
	}
	if out := run("ok"); out != "" || d.calls != 0 {
		t.Fatalf("short prompt should be skipped: %q", out)
	}
	if out := run("/lore-archive voice messages now"); out != "" {
		t.Fatalf("slash commands should be skipped: %q", out)
	}
	first := run("please change the payment reminder to three days")
	var resp struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(first), &resp); err != nil {
		t.Fatalf("hook output is not JSON: %q", first)
	}
	if resp.HookSpecificOutput.HookEventName != "UserPromptSubmit" || !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "Payment reminder") {
		t.Fatalf("hook output: %+v", resp)
	}
	if second := run("please change the payment reminder to three days"); second != "" {
		t.Fatalf("same context injected twice in a session: %q", second)
	}
	logged, _ := os.ReadFile(filepath.Join(p.LocalDir(), "log.jsonl"))
	if strings.Count(string(logged), `"source":"hook"`) != 2 {
		t.Fatalf("log:\n%s", logged)
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

// An item first mentioned as a pointer is not repeated as a pointer, but its
// content still comes when a later prompt scores it high.
func TestHookPointerUpgradesToContent(t *testing.T) {
	p := knowledgeProject(t)
	score := 0.5
	d := deciderFunc(func(state string, qs map[string]Question) map[string]float64 {
		out := map[string]float64{}
		for id, q := range qs {
			switch {
			case strings.Contains(q.Instructions, "«finance»"):
				out[id] = 0.9
			case strings.Contains(q.Instructions, "«finance › Payment reminder»"):
				out[id] = score
			default:
				out[id] = 0.1
			}
		}
		return out
	})
	run := func() string {
		in, _ := json.Marshal(map[string]string{"prompt": "something about money and payments", "session_id": "s2", "cwd": p.Root})
		var out bytes.Buffer
		RunHook(bytes.NewReader(in), &out, func(*Project) (Decider, error) { return d, nil })
		return out.String()
	}
	if out := run(); !strings.Contains(out, "maybe relevant: finance › Payment reminder") {
		t.Fatalf("first prompt should mention the requirement as a pointer: %q", out)
	}
	if out := run(); out != "" {
		t.Fatalf("pointer repeated in the same session: %q", out)
	}
	score = 0.9
	out := run()
	if !strings.Contains(out, "The system SHALL remind about a recurring payment") {
		t.Fatalf("content not injected after the pointer: %q", out)
	}
	if out := run(); out != "" {
		t.Fatalf("content injected twice: %q", out)
	}
}

type deciderFunc func(state string, qs map[string]Question) map[string]float64

func (f deciderFunc) Noul(_ context.Context, state string, qs map[string]Question) (map[string]float64, error) {
	return f(state, qs), nil
}
