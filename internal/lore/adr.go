package lore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"lore/internal/assets"
)

// NewADR creates <date>-<slug>.md from the template with status proposed:
// in adr/, or as a draft in changes/<change>/adr/ when change is set.
// supersedes lists ADRs the new one replaces entirely.
func (p *Project) NewADR(slug string, zones []string, title string, supersedes []string, change string, now time.Time) (string, error) {
	if !slugRe.MatchString(slug) {
		return "", fmt.Errorf("slug %q: use lowercase letters, digits and dashes", slug)
	}
	if len(zones) == 0 {
		return "", errors.New("--zones is required: zone ids or global")
	}
	dir := p.Path("adr")
	newZones := map[string]bool{}
	if change != "" {
		c, err := p.FindChange(change)
		if err != nil {
			return "", err
		}
		dir = filepath.Join(c.Dir, "adr")
		newZones = c.newZones()
	}
	known, err := p.LoadZones()
	if err != nil {
		return "", err
	}
	for _, z := range zones {
		if z != GlobalZone && FindZone(known, z) == nil && !newZones[z] {
			return "", fmt.Errorf("no zone %q; zones: %s (or %s)", z, strings.Join(zoneIDs(known), ", "), GlobalZone)
		}
	}
	all, err := p.LoadAllADRs()
	if err != nil {
		return "", err
	}
	for _, s := range supersedes {
		old := findADR(all, s)
		switch {
		case old == nil || old.Change != "":
			return "", fmt.Errorf("--supersedes: no ADR %q in adr/", s)
		case old.Status == "superseded":
			return "", fmt.Errorf("--supersedes: %s is already superseded by %s", s, old.SupersededBy)
		}
	}
	id := now.Format("2006-01-02") + "-" + slug
	if a := findADR(all, id); a != nil {
		return "", fmt.Errorf("%s already exists", a.File)
	}
	path := filepath.Join(dir, id+".md")
	if title == "" {
		title = strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
	}

	tmpl := template.Must(template.New("adr").Parse(assets.Read("templates/adr.md")))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"ID": id, "Zones": strings.Join(zones, ", "), "Title": title,
		"Supersedes": strings.Join(supersedes, ", "),
	}); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return p.Rel(path), os.WriteFile(path, buf.Bytes(), 0o644)
}

func findADR(adrs []*ADR, id string) *ADR {
	for _, a := range adrs {
		if a.Key() == id {
			return a
		}
	}
	return nil
}

// AcceptADR records the owner's approval: proposed → accepted. For an ADR in
// adr/ the ADRs it supersedes are marked superseded right away; for a draft in
// a change that happens when the change is archived. It refuses while the ADR
// has lint errors. Returns the files it changed.
func (p *Project) AcceptADR(id string) ([]string, error) {
	all, err := p.LoadAllADRs()
	if err != nil {
		return nil, err
	}
	a := findADR(all, id)
	if a == nil {
		return nil, fmt.Errorf("no ADR %q", id)
	}
	if a.Status != "proposed" {
		return nil, fmt.Errorf("%s is %s, only proposed ADRs can be accepted", id, a.Status)
	}
	issues, err := p.Lint()
	if err != nil {
		return nil, err
	}
	var blocking []string
	for _, i := range issues {
		if i.Severity == SevError && i.File == a.File {
			blocking = append(blocking, "  "+i.String())
		}
	}
	if len(blocking) > 0 {
		return nil, fmt.Errorf("fix %s first:\n%s", a.File, strings.Join(blocking, "\n"))
	}

	var olds []*ADR
	// ADR 2026-10-27-status-only-via-accept: a draft in a change supersedes nothing here, only at archive.
	if a.Change == "" {
		for _, s := range a.Supersedes {
			old := findADR(all, s)
			if old == nil || old.Change != "" {
				return nil, fmt.Errorf("supersedes unknown ADR %q", s)
			}
			if old.Status == "superseded" && old.SupersededBy != id {
				return nil, fmt.Errorf("%s is already superseded by %s", s, old.SupersededBy)
			}
			olds = append(olds, old)
		}
	}
	if err := p.setFields(a, [2]string{"status", "accepted"}); err != nil {
		return nil, err
	}
	changed := []string{a.File}
	for _, old := range olds {
		if err := p.supersede(old, id); err != nil {
			return nil, err
		}
		changed = append(changed, old.File)
	}
	return changed, nil
}

func (p *Project) supersede(old *ADR, by string) error {
	return p.setFields(old, [2]string{"status", "superseded"}, [2]string{"superseded_by", by})
}

func (p *Project) setFields(a *ADR, fields ...[2]string) error {
	path := filepath.Join(p.Root, a.File)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := string(b)
	for _, f := range fields {
		if src, err = setFrontField(src, f[0], f[1]); err != nil {
			return fmt.Errorf("%s: %w", a.File, err)
		}
	}
	return os.WriteFile(path, []byte(src), 0o644)
}

// setFrontField replaces a top-level `key: value` line in the frontmatter,
// or adds it before the closing delimiter. Other lines stay as they are.
func setFrontField(src, key, value string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("no frontmatter")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			lines = append(lines[:i], append([]string{key + ": " + value}, lines[i:]...)...)
			return strings.Join(lines, "\n"), nil
		}
		if strings.HasPrefix(lines[i], key+":") {
			lines[i] = key + ": " + value
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", errors.New("unterminated frontmatter")
}

// ListADRs filters ADRs in adr/ by zone (own or global) and status; empty filters match all.
func (p *Project) ListADRs(zone, status string) ([]*ADR, error) {
	adrs, err := p.LoadADRs()
	if err != nil {
		return nil, err
	}
	var out []*ADR
	for _, a := range adrs {
		if zone != "" && !a.AppliesTo(zone) {
			continue
		}
		if status != "" && a.Status != status {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}
