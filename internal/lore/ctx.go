package lore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ctxHeader  = "[lore] relevant to this text (picked by the decisions model):"
	taskHeader = "[lore] relevant to the next task (picked by the decisions model):"
)

// Ctx selects knowledge for a free text and writes the [lore] block.
func (p *Project) Ctx(w io.Writer, text string, d Decider) error {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sel, err := p.Select(ctx, d, text, p.Cfg.Ctx.BudgetTokens)
	entry := logEntry{Source: "ctx", State: text}
	if sel != nil {
		entry.Scores, entry.Injected = sel.Scores, sel.Keys()
	}
	if err != nil {
		entry.Error = err.Error()
	}
	p.logDecision(entry, started)
	if sel != nil && err == nil {
		if out := sel.Render(ctxHeader); out != "" {
			fmt.Fprint(w, out)
		} else {
			fmt.Fprintln(w, "[lore] nothing relevant found")
		}
	}
	return err
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
	zones, _, err := p.knowledge()
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
