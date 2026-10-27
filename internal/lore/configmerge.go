package lore

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"lore/internal/assets"
)

var (
	topKeyRe    = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):`)
	nestedKeyRe = regexp.MustCompile(`^\s+([A-Za-z_][A-Za-z0-9_]*):`)
)

// tmplBlock is a top-level key of the lore.yaml template with the comment
// lines right above it and its indented children.
type tmplBlock struct {
	key      string
	lines    []string          // comments + key line + children
	children map[string]string // nested key → its line
	order    []string
}

func parseTemplateBlocks(src string) []tmplBlock {
	var blocks []tmplBlock
	var pending []string // comment lines waiting for the next key
	for _, l := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
		switch {
		case topKeyRe.MatchString(l):
			b := tmplBlock{key: topKeyRe.FindStringSubmatch(l)[1], children: map[string]string{}}
			b.lines = append(append(b.lines, pending...), l)
			pending = nil
			blocks = append(blocks, b)
		case strings.TrimSpace(l) == "":
			pending = nil
		case strings.HasPrefix(l, "#"):
			pending = append(pending, l)
		case len(blocks) > 0:
			b := &blocks[len(blocks)-1]
			b.lines = append(b.lines, l)
			if m := nestedKeyRe.FindStringSubmatch(l); m != nil {
				b.children[m[1]] = l
				b.order = append(b.order, m[1])
			}
		}
	}
	return blocks
}

// mergeConfig adds keys of the template that the project's lore.yaml lacks,
// keeping everything already there byte for byte. It returns the new text and
// the added keys.
// ADR 2026-10-27-upgrade-merges-config-as-text: insert lines, never re-encode lore.yaml through YAML.
func mergeConfig(have, tmpl string) (string, []string, error) {
	var current map[string]any
	if err := yaml.Unmarshal([]byte(have), &current); err != nil {
		return "", nil, err
	}
	if current == nil {
		current = map[string]any{}
	}
	lines := strings.Split(strings.TrimRight(have, "\n"), "\n")
	var added []string
	prev := "" // the previous template key present in the file

	for _, b := range parseTemplateBlocks(tmpl) {
		val, ok := current[b.key]
		if !ok {
			// Place the block right after its template neighbour; a block with
			// children or comments gets a blank line before it.
			at := len(lines)
			if prev != "" {
				if end := keyBlockEnd(lines, prev); end >= 0 {
					at = end
				}
			}
			block := b.lines
			if len(b.lines) > 1 || at == len(lines) {
				block = append([]string{""}, b.lines...)
			}
			lines = append(lines[:at], append(append([]string{}, block...), lines[at:]...)...)
			added = append(added, b.key)
			prev = b.key
			continue
		}
		prev = b.key
		sub, isMap := val.(map[string]any)
		if !isMap || len(b.order) == 0 {
			continue
		}
		var missing []string
		for _, k := range b.order {
			if _, ok := sub[k]; !ok {
				missing = append(missing, b.children[k])
				added = append(added, b.key+"."+k)
			}
		}
		if len(missing) == 0 {
			continue
		}
		at := keyBlockEnd(lines, b.key)
		if at < 0 {
			continue
		}
		lines = append(lines[:at], append(missing, lines[at:]...)...)
	}
	if len(added) == 0 {
		return have, nil, nil
	}
	out := strings.Join(lines, "\n") + "\n"
	var check map[string]any
	if err := yaml.Unmarshal([]byte(out), &check); err != nil {
		return "", nil, fmt.Errorf("merged lore.yaml would be invalid (%v) — add %s by hand", err, strings.Join(added, ", "))
	}
	return out, added, nil
}

// keyBlockEnd is the index after the last indented child line of a top-level key.
func keyBlockEnd(lines []string, key string) int {
	start := -1
	for i, l := range lines {
		if m := topKeyRe.FindStringSubmatch(l); m != nil && m[1] == key {
			start = i
			break
		}
	}
	if start < 0 {
		return -1
	}
	end := start + 1
	for i := start + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") {
			continue
		}
		if !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") {
			break
		}
		end = i + 1
	}
	return end
}

// upgradeConfig writes keys that newer lore versions added to lore.yaml.
func (p *Project) upgradeConfig(log Log) error {
	path := filepath.Join(p.Root, ConfigFile)
	have, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, added, err := mergeConfig(string(have), assets.Read("lore.yaml"))
	if err != nil {
		log.f("skipped  %s: %v", ConfigFile, err)
		return nil
	}
	if len(added) == 0 {
		return nil
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	log.f("added    %s: %s", ConfigFile, strings.Join(added, ", "))
	return nil
}
