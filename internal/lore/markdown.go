package lore

import (
	"regexp"
	"strings"
)

// mdDoc is a markdown file split into frontmatter and body, with headings
// indexed. Fenced code blocks are skipped when looking for headings.
type mdDoc struct {
	front     string
	hasFront  bool
	lines     []string // body lines
	bodyStart int      // 1-based line number of the first body line in the file
	headings  []heading
}

type heading struct {
	level int
	title string
	line  int // index into lines
}

var (
	fenceRe   = regexp.MustCompile("^\\s*(```|~~~)")
	commentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
)

func parseDoc(src string) *mdDoc {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(src, "\n")
	d := &mdDoc{lines: lines, bodyStart: 1}
	if len(lines) > 0 && strings.TrimRight(lines[0], " \t") == "---" {
		for j := 1; j < len(lines); j++ {
			if strings.TrimRight(lines[j], " \t") == "---" {
				d.front = strings.Join(lines[1:j], "\n")
				d.hasFront = true
				d.lines = lines[j+1:]
				d.bodyStart = j + 2
				break
			}
		}
	}
	d.headings = findHeadings(d.lines)
	return d
}

func findHeadings(lines []string) []heading {
	var hs []heading
	var fence string
	for i, l := range lines {
		if m := fenceRe.FindStringSubmatch(l); m != nil {
			switch {
			case fence == "":
				fence = m[1]
			case strings.HasPrefix(strings.TrimSpace(l), fence):
				fence = ""
			}
			continue
		}
		if fence != "" || !strings.HasPrefix(l, "#") {
			continue
		}
		n := 0
		for n < len(l) && l[n] == '#' {
			n++
		}
		if n > 6 || n == len(l) || (l[n] != ' ' && l[n] != '\t') {
			continue
		}
		hs = append(hs, heading{level: n, title: strings.TrimSpace(l[n:]), line: i})
	}
	return hs
}

// section returns the text under the first heading with the given level and
// title (case-insensitive), up to the next heading of the same or higher level.
func (d *mdDoc) section(level int, title string) (text string, line int, ok bool) {
	for i, h := range d.headings {
		if h.level != level || !strings.EqualFold(h.title, title) {
			continue
		}
		end := len(d.lines)
		for _, next := range d.headings[i+1:] {
			if next.level <= level {
				end = next.line
				break
			}
		}
		return strings.Join(d.lines[h.line+1:end], "\n"), d.bodyStart + h.line, true
	}
	return "", 0, false
}

// title is the first level-1 heading.
func (d *mdDoc) title() string {
	for _, h := range d.headings {
		if h.level == 1 {
			return h.title
		}
	}
	return ""
}

// fileLine converts a body line index to a 1-based file line number.
func (d *mdDoc) fileLine(i int) int { return d.bodyStart + i }

const (
	requirementPrefix = "Requirement:"
	scenarioPrefix    = "Scenario:"
)

type named struct {
	name string
	line int // 1-based file line
}

// requirements lists `### Requirement: <name>` headings.
func (d *mdDoc) requirements() []named {
	var out []named
	for _, h := range d.headings {
		if h.level == 3 && strings.HasPrefix(h.title, requirementPrefix) {
			out = append(out, named{strings.TrimSpace(strings.TrimPrefix(h.title, requirementPrefix)), d.fileLine(h.line)})
		}
	}
	return out
}

// isBlank reports whether text has no content besides whitespace and HTML comments.
func isBlank(text string) bool {
	return strings.TrimSpace(commentRe.ReplaceAllString(text, "")) == ""
}
