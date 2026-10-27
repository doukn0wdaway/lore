package lore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const ConfigFile = "lore.yaml"

// Config is the part of lore.yaml the binary reads.
type Config struct {
	Root       string `yaml:"root"`
	MainBranch string `yaml:"main_branch"`

	DecisionsModel     string   `yaml:"decisions_model"`
	DecisionsAPIKeyEnv string   `yaml:"decisions_api_key_env"`
	DecisionsEndpoint  string   `yaml:"decisions_endpoint"`
	DecisionsProvider  []string `yaml:"decisions_provider"`

	Ctx        CtxConfig         `yaml:"ctx"`
	Thresholds Thresholds        `yaml:"thresholds"`
	Prompts    map[string]string `yaml:"prompts"`

	AnchorsIgnore []string `yaml:"anchors_ignore"`
}

type CtxConfig struct {
	BudgetTokens int `yaml:"budget_tokens"` // how much of whole zones `lore ctx` prints
}

type Thresholds struct {
	Zones          float64 `yaml:"zones"`
	IncludeContent float64 `yaml:"include_content"`
	IncludePointer float64 `yaml:"include_pointer"`
	Contradiction  float64 `yaml:"contradiction"`
}

func (c *Config) defaults() {
	set := func(v *string, d string) {
		if *v == "" {
			*v = d
		}
	}
	setF := func(v *float64, d float64) {
		if *v == 0 {
			*v = d
		}
	}
	setI := func(v *int, d int) {
		if *v == 0 {
			*v = d
		}
	}
	set(&c.Root, "lore")
	set(&c.MainBranch, "main")
	set(&c.DecisionsModel, "perplexity/pplx-decider-v1.2") // ADR 2026-10-27-pplx-decider-only: no flash model
	set(&c.DecisionsAPIKeyEnv, "LORE_OPENROUTER_API_KEY")
	set(&c.DecisionsEndpoint, "https://openrouter.ai/api/alpha/decisions")
	setI(&c.Ctx.BudgetTokens, 8000)
	setF(&c.Thresholds.Zones, 0.3)
	setF(&c.Thresholds.IncludeContent, 0.7)
	setF(&c.Thresholds.IncludePointer, 0.4)
	setF(&c.Thresholds.Contradiction, 0.5)
}

// Project is a directory with lore.yaml.
type Project struct {
	Root string // absolute path of the directory holding lore.yaml
	Cfg  Config
}

var ErrNoProject = errors.New("no lore.yaml found here or in any parent directory — run `lore init`")

// FindProject walks up from start looking for lore.yaml, the way git looks for .git.
func FindProject(start string) (*Project, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
			return LoadProject(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNoProject
		}
		dir = parent
	}
}

func LoadProject(dir string) (*Project, error) {
	p := &Project{Root: dir, Cfg: Config{Root: "lore", MainBranch: "main"}}
	b, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &p.Cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	p.Cfg.defaults()
	return p, nil
}

// OldHookLine is the line of a top-level `hook:` block left in lore.yaml by an
// older version (lore has no prompt hook any more), or 0.
func (p *Project) OldHookLine() int {
	b, err := os.ReadFile(filepath.Join(p.Root, ConfigFile))
	if err != nil {
		return 0
	}
	for i, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "hook:") {
			return i + 1
		}
	}
	return 0
}

// oldHookMsg explains what to do with an old `hook:` block.
const oldHookMsg = "`hook:` is no longer used (lore has no prompt hook) — delete the block; the budget of `lore ctx` is `ctx.budget_tokens`"

// Dir is the absolute path of the docs folder.
func (p *Project) Dir() string { return filepath.Join(p.Root, p.Cfg.Root) }

// LocalDir holds per-machine state inside the docs folder (gitignored):
// decisions log, bootstrap progress.
func (p *Project) LocalDir() string { return p.Path(".local") }

// EnvFile is the project's own lore env file (gitignored), separate from the app's .env.
func (p *Project) EnvFile() string { return p.Path(".env") }

// Path joins parts under the docs folder.
func (p *Project) Path(parts ...string) string {
	return filepath.Join(append([]string{p.Dir()}, parts...)...)
}

