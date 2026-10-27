package lore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"lore/internal/assets"
)

// Change is lore/changes/<name>/: the work in progress.
type Change struct {
	Name    string
	Dir     string // absolute
	Meta    ChangeMeta
	MetaErr error
	Deltas  []*Delta
	Tasks   []Task
	Drafts  []*ADR // ADR drafts in adr/
}

type ChangeMeta struct {
	Mode     string `yaml:"mode"` // spec | code
	Base     string `yaml:"base,omitempty"`
	Created  string `yaml:"created"`
	Archived string `yaml:"archived,omitempty"`
	Head     string `yaml:"head,omitempty"`
}

type Task struct {
	Done bool
	Text string
	Line int
}

const (
	ModeSpec = "spec"
	ModeCode = "code"
	metaFile = ".meta.yaml"
)

var taskRe = regexp.MustCompile(`^\s*[-*]\s+\[([ xX])\]\s+(.+)$`)

func (p *Project) changesDir() string { return p.Path("changes") }

// LoadChanges returns active changes (not archived), sorted by name.
func (p *Project) LoadChanges() ([]*Change, error) {
	entries, err := os.ReadDir(p.changesDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*Change
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		c, err := p.loadChange(filepath.Join(p.changesDir(), e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (p *Project) loadChange(dir string) (*Change, error) {
	c := &Change{Name: filepath.Base(dir), Dir: dir}
	if b, err := os.ReadFile(filepath.Join(dir, metaFile)); err == nil {
		c.MetaErr = yaml.Unmarshal(b, &c.Meta)
	} else {
		c.MetaErr = fmt.Errorf("missing %s", metaFile)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "deltas", "*.md"))
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		c.Deltas = append(c.Deltas, parseDelta(strings.TrimSuffix(filepath.Base(f), ".md"), p.Rel(f), string(b)))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "tasks.md")); err == nil {
		for i, row := range strings.Split(string(b), "\n") {
			if m := taskRe.FindStringSubmatch(row); m != nil {
				c.Tasks = append(c.Tasks, Task{Done: m[1] != " ", Text: strings.TrimSpace(m[2]), Line: i + 1})
			}
		}
	}
	drafts, err := p.loadADRDir(filepath.Join(dir, "adr"))
	if err != nil {
		return nil, err
	}
	for _, a := range drafts {
		a.Change = c.Name
	}
	c.Drafts = drafts
	return c, nil
}

// newZones lists zones this change creates (deltas with frontmatter).
func (c *Change) newZones() map[string]bool {
	out := map[string]bool{}
	for _, d := range c.Deltas {
		if d.NewZone {
			out[d.Zone] = true
		}
	}
	return out
}

func (c *Change) progress() (done, total int) {
	for _, t := range c.Tasks {
		if t.Done {
			done++
		}
	}
	return done, len(c.Tasks)
}

func (c *Change) nextTask() *Task {
	for i := range c.Tasks {
		if !c.Tasks[i].Done {
			return &c.Tasks[i]
		}
	}
	return nil
}

// FindChange resolves a change by name; an empty name means the only active one.
func (p *Project) FindChange(name string) (*Change, error) {
	changes, err := p.LoadChanges()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, c := range changes {
		if c.Name == name {
			return c, nil
		}
		names = append(names, c.Name)
	}
	switch {
	case name != "":
		return nil, fmt.Errorf("no active change %q; active: %s", name, listOrNone(names))
	case len(changes) == 1:
		return changes[0], nil
	case len(changes) == 0:
		return nil, errors.New("no active changes")
	}
	return nil, fmt.Errorf("several active changes, name one: %s", strings.Join(names, ", "))
}

func listOrNone(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// --- git ---

func (p *Project) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = p.Root
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// --- new ---

// NewChange creates changes/<name>/. Spec mode starts from HEAD and gets a
// tasks.md; code mode (fromDiff) documents work already done since base
// (default: merge-base with the main branch).
func (p *Project) NewChange(name string, fromDiff bool, base string, now time.Time) (string, error) {
	if !slugRe.MatchString(name) {
		return "", fmt.Errorf("change name %q: use lowercase letters, digits and dashes", name)
	}
	dir := filepath.Join(p.changesDir(), name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("%s already exists", p.Rel(dir))
	}
	meta := ChangeMeta{Mode: ModeSpec, Created: now.Format("2006-01-02")}
	if fromDiff {
		meta.Mode = ModeCode
		if base == "" {
			b, err := p.git("merge-base", "HEAD", p.Cfg.MainBranch)
			if err != nil {
				return "", fmt.Errorf("can't find where the work started (%v) — pass --base <ref>", err)
			}
			base = b
		}
		b, err := p.git("rev-parse", "--verify", base+"^{commit}")
		if err != nil {
			return "", err
		}
		meta.Base = b
	} else if head, err := p.git("rev-parse", "HEAD"); err == nil {
		meta.Base = head
	}

	title := strings.ToUpper(name[:1]) + strings.ReplaceAll(name[1:], "-", " ")
	files := map[string]string{"proposal.md": "templates/proposal.md"}
	if !fromDiff {
		files["tasks.md"] = "templates/tasks.md"
	}
	for _, sub := range []string{"deltas", "adr"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return "", err
		}
	}
	for out, src := range files {
		tmpl := template.Must(template.New(out).Parse(assets.Read(src)))
		var b strings.Builder
		if err := tmpl.Execute(&b, map[string]string{"Title": title, "Mode": meta.Mode}); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, out), []byte(b.String()), 0o644); err != nil {
			return "", err
		}
	}
	if err := writeMeta(dir, meta); err != nil {
		return "", err
	}
	return p.Rel(dir), nil
}

