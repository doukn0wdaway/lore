package lore

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Severity string

const (
	SevError Severity = "error"
	SevWarn  Severity = "warning"
)

type Issue struct {
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line,omitempty"`
	Msg      string   `json:"message"`
}

func (i Issue) String() string {
	loc := i.File
	if i.Line > 0 {
		loc = fmt.Sprintf("%s:%d", i.File, i.Line)
	}
	return fmt.Sprintf("%-7s %s: %s", i.Severity, loc, i.Msg)
}

type linter struct {
	p      *Project
	issues []Issue
}

func (l *linter) errf(file string, line int, format string, a ...any) {
	l.issues = append(l.issues, Issue{SevError, file, line, fmt.Sprintf(format, a...)})
}

func (l *linter) warnf(file string, line int, format string, a ...any) {
	l.issues = append(l.issues, Issue{SevWarn, file, line, fmt.Sprintf(format, a...)})
}

const (
	zoneMaxLines = 300
	zoneMinLines = 50
	tldrMinWords = 5
)

var adrStatuses = map[string]bool{"proposed": true, "accepted": true, "superseded": true}

// Lint checks the whole lore folder. Errors make `lore lint` exit non-zero.
func (p *Project) Lint() ([]Issue, error) {
	zones, err := p.LoadZones()
	if err != nil {
		return nil, err
	}
	changes, err := p.LoadChanges()
	if err != nil {
		return nil, err
	}
	adrs, err := p.LoadADRs()
	if err != nil {
		return nil, err
	}
	newZones := map[string]map[string]bool{} // change → zones it creates
	for _, c := range changes {
		adrs = append(adrs, c.Drafts...)
		newZones[c.Name] = c.newZones()
	}
	l := &linter{p: p}
	if err := p.Cfg.checkKeyEnv(); err != nil {
		l.errf(ConfigFile, 0, "%s", strings.TrimPrefix(err.Error(), ConfigFile+": "))
	}
	if line := p.OldHookLine(); line > 0 {
		l.warnf(ConfigFile, line, "%s", oldHookMsg)
	}
	l.zones(zones)
	l.adrs(adrs, func(a *ADR, z string) bool { return FindZone(zones, z) != nil || newZones[a.Change][z] })
	l.changes(changes)
	l.index(zones)
	l.product(zones)
	if err := l.anchors(adrs); err != nil {
		return nil, err
	}
	sort.SliceStable(l.issues, func(i, j int) bool {
		a, b := l.issues[i], l.issues[j]
		if a.Severity != b.Severity {
			return a.Severity == SevError
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return l.issues, nil
}

func (l *linter) zones(zones []*Zone) {
	seen := map[string]string{}
	for _, z := range zones {
		f := z.File
		if !z.doc.hasFront {
			l.errf(f, 1, "no frontmatter: need id and tldr")
		} else if z.FrontErr != nil {
			l.errf(f, 1, "frontmatter: %v%s", z.FrontErr, yamlHint(z.FrontErr))
		}
		switch {
		case z.ID == "":
			l.errf(f, 1, "missing id")
		case z.ID != z.Name:
			l.errf(f, 1, "id %q does not match the file name %q", z.ID, z.Name)
		case z.ID == GlobalZone:
			l.errf(f, 1, "%q is reserved for project-wide ADRs", GlobalZone)
		case !zoneIDRe.MatchString(z.ID):
			l.errf(f, 1, "id %q: use lowercase kebab-case", z.ID)
		}
		if prev, ok := seen[z.Key()]; ok {
			l.errf(f, 1, "duplicate zone id %q (also %s)", z.Key(), prev)
		}
		seen[z.Key()] = f
		l.tldr(f, z.TLDR)

		if text, line, ok := z.doc.section(2, "Purpose"); !ok {
			l.errf(f, 0, "missing ## Purpose")
		} else if isBlank(text) {
			l.errf(f, line, "## Purpose is empty")
		}

		names := map[string]int{}
		for _, r := range z.doc.requirements() {
			if r.name == "" {
				l.errf(f, r.line, "requirement without a name")
				continue
			}
			if prev, ok := names[r.name]; ok {
				l.errf(f, r.line, "duplicate requirement %q (also line %d)", r.name, prev)
			}
			names[r.name] = r.line
		}

		l.entryPoints(z)

		n := len(strings.Split(strings.TrimRight(z.Src, "\n"), "\n"))
		if n > zoneMaxLines {
			l.warnf(f, 0, "%d lines — consider splitting the zone (guide: 100–300)", n)
		} else if n < zoneMinLines {
			l.warnf(f, 0, "%d lines — maybe a section of a neighbouring zone (guide: 100–300)", n)
		}
	}
}

// yamlHint explains the most common frontmatter mistake: an unquoted value with ": ".
func yamlHint(err error) string {
	if err != nil && strings.Contains(err.Error(), "mapping values are not allowed") {
		return ` — a value containing ": " must be quoted, e.g. tldr: "Notes: search, read, edit"`
	}
	return ""
}

func (l *linter) tldr(file, tldr string) {
	t := strings.TrimSpace(tldr)
	switch {
	case t == "":
		l.errf(file, 1, "missing tldr")
	case len(strings.Fields(t)) < tldrMinWords:
		l.warnf(file, 1, "tldr is too short to match tasks against: %q", t)
	}
}

var backtickRe = regexp.MustCompile("`([^`]+)`")

// entryPoints checks that file paths listed under ## Entry points exist.
// Symbols (`pkg.Func()`), globs and anything that doesn't look like a path are skipped.
func (l *linter) entryPoints(z *Zone) {
	text, line, ok := z.doc.section(2, "Entry points")
	if !ok {
		return
	}
	for i, row := range strings.Split(text, "\n") {
		row = strings.TrimSpace(row)
		if !strings.HasPrefix(row, "-") && !strings.HasPrefix(row, "*") {
			continue
		}
		m := backtickRe.FindStringSubmatch(row)
		if m == nil {
			continue
		}
		path := m[1]
		if strings.ContainsAny(path, "()* ") || !strings.Contains(path, "/") && !strings.Contains(path, ".") {
			continue
		}
		if i := strings.IndexAny(path, ":#"); i > 0 {
			path = path[:i]
		}
		if _, err := os.Stat(filepath.Join(l.p.Root, path)); err != nil {
			l.errf(z.File, line+1+i, "entry point %q does not exist", path)
		}
	}
}

func (l *linter) adrs(adrs []*ADR, zoneOK func(a *ADR, zone string) bool) {
	byID := map[string]*ADR{}
	for _, a := range adrs {
		if prev, ok := byID[a.Key()]; ok {
			l.errf(a.File, 1, "duplicate ADR id %q (also %s)", a.Key(), prev.File)
		}
		byID[a.Key()] = a
	}
	for _, a := range adrs {
		f := a.File
		if !a.doc.hasFront {
			l.errf(f, 1, "no frontmatter: need id, status, zones, tldr")
		} else if a.FrontErr != nil {
			l.errf(f, 1, "frontmatter: %v%s", a.FrontErr, yamlHint(a.FrontErr))
		}
		switch {
		case a.ID == "":
			l.errf(f, 1, "missing id")
		case a.ID != a.Name:
			l.errf(f, 1, "id %q does not match the file name %q", a.ID, a.Name)
		case !adrIDRe.MatchString(a.ID):
			l.errf(f, 1, "id %q: expected YYYY-MM-DD-slug", a.ID)
		}
		if !adrStatuses[a.Status] {
			l.errf(f, 1, "status %q: expected proposed, accepted or superseded", a.Status)
		}
		if len(a.Zones) == 0 {
			l.errf(f, 1, "missing zones (zone ids or %s)", GlobalZone)
		}
		for _, z := range a.Zones {
			if z != GlobalZone && !zoneOK(a, z) {
				l.errf(f, 1, "zones: no zone %q", z)
			}
		}
		l.tldr(f, a.TLDR)

		for _, s := range []string{"Context", "Decision", "Why"} {
			text, line, ok := a.doc.section(2, s)
			if !ok {
				l.errf(f, 0, "missing ## %s", s)
			} else if isBlank(text) {
				l.errf(f, line, "## %s is empty", s)
			}
		}

		for _, s := range a.Supersedes {
			if old, ok := byID[s]; !ok || old.Change != "" {
				l.errf(f, 1, "supersedes unknown ADR %q", s)
			}
		}
		if a.Change != "" && a.Status == "superseded" {
			l.errf(f, 1, "a draft in a change can't be superseded")
		}
		if a.Status == "superseded" && a.SupersededBy == "" {
			l.errf(f, 1, "status superseded without superseded_by")
		}
		if a.SupersededBy != "" {
			if a.Status != "superseded" {
				l.errf(f, 1, "superseded_by is set but status is %q", a.Status)
			}
			if _, ok := byID[a.SupersededBy]; !ok {
				l.errf(f, 1, "superseded_by unknown ADR %q", a.SupersededBy)
			}
		}
		if a.Inferred {
			l.warnf(f, 1, "inferred: the reason behind this decision is unknown")
		}
	}
}

func (l *linter) changes(changes []*Change) {
	type owner struct{ change, file string }
	touched := map[string]owner{} // zone › requirement → first change touching it
	for _, c := range changes {
		rel := l.p.Rel(c.Dir)
		if !slugRe.MatchString(c.Name) {
			l.errf(rel, 0, "change name %q: use lowercase kebab-case", c.Name)
		}
		switch {
		case c.MetaErr != nil:
			l.errf(rel+"/"+metaFile, 0, "%v", c.MetaErr)
		case c.Meta.Mode != ModeSpec && c.Meta.Mode != ModeCode:
			l.errf(rel+"/"+metaFile, 0, "mode %q: expected spec or code", c.Meta.Mode)
		}
		for _, d := range c.Deltas {
			if !zoneIDRe.MatchString(d.Zone) || d.Zone == GlobalZone {
				l.errf(d.File, 0, "file name must be a zone id")
			}
			for _, pr := range d.Problems {
				l.errf(d.File, pr.line, "%s", pr.name)
			}
			if d.NewZone {
				z := &Zone{File: d.File, Name: d.Zone, Src: d.src, doc: d.doc}
				if z.FrontErr = yamlUnmarshal(d.doc.front, z); z.FrontErr != nil {
					l.errf(d.File, 1, "frontmatter: %v%s", z.FrontErr, yamlHint(z.FrontErr))
				}
				if z.ID != d.Zone {
					l.errf(d.File, 1, "new zone: id %q does not match the file name %q", z.ID, d.Zone)
				}
				l.tldr(d.File, z.TLDR)
				if text, line, ok := d.doc.section(2, "Purpose"); !ok {
					l.errf(d.File, 0, "new zone: missing ## Purpose")
				} else if isBlank(text) {
					l.errf(d.File, line, "## Purpose is empty")
				}
			}
			for _, r := range d.touched() {
				key := d.Zone + " › " + r
				if prev, ok := touched[key]; ok && prev.change != c.Name {
					l.warnf(d.File, 0, "%q is also changed by %s (%s) — merge carefully", key, prev.change, prev.file)
					continue
				}
				touched[key] = owner{c.Name, d.File}
			}
		}
	}
}

func (l *linter) index(zones []*Zone) {
	path := l.p.Path("INDEX.md")
	have, err := os.ReadFile(path)
	if err != nil {
		l.errf(l.p.Rel(path), 0, "missing — run `lore index`")
		return
	}
	if string(have) != RenderIndex(zones) {
		l.errf(l.p.Rel(path), 0, "out of date — run `lore index`")
	}
}

var scenarioZonesRe = regexp.MustCompile(`^\s*zones:\s*(.+)$`)

func (l *linter) product(zones []*Zone) {
	path := l.p.Path("product.md")
	b, err := os.ReadFile(path)
	if err != nil {
		l.errf(l.p.Rel(path), 0, "missing")
		return
	}
	// Ignore the commented example in the template.
	src := commentRe.ReplaceAllStringFunc(string(b), func(c string) string {
		return strings.Repeat("\n", strings.Count(c, "\n"))
	})
	for i, row := range strings.Split(src, "\n") {
		m := scenarioZonesRe.FindStringSubmatch(row)
		if m == nil {
			continue
		}
		for _, z := range strings.Split(m[1], ",") {
			z = strings.TrimSpace(z)
			if z != "" && FindZone(zones, z) == nil {
				l.errf(l.p.Rel(path), i+1, "scenario refers to unknown zone %q", z)
			}
		}
	}
}

var anchorRe = regexp.MustCompile(`\bADR\s+(\d{4}-\d{2}-\d{2}-[a-z0-9]+(?:-[a-z0-9]+)*)`)

// anchors checks `ADR <id>` comments in code: the ADR must exist and be in force.
func (l *linter) anchors(adrs []*ADR) error {
	byID := map[string]*ADR{}
	for _, a := range adrs {
		byID[a.Key()] = a
	}
	files, err := l.codeFiles()
	if err != nil {
		return err
	}
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(l.p.Root, rel))
		if err != nil || len(b) > 1<<20 || bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0 {
			continue
		}
		sc := bufio.NewScanner(bytes.NewReader(b))
		sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for n := 1; sc.Scan(); n++ {
			for _, m := range anchorRe.FindAllStringSubmatch(sc.Text(), -1) {
				a, ok := byID[m[1]]
				switch {
				case !ok:
					l.errf(rel, n, "anchor refers to unknown ADR %q", m[1])
				case a.Status == "superseded":
					l.errf(rel, n, "anchor refers to superseded ADR %q (now %s)", m[1], a.SupersededBy)
				}
			}
		}
	}
	return nil
}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true, ".direnv": true}

