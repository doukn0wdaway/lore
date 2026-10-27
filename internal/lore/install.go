package lore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"lore/internal/assets"
)

// Features says which parts of lore this binary implements; the AGENTS.md
// block only mentions what exists.
type Features struct {
	Changes bool // stage 2: lore change ...
}

var Current = Features{Changes: true}

// Log collects human-readable lines about what install steps did.
type Log struct{ w io.Writer }

func (l Log) f(format string, a ...any) { fmt.Fprintf(l.w, format+"\n", a...) }

// Init creates or completes a lore setup in dir. It is idempotent: existing
// files are left alone, only missing pieces are added.
func Init(dir string, out io.Writer) (*Project, error) {
	log := Log{out}
	cfgPath := filepath.Join(dir, ConfigFile)
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(cfgPath, []byte(assets.Read("lore.yaml")), 0o644); err != nil {
			return nil, err
		}
		log.f("created  %s", ConfigFile)
	}
	p, err := LoadProject(dir)
	if err != nil {
		return nil, err
	}

	for _, sub := range []string{"zones", "adr", "changes/archive"} {
		if err := os.MkdirAll(p.Path(sub), 0o755); err != nil {
			return nil, err
		}
		keep := p.Path(sub, ".gitkeep")
		if entries, _ := os.ReadDir(p.Path(sub)); len(entries) == 0 {
			if err := os.WriteFile(keep, nil, 0o644); err != nil {
				return nil, err
			}
		}
	}
	for _, doc := range []string{"product.md", "architecture.md"} {
		path := p.Path(doc)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(path, []byte(assets.Read("templates/"+doc)), 0o644); err != nil {
				return nil, err
			}
			log.f("created  %s", p.Rel(path))
		}
	}
	if err := p.syncManaged(log, false); err != nil {
		return nil, err
	}
	if changed, err := p.WriteIndex(); err != nil {
		return nil, err
	} else if changed {
		log.f("wrote    %s", p.Rel(p.Path("INDEX.md")))
	}
	if err := p.wireAgents(log); err != nil {
		return nil, err
	}
	return p, nil
}

// Upgrade refreshes workflows and prompts to this binary's version, keeping
// local edits unless force is set, and rewrites the AGENTS.md block.
func (p *Project) Upgrade(out io.Writer, force bool) error {
	log := Log{out}
	if err := p.upgradeConfig(log); err != nil {
		return err
	}
	if line := p.OldHookLine(); line > 0 {
		// the block is kept byte for byte, like every existing line (ADR 2026-10-27-upgrade-merges-config-as-text)
		log.f("hint     %s:%d: %s", ConfigFile, line, oldHookMsg)
	}
	if err := p.syncManaged(log, force); err != nil {
		return err
	}
	return p.wireAgents(log)
}

// wireAgents writes everything that connects lore to the agent: the AGENTS.md
// block, CLAUDE.md import, Claude Code settings, skill symlinks, .gitignore.
func (p *Project) wireAgents(log Log) error {
	if err := p.writeAgentsBlock(log); err != nil {
		return err
	}
	if err := p.ensureClaudeImport(log); err != nil {
		return err
	}
	if err := p.mergeClaudeSettings(log); err != nil {
		return err
	}
	if err := p.SyncAgents(log.w); err != nil {
		return err
	}
	if err := p.migrateLegacyLocal(log); err != nil {
		return err
	}
	root := filepath.ToSlash(p.Cfg.Root)
	for _, line := range []string{root + "/.env", root + "/.local/"} {
		if err := ensureLine(filepath.Join(p.Root, ".gitignore"), line, log, p); err != nil {
			return err
		}
	}
	return nil
}