func writeMeta(dir string, m ChangeMeta) error {
	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, metaFile), b, 0o644)
}

// --- list / status ---

func (p *Project) ListChanges(w io.Writer) error {
	changes, err := p.LoadChanges()
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		fmt.Fprintln(w, "no active changes")
		return nil
	}
	for _, c := range changes {
		done, total := c.progress()
		prog := "no tasks"
		if total > 0 {
			prog = fmt.Sprintf("%d/%d tasks", done, total)
		}
		var zones []string
		for _, d := range c.Deltas {
			zones = append(zones, d.Zone)
		}
		fmt.Fprintf(w, "%-28s %-5s %-10s zones: %s\n", c.Name, c.Meta.Mode, prog, listOrNone(zones))
	}
	return nil
}

// Status prints the next task and the context for it: this change's deltas and
// the decisions of the zones they touch.
func (p *Project) Status(w io.Writer, c *Change, d Decider) error {
	done, total := c.progress()
	fmt.Fprintf(w, "change: %s   mode: %s", c.Name, c.Meta.Mode)
	if total > 0 {
		fmt.Fprintf(w, "   progress: %d/%d", done, total)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "\nNEXT")
	switch next := c.nextTask(); {
	case next != nil:
		fmt.Fprintf(w, "  %s   (tasks.md:%d)\n", next.Text, next.Line)
	case c.Meta.Mode == ModeSpec && total == 0:
		fmt.Fprintln(w, "  tasks.md has no tasks yet — finish the proposal (/lore-propose)")
	default:
		fmt.Fprintf(w, "  all work done → run `lore change archive %s` (/lore-archive)\n", c.Name)
	}

	fmt.Fprintln(w, "\nDELTAS IN THIS CHANGE")
	if len(c.Deltas) == 0 {
		fmt.Fprintln(w, "  (none yet)")
	}
	zones := map[string]bool{}
	for _, d := range c.Deltas {
		zones[d.Zone] = true
		for _, s := range d.summary() {
			fmt.Fprintf(w, "  %-14s %s\n", d.Zone+":", s)
		}
	}

	adrs, err := p.LoadADRs()
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "\nDECISIONS FOR THESE ZONES")
	n := 0
	for _, a := range adrs {
		if a.Status != "accepted" {
			continue
		}
		for _, z := range a.Zones {
			if zones[z] || z == GlobalZone {
				writeADRSummary(indent{w}, a)
				n++
				break
			}
		}
	}
	if n == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	if next := c.nextTask(); next != nil {
		switch {
		case d == nil:
			fmt.Fprintln(w, "\n(decisions model not configured: no extra context for the task)")
		default:
			started := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			sel, err := p.Select(ctx, d, next.Text, p.Cfg.Hook.BudgetTokens)
			cancel()
			entry := logEntry{Source: "status", State: next.Text}
			if sel != nil {
				entry.Scores, entry.Injected = sel.Scores, sel.Keys()
				if out := sel.Render(taskHeader, nil); out != "" {
					fmt.Fprint(w, "\n"+out)
				}
			}
			if err != nil {
				entry.Error = err.Error()
				fmt.Fprintf(w, "\n(decisions model failed: %v)\n", err)
			}
			p.logDecision(entry, started)
		}
	}
	if len(c.Drafts) > 0 {
		fmt.Fprintln(w, "\nADR DRAFTS IN THIS CHANGE")
		for _, a := range c.Drafts {
			fmt.Fprintf(w, "  [%s] ", a.Status)
			writeADRSummary(w, a)
		}
	}

	for _, d := range c.Deltas {
		fmt.Fprintf(w, "\n--- %s ---\n%s\n", d.File, strings.TrimRight(d.src, "\n"))
	}
	return nil
}

