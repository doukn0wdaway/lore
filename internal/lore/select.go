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
	Key     string // zone:<id> | req:<zone>›<name> | sec:<zone>›<title> | adr:<id>
	Kind    string // zone | requirement | section | decision
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

// Selection is the result of the two-stage choice for one state.
type Selection struct {
	Zones    []Scored // stage 1, above the zones threshold
	Content  []Scored // inject the content
	Pointers []Scored // mention only
	Scores   map[string]float64
}

// Keys lists what the selection injects (zones, content and pointers).
func (s *Selection) Keys() []string {
	var out []string
	for _, group := range [][]Scored{s.Zones, s.Content, s.Pointers} {
		for _, c := range group {
			out = append(out, c.Key)
		}
	}
	return out
}

// sections that are not offered separately: Purpose is in the zone's tldr,
// requirements are offered one by one, entry points are code paths.
var skipSections = map[string]bool{"purpose": true, "requirements": true, "entry points": true}

const (
	modelTextLimit = 700
	contentLimit   = 1800
)

// knowledge collects every candidate: zones, their requirements and free
// sections, and accepted ADRs.
func (p *Project) knowledge() (zones []Candidate, inZone map[string][]Candidate, global []Candidate, err error) {
	zs, err := p.LoadZones()
	if err != nil {
		return nil, nil, nil, err
	}
	adrs, err := p.LoadADRs()
	if err != nil {
		return nil, nil, nil, err
	}
	inZone = map[string][]Candidate{}
	for _, z := range zs {
		id := z.Key()
		zones = append(zones, Candidate{
			Key: "zone:" + id, Kind: "zone", Zone: id, Name: id,
			Text:    z.TLDR,
			Content: fmt.Sprintf("zone %s — %s (open: `lore show %s`)", id, z.TLDR, id),
		})
		for _, r := range z.doc.requirements() {
			body, _, _ := z.doc.section(3, requirementPrefix+" "+r.name)
			inZone[id] = append(inZone[id], Candidate{
				Key: "req:" + id + "›" + r.name, Kind: "requirement", Zone: id, Name: id + " › " + r.name,
				Text:    truncate(compact(body), modelTextLimit),
				Content: id + " › " + r.name + "\n" + truncate(strings.TrimSpace(commentRe.ReplaceAllString(body, "")), contentLimit),
			})
		}
		for _, h := range z.doc.headings {
			if h.level != 2 || skipSections[strings.ToLower(h.title)] {
				continue
			}
			body, _, _ := z.doc.section(2, h.title)
			if isBlank(body) {
				continue
			}
			inZone[id] = append(inZone[id], Candidate{
				Key: "sec:" + id + "›" + h.title, Kind: "section", Zone: id, Name: id + " › " + h.title,
				Text:    truncate(compact(body), modelTextLimit),
				Content: id + " › " + h.title + "\n" + truncate(strings.TrimSpace(commentRe.ReplaceAllString(body, "")), contentLimit),
			})
		}
	}
	for _, a := range adrs {
		if a.Status != "accepted" {
			continue
		}
		c := adrCandidate(a)
		isGlobal := false
		for _, z := range a.Zones {
			if z == GlobalZone {
				isGlobal = true
				continue
			}
			c.Zone = z
			inZone[z] = append(inZone[z], c)
		}
		if isGlobal {
			c.Zone = GlobalZone
			global = append(global, c)
		}
	}
	return zones, inZone, global, nil
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
		Text:    strings.TrimSpace(a.TLDR + "\n" + rules),
		Content: content.String(),
	}
}

// compact collapses a markdown block into one line for the model.
func compact(s string) string {
	s = commentRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(strings.NewReplacer("**", "", "#### ", "", "`", "").Replace(s)), " ")
}

// Select runs the two-stage choice: zones first, then the requirements,
// sections and ADRs of the chosen zones plus global ADRs. On a stage-2
// failure (timeout) the zones of stage 1 are still returned.
// ADR 2026-10-27-inject-content-per-requirement: requirement-level content, never whole zones; no optimisation before the pilot.
func (p *Project) Select(ctx context.Context, d Decider, state string, budgetTokens int) (*Selection, error) {
	zones, inZone, global, err := p.knowledge()
	if err != nil {
		return nil, err
	}
	sel := &Selection{Scores: map[string]float64{}}
	t := p.Cfg.Thresholds

	stage1, err := p.ask(ctx, d, state, zones)
	if err != nil {
		return sel, err
	}
	for _, s := range stage1 {
		sel.Scores[s.Key] = s.P
		if s.P >= t.Zones {
			sel.Zones = append(sel.Zones, s)
		}
	}

	var pool []Candidate
	seen := map[string]bool{}
	add := func(cs []Candidate) {
		for _, c := range cs {
			if !seen[c.Key] {
				seen[c.Key] = true
				pool = append(pool, c)
			}
		}
	}
	for _, z := range sel.Zones {
		add(inZone[z.Zone])
	}
	add(global)

	stage2, err := p.ask(ctx, d, state, pool)
	if err != nil {
		return sel, err
	}
	budget := budgetTokens * 4 // ~4 characters per token
	for _, s := range stage2 {
		sel.Scores[s.Key] = s.P
		switch {
		case s.P >= t.IncludeContent && len(s.Content) <= budget:
			budget -= len(s.Content)
			sel.Content = append(sel.Content, s)
		case s.P >= t.IncludePointer || s.P >= t.IncludeContent:
			sel.Pointers = append(sel.Pointers, s)
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

// pointerPfx marks a key that was only mentioned by name in a session: the
// pointer is not repeated, but the item may still be injected as content later.
const pointerPfx = "ptr:"

// SessionKeys lists what rendering this selection shows, for per-session
// dedup: content and zones by key, pointers by pointerPfx+key.
func (s *Selection) SessionKeys() []string {
	var out []string
	for _, c := range s.Content {
		out = append(out, c.Key)
	}
	for _, c := range s.Pointers {
		out = append(out, pointerPfx+c.Key)
	}
	for _, z := range s.Zones {
		out = append(out, z.Key)
	}
	return out
}

// Render formats a selection as the [lore] block for the agent. skip holds
// session keys already shown earlier in the session: content and zones are
// never repeated, a pointer is not repeated as a pointer but its content can
// still come once it scores high enough.
func (s *Selection) Render(header string, skip map[string]bool) string {
	var b strings.Builder
	for _, c := range s.Content {
		if skip[c.Key] {
			continue
		}
		head, rest, _ := strings.Cut(c.Content, "\n")
		fmt.Fprintf(&b, "\n%s  (%.2f)\n", head, c.P)
		if rest != "" {
			fmt.Fprintf(&b, "  %s\n", strings.ReplaceAll(rest, "\n", "\n  "))
		}
	}
	var ptr []string
	for _, c := range s.Pointers {
		if !skip[c.Key] && !skip[pointerPfx+c.Key] {
			ptr = append(ptr, fmt.Sprintf("%s (%.2f)", c.Name, c.P))
		}
	}
	if len(ptr) > 0 {
		fmt.Fprintf(&b, "\nmaybe relevant: %s\n", strings.Join(ptr, ", "))
	}
	var zs []string
	for _, z := range s.Zones {
		if !skip[z.Key] {
			zs = append(zs, fmt.Sprintf("%s (%.2f)", z.Zone, z.P))
		}
	}
	if len(zs) > 0 {
		fmt.Fprintf(&b, "\nzones: %s — open with `lore show <zone>`\n", strings.Join(zs, ", "))
	}
	if b.Len() == 0 {
		return ""
	}
	return header + "\n" + b.String()
}
