package lore

import (
	"fmt"
	"regexp"
	"strings"
)

// Delta is changes/<name>/deltas/<zone>.md: OpenSpec-style ADDED / MODIFIED /
// REMOVED / RENAMED requirements plus MODIFIED Sections for free-form zone
// sections. A delta with frontmatter is a whole new zone doc instead.
type Delta struct {
	Zone     string // file name without .md = target zone id
	File     string // relative to project root
	NewZone  bool   // has frontmatter: the full doc of a zone that doesn't exist yet
	Added    []named
	Modified []named
	Removed  []named
	Renamed  []rename
	Sections []named // MODIFIED Sections, by `## <title>` of the zone
	Problems []named // format problems, name = message
	doc      *mdDoc
	src      string
}

type rename struct {
	from, to string
	line     int
}

const (
	opAdded    = "added requirements"
	opModified = "modified requirements"
	opRemoved  = "removed requirements"
	opRenamed  = "renamed requirements"
	opSections = "modified sections"
	sectionPfx = "Section:"
)

var renameRe = regexp.MustCompile("^\\s*[-*]?\\s*(FROM|TO):\\s*`?\\s*(?:###\\s*)?(?:Requirement:)?\\s*(.+?)\\s*`?\\s*$")

func parseDelta(zone, file, src string) *Delta {
	d := &Delta{Zone: zone, File: file, src: src, doc: parseDoc(src)}
	if d.doc.hasFront {
		d.NewZone = true
		return d
	}
	op := ""
	for _, h := range d.doc.headings {
		line := d.doc.fileLine(h.line)
		switch h.level {
		case 1:
			continue
		case 2:
			op = strings.ToLower(h.title)
			switch op {
			case opAdded, opModified, opRemoved, opRenamed, opSections:
			default:
				d.problem(line, "unknown delta section %q (expected ADDED/MODIFIED/REMOVED/RENAMED Requirements or MODIFIED Sections)", h.title)
				op = ""
			}
		case 3:
			switch op {
			case opAdded, opModified, opRemoved:
				if !strings.HasPrefix(h.title, requirementPrefix) {
					d.problem(line, "expected `### Requirement: <name>` under %s", op)
					continue
				}
				n := named{strings.TrimSpace(strings.TrimPrefix(h.title, requirementPrefix)), line}
				switch op {
				case opAdded:
					d.Added = append(d.Added, n)
				case opModified:
					d.Modified = append(d.Modified, n)
				default:
					d.Removed = append(d.Removed, n)
				}
			case opSections:
				if !strings.HasPrefix(h.title, sectionPfx) {
					d.problem(line, "expected `### Section: <title>` under MODIFIED Sections")
					continue
				}
				d.Sections = append(d.Sections, named{strings.TrimSpace(strings.TrimPrefix(h.title, sectionPfx)), line})
			}
		}
	}
	d.parseRenames()
	return d
}

func (d *Delta) problem(line int, format string, a ...any) {
	d.Problems = append(d.Problems, named{fmt.Sprintf(format, a...), line})
}

func (d *Delta) parseRenames() {
	text, line, ok := d.doc.section(2, "RENAMED Requirements")
	if !ok {
		return
	}
	var from *named
	for i, row := range strings.Split(text, "\n") {
		m := renameRe.FindStringSubmatch(row)
		if m == nil {
			continue
		}
		n := named{m[2], line + 1 + i}
		if m[1] == "FROM" {
			from = &n
			continue
		}
		if from == nil {
			d.problem(n.line, "TO without a preceding FROM")
			continue
		}
		d.Renamed = append(d.Renamed, rename{from.name, n.name, from.line})
		from = nil
	}
	if from != nil {
		d.problem(from.line, "FROM without a TO")
	}
}

// touched lists requirement names this delta changes in the existing zone,
// used to warn about two active changes editing the same requirement.
func (d *Delta) touched() []string {
	var out []string
	for _, n := range d.Modified {
		out = append(out, n.name)
	}
	for _, n := range d.Removed {
		out = append(out, n.name)
	}
	for _, r := range d.Renamed {
		out = append(out, r.from)
	}
	return out
}

// summary is one line per operation, for `lore change status`.
func (d *Delta) summary() []string {
	if d.NewZone {
		return []string{"NEW ZONE"}
	}
	var out []string
	add := func(op string, ns []named) {
		for _, n := range ns {
			out = append(out, fmt.Sprintf("%s %q", op, n.name))
		}
	}
	add("ADDED", d.Added)
	add("MODIFIED", d.Modified)
	add("REMOVED", d.Removed)
	for _, r := range d.Renamed {
		out = append(out, fmt.Sprintf("RENAMED %q → %q", r.from, r.to))
	}
	add("MODIFIED section", d.Sections)
	return out
}