type indent struct{ w io.Writer }

func (i indent) Write(b []byte) (int, error) {
	s := strings.ReplaceAll(strings.TrimRight(string(b), "\n"), "\n", "\n  ")
	_, err := io.WriteString(i.w, "  "+s+"\n")
	return len(b), err
}

// --- archive ---

// ArchiveCheck runs before the merge: the deltas must fit the current zones,
// spec-mode tasks must be done. It returns lint-style issues.
func (p *Project) ArchiveCheck(c *Change) ([]Issue, error) {
	zones, err := p.LoadZones()
	if err != nil {
		return nil, err
	}
	l := &linter{p: p}
	if c.Meta.Mode == ModeSpec {
		for _, t := range c.Tasks {
			if !t.Done {
				l.errf(p.Rel(filepath.Join(c.Dir, "tasks.md")), t.Line, "task not done: %s", t.Text)
			}
		}
	}
	if len(c.Deltas) == 0 {
		l.warnf(p.Rel(c.Dir), 0, "no deltas: nothing will change in the zones")
	}
	for _, d := range c.Deltas {
		d.checkBefore(FindZone(zones, d.Zone), l)
	}
	for _, a := range c.Drafts {
		if a.Status != "accepted" {
			l.warnf(a.File, 1, "draft is %s — the owner must accept it (`lore adr accept %s`) before --finish", a.Status, a.Key())
		}
	}
	all, err := p.Lint()
	if err != nil {
		return nil, err
	}
	prefix := p.Rel(c.Dir) + "/"
	for _, i := range all {
		if strings.HasPrefix(i.File, prefix) {
			l.issues = append(l.issues, i)
		}
	}
	return l.issues, nil
}

// ArchiveFinish runs after the agent merged the deltas and the owner approved
// the diff: it checks the merge, moves ADR drafts into adr/ (superseding what
// they replace), regenerates the index and moves the change to the archive.
func (p *Project) ArchiveFinish(c *Change, now time.Time, out io.Writer) ([]Issue, error) {
	zones, err := p.LoadZones()
	if err != nil {
		return nil, err
	}
	l := &linter{p: p}
	for _, d := range c.Deltas {
		d.checkMerged(FindZone(zones, d.Zone), l)
	}
	for _, a := range c.Drafts {
		if a.Status != "accepted" {
			l.errf(a.File, 1, "draft is %s — the owner must accept it first (`lore adr accept %s`)", a.Status, a.Key())
		}
		if _, err := os.Stat(p.Path("adr", a.Name+".md")); err == nil {
			l.errf(a.File, 1, "%s already exists", "adr/"+a.Name+".md")
		}
	}
	if CountErrors(l.issues) > 0 {
		return l.issues, nil
	}
	if _, err := p.WriteIndex(); err != nil {
		return nil, err
	}
	// ADR 2026-10-27-finish-requires-clean-lint: the whole project, not only this change.
	all, err := p.Lint()
	if err != nil {
		return nil, err
	}
	if CountErrors(all) > 0 {
		return all, nil
	}

	log := Log{out}
	main, err := p.LoadADRs()
	if err != nil {
		return nil, err
	}
	for _, a := range c.Drafts {
		dst := p.Path("adr", a.Name+".md")
		if err := os.Rename(filepath.Join(p.Root, a.File), dst); err != nil {
			return nil, err
		}
		log.f("moved    %s → %s", a.File, p.Rel(dst))
		for _, s := range a.Supersedes {
			old := findADR(main, s)
			if old == nil {
				return nil, fmt.Errorf("%s supersedes unknown ADR %q", a.Key(), s)
			}
			if err := p.supersede(old, a.Key()); err != nil {
				return nil, err
			}
			log.f("superseded %s by %s", old.Key(), a.Key())
		}
	}

	c.Meta.Archived = now.Format("2006-01-02")
	if head, err := p.git("rev-parse", "HEAD"); err == nil {
		c.Meta.Head = head
	}
	if err := writeMeta(c.Dir, c.Meta); err != nil {
		return nil, err
	}
	_ = os.Remove(filepath.Join(c.Dir, "adr")) // empty after the move
	dst := filepath.Join(p.changesDir(), "archive", c.Meta.Archived+"-"+c.Name)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(c.Dir, dst); err != nil {
		return nil, err
	}
	log.f("archived %s → %s", p.Rel(c.Dir), p.Rel(dst))
	return nil, nil
}