// Rel makes an absolute path relative to the project root, for messages.
func (p *Project) Rel(abs string) string {
	if r, err := filepath.Rel(p.Root, abs); err == nil {
		return filepath.ToSlash(r)
	}
	return abs
}

var (
	zoneIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	adrIDRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[a-z0-9]+(-[a-z0-9]+)*$`)
	slugRe   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// GlobalZone is the reserved zone id for project-wide ADRs.
const GlobalZone = "global"

type Zone struct {
	ID   string `yaml:"id"`
	TLDR string `yaml:"tldr"`

	File     string // relative to project root
	Name     string // file name without .md
	Src      string
	FrontErr error
	doc      *mdDoc
}

// Key is the zone's id, falling back to the file name when the id is missing.
func (z *Zone) Key() string {
	if z.ID != "" {
		return z.ID
	}
	return z.Name
}

type ADR struct {
	ID           string   `yaml:"id"`
	Status       string   `yaml:"status"`
	Zones        []string `yaml:"zones"`
	TLDR         string   `yaml:"tldr"`
	Rules        []string `yaml:"rules"`
	Supersedes   []string `yaml:"supersedes"`
	SupersededBy string   `yaml:"superseded_by"`
	Inferred     bool     `yaml:"inferred"`
	Ref          string   `yaml:"ref"`

	Change   string // set for a draft living in changes/<name>/adr/
	File     string
	Name     string
	Title    string
	Src      string
	FrontErr error
	doc      *mdDoc
}

func (a *ADR) Key() string {
	if a.ID != "" {
		return a.ID
	}
	return a.Name
}

// AppliesTo reports whether the ADR is attached to the zone or is global.
func (a *ADR) AppliesTo(zone string) bool {
	for _, z := range a.Zones {
		if z == zone || z == GlobalZone {
			return true
		}
	}
	return false
}

func (p *Project) markdownFiles(sub string) ([]string, error) {
	entries, err := os.ReadDir(p.Path(sub))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, p.Path(sub, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func (p *Project) LoadZones() ([]*Zone, error) {
	files, err := p.markdownFiles("zones")
	if err != nil {
		return nil, err
	}
	var zones []*Zone
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		z := &Zone{File: p.Rel(f), Name: strings.TrimSuffix(filepath.Base(f), ".md"), Src: string(b)}
		z.doc = parseDoc(z.Src)
		if z.doc.hasFront {
			z.FrontErr = yaml.Unmarshal([]byte(z.doc.front), z)
		}
		zones = append(zones, z)
	}
	sort.Slice(zones, func(i, j int) bool { return zones[i].Key() < zones[j].Key() })
	return zones, nil
}

// LoadADRs returns the ADRs in adr/ — drafts inside changes are not included.
func (p *Project) LoadADRs() ([]*ADR, error) { return p.loadADRDir(p.Path("adr")) }

// LoadAllADRs returns ADRs in adr/ followed by the drafts of active changes.
func (p *Project) LoadAllADRs() ([]*ADR, error) {
	adrs, err := p.LoadADRs()
	if err != nil {
		return nil, err
	}
	changes, err := p.LoadChanges()
	if err != nil {
		return nil, err
	}
	for _, c := range changes {
		adrs = append(adrs, c.Drafts...)
	}
	return adrs, nil
}

func (p *Project) loadADRDir(dir string) ([]*ADR, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var adrs []*ADR
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		f := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		a := &ADR{File: p.Rel(f), Name: strings.TrimSuffix(e.Name(), ".md"), Src: string(b)}
		a.doc = parseDoc(a.Src)
		if a.doc.hasFront {
			a.FrontErr = yaml.Unmarshal([]byte(a.doc.front), a)
		}
		a.Title = a.doc.title()
		adrs = append(adrs, a)
	}
	return adrs, nil
}

func yamlUnmarshal(src string, v any) error { return yaml.Unmarshal([]byte(src), v) }

// FindZone returns the zone with the given id, or nil.
func FindZone(zones []*Zone, id string) *Zone {
	for _, z := range zones {
		if z.Key() == id {
			return z
		}
	}
	return nil
}

func zoneIDs(zones []*Zone) []string {
	ids := make([]string, 0, len(zones))
	for _, z := range zones {
		ids = append(ids, z.Key())
	}
	return ids
}