// migrateLegacyLocal moves the old top-level .lore/ (state and .env) into the
// docs folder and drops its .gitignore line.
func (p *Project) migrateLegacyLocal(log Log) error {
	old := filepath.Join(p.Root, ".lore")
	entries, err := os.ReadDir(old)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p.LocalDir(), 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		dst := filepath.Join(p.LocalDir(), e.Name())
		if e.Name() == ".env" || e.Name() == "env" {
			dst = p.EnvFile()
		}
		if _, err := os.Stat(dst); err == nil {
			log.f("kept     %s — %s already exists", p.Rel(filepath.Join(old, e.Name())), p.Rel(dst))
			continue
		}
		if err := os.Rename(filepath.Join(old, e.Name()), dst); err != nil {
			return err
		}
		log.f("moved    %s → %s", p.Rel(filepath.Join(old, e.Name())), p.Rel(dst))
	}
	if err := os.Remove(old); err == nil {
		log.f("removed  .lore/")
	}
	gi := filepath.Join(p.Root, ".gitignore")
	if b, err := os.ReadFile(gi); err == nil {
		var keep []string
		for _, l := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(l) != ".lore/" {
				keep = append(keep, l)
			}
		}
		if next := strings.Join(keep, "\n"); next != string(b) {
			if err := os.WriteFile(gi, []byte(next), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- managed files (workflows, prompts) ---

const manifestFile = ".manifest.json"

type manifest struct {
	Files map[string]string `json:"files"` // path under the docs folder → sha256 of the installed version
}

func hash(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func (p *Project) readManifest() manifest {
	m := manifest{Files: map[string]string{}}
	if b, err := os.ReadFile(p.Path(manifestFile)); err == nil {
		_ = json.Unmarshal(b, &m)
		if m.Files == nil {
			m.Files = map[string]string{}
		}
	}
	return m
}

// syncManaged installs embedded workflows and prompts. A local file is
// overwritten only if nobody edited it since it was installed (its hash still
// matches the manifest) or force is set; otherwise the diff is shown and the
// local version kept.
func (p *Project) syncManaged(log Log, force bool) error {
	m := p.readManifest()
	for _, rel := range assets.Managed() {
		want := []byte(assets.Read(rel))
		path := p.Path(rel)
		have, err := os.ReadFile(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			if err := writeFile(path, want); err != nil {
				return err
			}
			log.f("created  %s", p.Rel(path))
		case err != nil:
			return err
		case bytes.Equal(have, want):
		case hash(have) == m.Files[rel] || force:
			if err := writeFile(path, want); err != nil {
				return err
			}
			log.f("updated  %s", p.Rel(path))
		default:
			log.f("kept     %s — edited locally; new version differs (use --force to overwrite):", p.Rel(path))
			log.f("%s", diff(have, want))
			continue // keep the old hash so the file still counts as edited
		}
		m.Files[rel] = hash(want)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.Path(manifestFile), append(b, '\n'), 0o644)
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// diff returns a unified diff (local → new) via diff(1), or a note if unavailable.
func diff(local, next []byte) string {
	dir, err := os.MkdirTemp("", "lore-diff")
	if err != nil {
		return "(diff unavailable)"
	}
	defer os.RemoveAll(dir)
	a, b := filepath.Join(dir, "local"), filepath.Join(dir, "new")
	_ = os.WriteFile(a, local, 0o644)
	_ = os.WriteFile(b, next, 0o644)
	out, _ := exec.Command("diff", "-u", a, b).Output()
	if len(out) == 0 {
		return "(diff unavailable)"
	}
	return string(out)
}

// --- AGENTS.md / CLAUDE.md ---

const (
	blockBegin = "<!-- lore:begin -->"
	blockEnd   = "<!-- lore:end -->"
)

func (p *Project) workflowNames() []string {
	entries, _ := os.ReadDir(p.Path("workflows"))
	var names []string
	for _, e := range entries {
		if _, err := os.Stat(p.Path("workflows", e.Name(), "SKILL.md")); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

func (p *Project) renderAgentsBlock() (string, error) {
	var wf []string
	for _, n := range p.workflowNames() {
		wf = append(wf, "/lore-"+n)
	}
	tmpl, err := template.New("block").Parse(assets.Read("agents-block.md"))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Root": p.Cfg.Root, "Changes": Current.Changes,
		"Workflows": strings.Join(wf, ", "),
	})
	return blockBegin + "\n" + strings.TrimRight(buf.String(), "\n") + "\n" + blockEnd + "\n", err
}

func (p *Project) writeAgentsBlock(log Log) error {
	block, err := p.renderAgentsBlock()
	if err != nil {
		return err
	}
	path := filepath.Join(p.Root, "AGENTS.md")
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	src := string(old)
	var next string
	b, e := strings.Index(src, blockBegin), strings.Index(src, blockEnd)
	switch {
	case b >= 0 && e > b:
		next = src[:b] + block + strings.TrimPrefix(src[e+len(blockEnd):], "\n")
	case src == "":
		next = block
	default:
		next = strings.TrimRight(src, "\n") + "\n\n" + block
	}
	if next == src {
		return nil
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return err
	}
	log.f("wrote    AGENTS.md (lore block)")
	return nil
}

func (p *Project) ensureClaudeImport(log Log) error {
	return ensureLine(filepath.Join(p.Root, "CLAUDE.md"), "@AGENTS.md", log, p)
}

// ensureLine appends line to the file unless an identical line is present.
func ensureLine(path, line string, log Log, p *Project) error {
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, l := range strings.Split(string(old), "\n") {
		if strings.TrimSpace(l) == line {
			return nil
		}
	}
	src := string(old)
	if src != "" && !strings.HasSuffix(src, "\n") {
		src += "\n"
	}
	if err := os.WriteFile(path, []byte(src+line+"\n"), 0o644); err != nil {
		return err
	}
	log.f("added    %q to %s", line, p.Rel(path))
	return nil
}

// --- .claude/settings.json ---

// mergeClaudeSettings lets the agent run lore freely and makes edits of ADRs
// ask the owner; zones, product and architecture are approved at the gate.
// Existing settings are kept; only missing rules are added, and lore's own
// older entries are removed. Paths starting with "/" are relative to the
// project root; Edit rules cover Write too.
func (p *Project) mergeClaudeSettings(log Log) error {
	path := filepath.Join(p.Root, ".claude", "settings.json")
	settings := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &settings); err != nil {
			return fmt.Errorf("%s: %w", p.Rel(path), err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	perms, _ := settings["permissions"].(map[string]any)
	if perms == nil {
		perms = map[string]any{}
	}
	root := "/" + filepath.ToSlash(p.Cfg.Root)
	added := addRules(perms, "allow", "Bash(lore:*)")
	// ADR 2026-10-27-edits-ask-only-for-adrs: ask only for ADRs, never deny.
	added += addRules(perms, "ask", "Edit("+root+"/adr/**)")
	removed := removeRules(perms, "ask",
		"Edit("+root+"/zones/**)",
		"Edit("+root+"/product.md)",
		"Edit("+root+"/architecture.md)",
	)
	settings["permissions"] = perms
	hookRemoved := removeOldHook(settings)
	if added == 0 && removed == 0 && !hookRemoved {
		return nil
	}
	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := writeFile(path, append(b, '\n')); err != nil {
		return err
	}
	msg := fmt.Sprintf("%d permission rules added", added)
	if removed > 0 {
		msg += fmt.Sprintf(", %d old ask rules removed", removed)
	}
	if hookRemoved {
		msg += ", old prompt hook removed"
	}
	log.f("updated  %s (%s)", p.Rel(path), msg)
	return nil
}

// oldHookCommand is the UserPromptSubmit hook older versions registered.
const oldHookCommand = "lore ctx --hook"

// removeOldHook drops lore's own old prompt hook from the settings, and the
// UserPromptSubmit entries and hooks map it leaves empty; other hooks stay.
// ADR 2026-10-27-agent-pulls-knowledge-with-ctx: no prompt hook.
func removeOldHook(settings map[string]any) bool {
	hooks, _ := settings["hooks"].(map[string]any)
	groups, _ := hooks["UserPromptSubmit"].([]any)
	removed := false
	var keep []any
	for _, g := range groups {
		gm, ok := g.(map[string]any)
		list, _ := gm["hooks"].([]any)
		if !ok || list == nil {
			keep = append(keep, g)
			continue
		}
		var rest []any
		for _, h := range list {
			if hm, _ := h.(map[string]any); hm != nil && hm["command"] == oldHookCommand {
				continue
			}
			rest = append(rest, h)
		}
		if len(rest) < len(list) {
			removed = true
			if len(rest) == 0 {
				continue
			}
			gm["hooks"] = rest
		}
		keep = append(keep, gm)
	}
	if !removed {
		return false
	}
	if len(keep) == 0 {
		delete(hooks, "UserPromptSubmit")
	} else {
		hooks["UserPromptSubmit"] = keep
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
	return true
}

func addRules(perms map[string]any, key string, rules ...string) int {
	list, _ := perms[key].([]any)
	have := map[string]bool{}
	for _, r := range list {
		if s, ok := r.(string); ok {
			have[s] = true
		}
	}
	n := 0
	for _, r := range rules {
		if !have[r] {
			list = append(list, r)
			n++
		}
	}
	perms[key] = list
	return n
}

// removeRules drops the given rules from perms[key], keeping every other one.
func removeRules(perms map[string]any, key string, rules ...string) int {
	list, _ := perms[key].([]any)
	drop := map[string]bool{}
	for _, r := range rules {
		drop[r] = true
	}
	var keep []any
	n := 0
	for _, r := range list {
		if s, ok := r.(string); ok && drop[s] {
			n++
			continue
		}
		keep = append(keep, r)
	}
	if n > 0 {
		perms[key] = keep
	}
	return n
}

// --- skills ---

// SyncAgents links every workflow into .claude/skills as lore-<name>, and
// removes lore-* links whose workflow is gone.
func (p *Project) SyncAgents(out io.Writer) error {
	log := Log{out}
	skills := filepath.Join(p.Root, ".claude", "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		return err
	}
	want := map[string]bool{}
	for _, name := range p.workflowNames() {
		link := filepath.Join(skills, "lore-"+name)
		want[link] = true
		target, err := filepath.Rel(skills, p.Path("workflows", name))
		if err != nil {
			return err
		}
		if cur, err := os.Readlink(link); err == nil {
			if cur == target {
				continue
			}
			if err := os.Remove(link); err != nil {
				return err
			}
		} else if _, err := os.Lstat(link); err == nil {
			log.f("skipped  %s — exists and is not a symlink", p.Rel(link))
			continue
		}
		// ADR 2026-10-27-workflows-linked-not-copied: a symlink, never a copy.
		if err := os.Symlink(target, link); err != nil {
			return err
		}
		log.f("linked   %s → %s", p.Rel(link), target)
	}
	entries, _ := os.ReadDir(skills)
	for _, e := range entries {
		link := filepath.Join(skills, e.Name())
		if !strings.HasPrefix(e.Name(), "lore-") || want[link] {
			continue
		}
		if _, err := os.Readlink(link); err == nil {
			if err := os.Remove(link); err != nil {
				return err
			}
			log.f("removed  %s (workflow is gone)", p.Rel(link))
		}
	}
	return nil
}