// checkBefore reports what doesn't fit the current zone text before merging.
func (d *Delta) checkBefore(zone *Zone, l *linter) {
	if d.NewZone {
		if zone != nil {
			l.errf(d.File, 1, "has frontmatter (a new zone) but zone %q already exists — write ADDED/MODIFIED sections instead", d.Zone)
		}
		return
	}
	if zone == nil {
		l.errf(d.File, 0, "zone %q does not exist; a delta for a new zone must be the whole zone doc with frontmatter", d.Zone)
		return
	}
	reqs := map[string]bool{}
	for _, r := range zone.doc.requirements() {
		reqs[r.name] = true
	}
	for _, n := range d.Added {
		if reqs[n.name] {
			l.warnf(d.File, n.line, "ADDED %q already exists in the zone — it will be treated as MODIFIED", n.name)
		}
	}
	for _, n := range d.Modified {
		if reqs[n.name] {
			d.checkDropped(zone, n, l)
		}
	}
	for _, n := range append(append([]named{}, d.Modified...), d.Removed...) {
		if !reqs[n.name] {
			l.errf(d.File, n.line, "requirement %q not found in %s", n.name, zone.File)
		}
	}
	for _, r := range d.Renamed {
		if !reqs[r.from] {
			l.errf(d.File, r.line, "RENAMED FROM %q not found in %s", r.from, zone.File)
		}
		if reqs[r.to] {
			l.errf(d.File, r.line, "RENAMED TO %q already exists in %s", r.to, zone.File)
		}
	}
	for _, s := range d.Sections {
		if _, _, ok := zone.doc.section(2, s.name); !ok {
			l.errf(d.File, s.line, "section `## %s` not found in %s", s.name, zone.File)
		}
	}
}

// checkMerged verifies the delta landed in the zone after the agent merged it.
func (d *Delta) checkMerged(zone *Zone, l *linter) {
	if zone == nil {
		l.errf(d.File, 0, "not merged: %s does not exist", "zones/"+d.Zone+".md")
		return
	}
	if d.NewZone {
		return
	}
	reqs := map[string]bool{}
	for _, r := range zone.doc.requirements() {
		reqs[r.name] = true
	}
	for _, n := range append(append([]named{}, d.Added...), d.Modified...) {
		if !reqs[n.name] {
			l.errf(d.File, n.line, "not merged: requirement %q missing in %s", n.name, zone.File)
			continue
		}
		d.checkLines(zone, 3, requirementPrefix+" "+n.name, l)
	}
	for _, n := range d.Removed {
		if reqs[n.name] {
			l.errf(d.File, n.line, "not merged: REMOVED requirement %q still in %s", n.name, zone.File)
		}
	}
	for _, r := range d.Renamed {
		if reqs[r.from] || !reqs[r.to] {
			l.errf(d.File, r.line, "not merged: rename %q → %q not applied in %s", r.from, r.to, zone.File)
		}
	}
	for _, s := range d.Sections {
		if _, _, ok := zone.doc.section(2, s.name); !ok {
			l.errf(d.File, s.line, "not merged: section `## %s` missing in %s", s.name, zone.File)
			continue
		}
		d.checkLines(zone, 2, s.name, l)
	}
}

// checkDropped warns about scenarios the zone's requirement has but the
// MODIFIED block omits: a MODIFIED block carries the whole requirement, so
// they disappear on merge.
func (d *Delta) checkDropped(zone *Zone, n named, l *linter) {
	title := requirementPrefix + " " + n.name
	want, _, _ := d.doc.section(3, title)
	have, _, _ := zone.doc.section(3, title)
	kept := map[string]bool{}
	for _, s := range scenarios(want) {
		kept[s] = true
	}
	for _, s := range scenarios(have) {
		if !kept[s] {
			l.warnf(d.File, n.line, "MODIFIED %q drops scenario %q — a MODIFIED block carries the whole requirement: copy the scenario if it still holds", n.name, s)
		}
	}
}

// scenarios lists `#### Scenario: <name>` headings of a requirement block.
func scenarios(block string) []string {
	var out []string
	for _, h := range parseDoc(block).headings {
		if h.level == 4 && strings.HasPrefix(h.title, scenarioPrefix) {
			out = append(out, strings.TrimSpace(strings.TrimPrefix(h.title, scenarioPrefix)))
		}
	}
	return out
}

// checkLines verifies that the zone's block equals the delta's block: an
// ADDED or MODIFIED requirement and a MODIFIED section replace the whole
// block, so after a faithful merge the zone holds exactly the delta's lines —
// nothing missing, nothing left over. Comments, blank lines and whitespace
// differences are ignored.
// ADR 2026-10-27-modified-carries-whole-requirement: the zone block must equal the delta, don't relax to "delta lines are present".
func (d *Delta) checkLines(zone *Zone, level int, title string, l *linter) {
	deltaTitle := title
	if level == 2 { // MODIFIED Sections: `### Section: X` in the delta, `## X` in the zone
		deltaTitle = sectionPfx + " " + title
	}
	wantText, wantLine, _ := d.doc.section(3, deltaTitle)
	haveText, haveLine, _ := zone.doc.section(level, title)
	want, have := blockLines(wantText, wantLine), blockLines(haveText, haveLine)
	for i := 0; i < max(len(want), len(have)); i++ {
		switch {
		case i >= len(have):
			l.errf(d.File, want[i].line, "not merged into %s › %s: missing %q", zone.File, title, want[i].name)
		case i >= len(want):
			l.errf(zone.File, have[i].line, "not merged from %s › %s: %q is not in the delta — the block must equal the delta", d.File, title, have[i].name)
		case want[i].name != have[i].name:
			l.errf(d.File, want[i].line, "not merged into %s › %s: expected %q, found %q (%s:%d)", zone.File, title, want[i].name, have[i].name, zone.File, have[i].line)
		default:
			continue
		}
		return
	}
}

// blockLines are the non-blank lines of a block with comments removed and
// whitespace normalised; line is the file line of the block's heading.
func blockLines(text string, line int) []named {
	text = commentRe.ReplaceAllStringFunc(text, func(c string) string { return strings.Repeat("\n", strings.Count(c, "\n")) })
	var out []named
	for i, row := range strings.Split(text, "\n") {
		if n := normLine(row); n != "" {
			out = append(out, named{n, line + 1 + i})
		}
	}
	return out
}

func normLine(s string) string { return strings.Join(strings.Fields(s), " ") }
