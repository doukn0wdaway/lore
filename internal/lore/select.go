package lore

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"lore/internal/assets"
)

func builtinPrompt(name string) string { return assets.Read("prompts/" + name + ".txt") }

// Candidate is a unit of knowledge the decisions model can pick for a task.
type Candidate struct {
	Key     string // zone:<id> | adr:<id>
	Kind    string // zone | decision
	Zone    string
	Name    string
	Text    string // what the model sees
	Content string // what the agent gets when it is selected
}

// Scored is a candidate with the model's probability that it relates.
type Scored struct {
	Candidate
	P float64
}

// Selection is the knowledge picked for one state: decisions as content or
// pointers, zones printed whole or named.
type Selection struct {
	Decisions []Scored // ADRs shown as content
	Pointers  []Scored // ADRs mentioned by name ("maybe relevant")
	Zones     []Scored // zones printed whole
	Named     []Scored // zones above the threshold that are only named
	Scores    map[string]float64
}

// Keys lists what the selection shows.
func (s *Selection) Keys() []string {
	var out []string
	for _, group := range [][]Scored{s.Decisions, s.Pointers, s.Zones, s.Named} {
		for _, c := range group {
			out = append(out, c.Key)
		}
	}
	return out
}

// zoneModelSections are the sections the model reads together with a zone's
// tldr: they say what the zone guarantees and how it borders on other zones.
var zoneModelSections = []string{"Invariants", "Boundaries", "Contracts"}

const (
	zoneTextLimit = 3000
	adrTextLimit  = 700
)

// knowledge collects every candidate: each zone and each accepted ADR once.
func (p *Project) knowledge() (zones, decisions []Candidate, err error) {
	zs, err := p.LoadZones()
	if err != nil {
		return nil, nil, err
	}
	adrs, err := p.LoadADRs()
	if err != nil {
		return nil, nil, err
	}
	for _, z := range zs {
		zones = append(zones, zoneCandidate(z))
	}
	for _, a := range adrs {
		if a.Status == "accepted" {
			decisions = append(decisions, adrCandidate(a))
		}
	}
	return zones, decisions, nil
}

func zoneCandidate(z *Zone) Candidate {
	id := z.Key()
	text := []string{strings.TrimSpace(z.TLDR)}
	for _, title := range zoneModelSections {
		if body, _, ok := z.doc.section(2, title); ok && !isBlank(body) {
			text = append(text, title+": "+compact(body))
		}
	}
	return Candidate{
		Key: "zone:" + id, Kind: "zone", Zone: id, Name: id,
		Text:    truncate(strings.Join(text, "\n"), zoneTextLimit),
		Content: strings.TrimSpace(commentRe.ReplaceAllString(strings.Join(z.doc.lines, "\n"), "")),
	}
}

func adrCandidate(a *ADR) Candidate {
	rules := strings.Join(a.Rules, "\n")
	var content strings.Builder
	fmt.Fprintf(&content, "ADR %s\n%s", a.Key(), strings.TrimSpace(a.TLDR))
	for _, r := range a.Rules {
		content.WriteString("\n" + r)
	}
	fmt.Fprintf(&content, "\n→ %s", a.File)
	return Candidate{
		Key: "adr:" + a.Key(), Kind: "decision", Name: a.Key(),
		Text:    truncate(strings.TrimSpace(a.TLDR+"\n"+rules), adrTextLimit),
		Content: content.String(),
	}
}

// compact collapses a markdown block into one line for the model.
func compact(s string) string {
	s = commentRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(strings.NewReplacer("**", "", "#### ", "", "`", "").Replace(s)), " ")
}

// Select scores every zone and every accepted ADR in one go. Decisions at or
// above include_content are shown as content with no budget, the ones above
// include_pointer by name. Zones at or above the zones threshold are printed
// whole, highest first, while they fit budgetTokens; the rest are named
// (budgetTokens 0 names them all).
// ADR 2026-10-27-agent-pulls-knowledge-with-ctx: every ADR scored, decisions apart from zones, zones whole.
func (p *Project) Select(ctx context.Context, d Decider, state string, budgetTokens int) (*Selection, error) {
	zones, decisions, err := p.knowledge()
	if err != nil {
		return nil, err
	}
	sel := &Selection{Scores: map[string]float64{}}
	t := p.Cfg.Thresholds

	scored, err := p.ask(ctx, d, state, append(zones, decisions...))
	if err != nil {
		return sel, err
	}
	budget := budgetTokens * 4 // ~4 characters per token
	for _, s := range scored {
		sel.Scores[s.Key] = s.P
		switch {
		case s.Kind == "decision" && s.P >= t.IncludeContent:
			sel.Decisions = append(sel.Decisions, s)
		case s.Kind == "decision" && s.P >= t.IncludePointer:
			sel.Pointers = append(sel.Pointers, s)
		case s.Kind == "zone" && s.P >= t.Zones && len(s.Content) <= budget:
			budget -= len(s.Content)
			sel.Zones = append(sel.Zones, s)
		case s.Kind == "zone" && s.P >= t.Zones:
			sel.Named = append(sel.Named, s)
		}
	}
	return sel, nil
}

// ask scores candidates, highest first.
func (p *Project) ask(ctx context.Context, d Decider, state string, cs []Candidate) ([]Scored, error) {
	if len(cs) == 0 {
		return nil, nil
	}
	qs := map[string]Question{}
	ids := map[string]Candidate{}
	for i, c := range cs {
		id := fmt.Sprintf("q%03d", i)
		ids[id] = c
		qs[id] = p.relatesQuestion(c.Kind, c.Name, c.Text)
	}
	answers, err := d.Noul(ctx, state, qs)
	var out []Scored
	for id, prob := range answers {
		out = append(out, Scored{ids[id], prob})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].P != out[j].P {
			return out[i].P > out[j].P
		}
		return out[i].Key < out[j].Key
	})
	return out, err
}

// Render formats a selection as the [lore] block for the agent: decisions
// first, then whole zones, then the names of the rest.
func (s *Selection) Render(header string) string {
	var b strings.Builder
	for _, c := range s.Decisions {
		head, rest, _ := strings.Cut(c.Content, "\n")
		fmt.Fprintf(&b, "\n%s  (%.2f)\n", head, c.P)
		if rest != "" {
			fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(rest, "\n", "\n  "))
		}
	}
	if ptr := names(s.Pointers, func(c Scored) string { return c.Name }); len(ptr) > 0 {
		fmt.Fprintf(&b, "\nmaybe relevant: %s\n", strings.Join(ptr, ", "))
	}
	for _, z := range s.Zones {
		fmt.Fprintf(&b, "\n=== zone %s  (%.2f) ===\n%s\n", z.Zone, z.P, z.Content)
	}
	if zs := names(s.Named, func(c Scored) string { return c.Zone }); len(zs) > 0 {
		fmt.Fprintf(&b, "\nzones: %s — open with `lore show <zone>`\n", strings.Join(zs, ", "))
	}
	if b.Len() == 0 {
		return ""
	}
	return header + "\n" + b.String()
}

// names lists items as "name (p)".
func names(cs []Scored, name func(Scored) string) []string {
	var out []string
	for _, c := range cs {
		out = append(out, fmt.Sprintf("%s (%.2f)", name(c), c.P))
	}
	return out
}