// codeFiles lists project files outside the docs folder, honouring .gitignore when in git.
func (l *linter) codeFiles() ([]string, error) {
	docs := filepath.ToSlash(l.p.Cfg.Root) + "/"
	keep := func(rel string) bool {
		return !strings.HasPrefix(rel, docs) && rel != "AGENTS.md" && rel != "CLAUDE.md" && !l.p.anchorsIgnored(rel)
	}
	cmd := exec.Command("git", "ls-files", "-co", "--exclude-standard")
	cmd.Dir = l.p.Root
	if out, err := cmd.Output(); err == nil {
		var files []string
		for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if f != "" && keep(f) {
				files = append(files, f)
			}
		}
		return files, nil
	}
	var files []string
	err := filepath.WalkDir(l.p.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			if rel := l.p.Rel(path); keep(rel) {
				files = append(files, rel)
			}
		}
		return nil
	})
	return files, err
}

// anchorsIgnored reports whether rel matches anchors_ignore: the exact path,
// a folder above it, or a glob; like .gitignore, a pattern without a slash
// matches the file name at any depth.
func (p *Project) anchorsIgnored(rel string) bool {
	for _, pat := range p.Cfg.AnchorsIgnore {
		pat = path.Clean(filepath.ToSlash(pat))
		name := rel
		if !strings.Contains(pat, "/") {
			name = path.Base(rel)
		}
		if ok, _ := path.Match(pat, name); ok || strings.HasPrefix(rel, pat+"/") {
			return true
		}
	}
	return false
}

// CountErrors returns the number of error-level issues.
func CountErrors(issues []Issue) int {
	n := 0
	for _, i := range issues {
		if i.Severity == SevError {
			n++
		}
	}
	return n
}
