package lore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ctxHeader      = "[lore] relevant to this prompt (picked by the decisions model):"
	taskHeader     = "[lore] relevant to the next task (picked by the decisions model):"
	maxHookContext = 9500 // Claude Code caps additionalContext at 10k characters
)

// Ctx selects knowledge for a free text and writes the [lore] block.
func (p *Project) Ctx(w io.Writer, text string, d Decider) error {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sel, err := p.Select(ctx, d, text, p.Cfg.Hook.BudgetTokens)
	entry := logEntry{Source: "ctx", State: text}
	if sel != nil {
		entry.Scores, entry.Injected = sel.Scores, sel.Keys()
	}
	if err != nil {
		entry.Error = err.Error()
	}
	p.logDecision(entry, started)
	if sel != nil {
		if out := sel.Render(ctxHeader, nil); out != "" {
			fmt.Fprint(w, out)
		} else {
			fmt.Fprintln(w, "[lore] nothing relevant found")
		}
	}
	return err
}

// hookInput is what Claude Code sends to a UserPromptSubmit hook.
type hookInput struct {
	Prompt     string `json:"prompt"`
	PromptText string `json:"prompt_text"`
	SessionID  string `json:"session_id"`
	CWD        string `json:"cwd"`
}

// RunHook handles a UserPromptSubmit hook: it never fails the prompt. Any
// problem is logged and the hook prints nothing.
func RunHook(stdin io.Reader, w io.Writer, newDecider func(*Project) (Decider, error)) {
	var in hookInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		return
	}
	prompt := in.Prompt
	if prompt == "" {
		prompt = in.PromptText
	}
	dir := in.CWD
	if dir == "" {
		dir = "."
	}
	p, err := FindProject(dir)
	if err != nil || !p.Cfg.Hook.Enabled {
		return
	}
	if len(strings.Fields(prompt)) < p.Cfg.Hook.MinWords || strings.HasPrefix(strings.TrimSpace(prompt), "/") {
		return
	}
	started := time.Now()
	entry := logEntry{Source: "hook", State: prompt}
	defer func() { p.logDecision(entry, started) }()

	d, err := newDecider(p)
	if err != nil {
		entry.Error = err.Error()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.Cfg.Hook.TimeoutMS)*time.Millisecond)
	defer cancel()
	sel, err := p.Select(ctx, d, prompt, p.Cfg.Hook.BudgetTokens)
	if err != nil {
		entry.Error = err.Error()
	}
	if sel == nil {
		return
	}
	entry.Scores = sel.Scores

	seen := p.sessionSeen(in.SessionID)
	out := sel.Render(ctxHeader, seen)
	if out == "" {
		return
	}
	if len(out) > maxHookContext {
		out = out[:maxHookContext] + "\n…"
	}
	for _, k := range sel.SessionKeys() {
		if !seen[k] {
			entry.Injected = append(entry.Injected, k)
		}
	}
	p.sessionAdd(in.SessionID, entry.Injected)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"hookSpecificOutput": map[string]string{
			"hookEventName":     "UserPromptSubmit",
			"additionalContext": out,
		},
	})
}

var sessionIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (p *Project) sessionFile(id string) string {
	if !sessionIDRe.MatchString(id) {
		return ""
	}
	return filepath.Join(p.LocalDir(), "sessions", id+".json")
}

// sessionSeen returns keys already injected in this session.
func (p *Project) sessionSeen(id string) map[string]bool {
	seen := map[string]bool{}
	path := p.sessionFile(id)
	if path == "" {
		return seen
	}
	var keys []string
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &keys)
	}
	for _, k := range keys {
		seen[k] = true
	}
	return seen
}

func (p *Project) sessionAdd(id string, keys []string) {
	path := p.sessionFile(id)
	if path == "" || len(keys) == 0 {
		return
	}
	seen := p.sessionSeen(id)
	for _, k := range keys {
		seen[k] = true
	}
	all := make([]string, 0, len(seen))
	for k := range seen {
		all = append(all, k)
	}
	sort.Strings(all)
	b, _ := json.Marshal(all)
	if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
		_ = os.WriteFile(path, b, 0o644)
	}
}

// --- facts at the gate ---

// FactReport is what the decisions model says about one fact of a change.
type FactReport struct {
	Fact           string   `json:"fact"`
	Zones          []Scored `json:"zones"`
	Contradictions []Scored `json:"contradictions"`
}

// facts are the bullets of `## What changes` in proposal.md.
func (c *Change) facts() []string {
	b, err := os.ReadFile(filepath.Join(c.Dir, "proposal.md"))
	if err != nil {
		return nil
	}
	text, _, _ := parseDoc(string(b)).section(2, "What changes")
	var out []string
	for _, row := range strings.Split(commentRe.ReplaceAllString(text, ""), "\n") {
		row = strings.TrimSpace(row)
		if strings.HasPrefix(row, "- ") || strings.HasPrefix(row, "* ") {
			out = append(out, strings.TrimSpace(row[2:]))
		}
	}
	return out
}

// CheckFacts asks, for every fact of the change, which zones it touches and
// which accepted ADRs it may contradict. Contradictions are questions for the
// owner, not verdicts.
func (p *Project) CheckFacts(c *Change, d Decider) ([]FactReport, error) {
	facts := c.facts()
	if len(facts) == 0 {
		return nil, fmt.Errorf("%s: no facts — write them as bullets under `## What changes` in proposal.md", c.Name)
	}
	zones, _, _, err := p.knowledge()
	if err != nil {
		return nil, err
	}
	adrs, err := p.LoadADRs()
	if err != nil {
		return nil, err
	}
	var decisions []*ADR
	for _, a := range adrs {
		if a.Status == "accepted" {
			decisions = append(decisions, a)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var reports []FactReport
	for _, fact := range facts {
		started := time.Now()
		r := FactReport{Fact: fact}
		scored, err := p.ask(ctx, d, fact, zones)
		if err != nil {
			return nil, err
		}
		scores := map[string]float64{}
		for _, s := range scored {
			scores[s.Key] = s.P
			if s.P >= p.Cfg.Thresholds.Zones {
				r.Zones = append(r.Zones, s)
			}
		}
		qs := map[string]Question{}
		byQ := map[string]*ADR{}
		for i, a := range decisions {
			id := fmt.Sprintf("a%03d", i)
			byQ[id] = a
			qs[id] = p.contradictsQuestion(a.Key(), strings.TrimSpace(a.TLDR+"\n"+strings.Join(a.Rules, "\n")))
		}
		if len(qs) > 0 {
			answers, err := d.Noul(ctx, fact, qs)
			if err != nil {
				return nil, err
			}
			for id, prob := range answers {
				a := byQ[id]
				scores["contradicts:"+a.Key()] = prob
				if prob >= p.Cfg.Thresholds.Contradiction {
					r.Contradictions = append(r.Contradictions, Scored{adrCandidate(a), prob})
				}
			}
			sort.Slice(r.Contradictions, func(i, j int) bool { return r.Contradictions[i].P > r.Contradictions[j].P })
		}
		p.logDecision(logEntry{Source: "check", State: fact, Scores: scores}, started)
		reports = append(reports, r)
	}
	return reports, nil
}

// WriteFactReports prints the reports and the zones touched by facts but
// missing from the change's deltas.
func WriteFactReports(w io.Writer, c *Change, reports []FactReport) {
	deltaZones := map[string]bool{}
	for _, d := range c.Deltas {
		deltaZones[d.Zone] = true
	}
	missing := map[string]bool{}
	conflicts := 0
	for _, r := range reports {
		fmt.Fprintf(w, "\nFACT  %s\n", r.Fact)
		var zs []string
		for _, z := range r.Zones {
			zs = append(zs, fmt.Sprintf("%s (%.2f)", z.Zone, z.P))
			if !deltaZones[z.Zone] {
				missing[z.Zone] = true
			}
		}
		fmt.Fprintf(w, "  zones: %s\n", listOrNone(zs))
		for _, a := range r.Contradictions {
			conflicts++
			fmt.Fprintf(w, "  MAY CONTRADICT %s (%.2f)\n", a.Name, a.P)
			fmt.Fprintf(w, "    %s\n", strings.ReplaceAll(strings.TrimPrefix(a.Content, "ADR "+a.Name+"\n"), "\n", "\n    "))
		}
	}
	fmt.Fprintln(w)
	if len(missing) > 0 {
		var zs []string
		for z := range missing {
			zs = append(zs, z)
		}
		sort.Strings(zs)
		fmt.Fprintf(w, "zones touched by facts but without a delta: %s — add a delta or explain why not\n", strings.Join(zs, ", "))
	}
	if conflicts > 0 {
		fmt.Fprintf(w, "%d possible contradiction(s): ask the owner about each — new decision (supersede) or bug in the code?\n", conflicts)
	} else {
		fmt.Fprintln(w, "no contradictions with accepted ADRs found")
	}
}
