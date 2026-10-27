package lore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Question is one typed question of the Decisions API (OpenRouter System One).
type Question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

// Decider answers yes/no questions about a state with calibrated probabilities.
type Decider interface {
	Noul(ctx context.Context, state string, qs map[string]Question) (map[string]float64, error)
}

// ErrNoKey means the decisions model is not configured.
var ErrNoKey = errors.New("decisions model: no API key")

// maxQuestions per request; larger batches are split and sent in parallel.
const maxQuestions = 50

type openRouter struct {
	endpoint string
	model    string
	key      string
	provider []string
	client   *http.Client
}

// NewDecider builds the client from lore.yaml. The key is lore's own, never the
// application's: the env var named in decisions_api_key_env, else that name in
// lore/.env (this project, gitignored), else in ~/.config/lore/.env (all projects).
// ADR 2026-10-27-lore-owns-its-key: never read the application's .env.
func (p *Project) NewDecider() (Decider, error) {
	if err := p.Cfg.checkKeyEnv(); err != nil {
		return nil, err
	}
	name := p.Cfg.DecisionsAPIKeyEnv
	key := os.Getenv(name)
	for _, file := range []string{p.EnvFile(), GlobalEnvFile()} {
		if key == "" && file != "" {
			key = readDotEnv(file)[name]
		}
	}
	if key == "" {
		return nil, fmt.Errorf("%w: put %s=<key> into %s (or %s, or the environment)", ErrNoKey, name, GlobalEnvFile(), p.Rel(p.EnvFile()))
	}
	return &openRouter{
		endpoint: p.Cfg.DecisionsEndpoint,
		model:    p.Cfg.DecisionsModel,
		key:      key,
		provider: p.Cfg.DecisionsProvider,
		client:   &http.Client{},
	}, nil
}

// GlobalEnvFile is lore's own env file for all projects: $XDG_CONFIG_HOME/lore/.env.
func GlobalEnvFile() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "lore", ".env")
}

var envNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// checkKeyEnv guards against putting the key itself into lore.yaml (a file
// that may be committed). The value is never echoed.
func (c *Config) checkKeyEnv() error {
	if !envNameRe.MatchString(c.DecisionsAPIKeyEnv) || strings.HasPrefix(c.DecisionsAPIKeyEnv, "sk") && len(c.DecisionsAPIKeyEnv) > 30 {
		return fmt.Errorf("%s: decisions_api_key_env must be the NAME of an env var (e.g. LORE_OPENROUTER_API_KEY), not the key — put the key into %s and rotate it if lore.yaml was shared", ConfigFile, GlobalEnvFile())
	}
	return nil
}

func readDotEnv(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return out
}

func (o *openRouter) Noul(ctx context.Context, state string, qs map[string]Question) (map[string]float64, error) {
	keys := make([]string, 0, len(qs))
	for k := range qs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		out  = map[string]float64{}
		errs []error
	)
	for start := 0; start < len(keys); start += maxQuestions {
		batch := map[string]Question{}
		for _, k := range keys[start:min(start+maxQuestions, len(keys))] {
			batch[k] = qs[k]
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := o.call(ctx, state, batch)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for k, v := range res {
				out[k] = v
			}
		}()
	}
	wg.Wait()
	return out, errors.Join(errs...)
}

type answer struct {
	Noul          *float64           `json:"noul"`
	Probabilities map[string]float64 `json:"probabilities"`
}

func (o *openRouter) call(ctx context.Context, state string, qs map[string]Question) (map[string]float64, error) {
	body := map[string]any{"model": o.model, "state": state, "questions": qs}
	if len(o.provider) > 0 {
		body["provider"] = map[string]any{"order": o.provider, "allow_fallbacks": false}
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+o.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("decisions model: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseAnswers(raw, resp.StatusCode)
}

// parseAnswers accepts {"answers": {...}} and a bare answers object.
func parseAnswers(raw []byte, status int) (map[string]float64, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("decisions model: HTTP %d: %s", status, truncate(string(raw), 200))
	}
	if e, ok := top["error"]; ok {
		var m struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(e, &m)
		if m.Message == "" {
			m.Message = truncate(string(e), 200)
		}
		return nil, fmt.Errorf("decisions model: HTTP %d: %s", status, m.Message)
	}
	if status >= 300 {
		return nil, fmt.Errorf("decisions model: HTTP %d: %s", status, truncate(string(raw), 200))
	}
	src := top
	if a, ok := top["answers"]; ok {
		src = nil
		if err := json.Unmarshal(a, &src); err != nil {
			return nil, fmt.Errorf("decisions model: answers: %w", err)
		}
	}
	out := map[string]float64{}
	for k, v := range src {
		var a answer
		if json.Unmarshal(v, &a) != nil {
			continue
		}
		switch {
		case a.Noul != nil:
			out[k] = *a.Noul
		case a.Probabilities != nil:
			if p, ok := a.Probabilities["true"]; ok {
				out[k] = p
			}
		}
	}
	return out, nil
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// --- prompts ---

// relatesQuestion asks whether a candidate relates to the work in the state.
func (p *Project) relatesQuestion(kind, name, text string) Question {
	return Question{
		Type:         "noul",
		Instructions: fillPrompt(p.prompt("relates"), kind, name, text),
		Criteria: map[string]string{
			"true":  fmt.Sprintf("The work relates to %s «%s».", kind, name),
			"false": fmt.Sprintf("The work does not relate to %s «%s».", kind, name),
		},
	}
}

// contradictsQuestion asks whether the fact in the state contradicts a decision.
func (p *Project) contradictsQuestion(name, text string) Question {
	return Question{
		Type:         "noul",
		Instructions: fillPrompt(p.prompt("contradicts"), "decision", name, text),
		Criteria: map[string]string{
			"true":  fmt.Sprintf("The fact contradicts the decision «%s».", name),
			"false": fmt.Sprintf("The fact does not contradict the decision «%s».", name),
		},
	}
}

func fillPrompt(tmpl, kind, name, text string) string {
	return strings.TrimSpace(strings.NewReplacer("{kind}", kind, "{name}", name, "{text}", text).Replace(tmpl))
}

// prompt reads the question text from the path in lore.yaml, falling back to
// lore/prompts/<name>.txt and then to the built-in text.
func (p *Project) prompt(name string) string {
	paths := []string{p.Path("prompts", name+".txt")}
	if rel := p.Cfg.Prompts[name]; rel != "" {
		paths = append([]string{filepath.Join(p.Root, rel)}, paths...)
	}
	for _, path := range paths {
		if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
			return string(b)
		}
	}
	return builtinPrompt(name)
}

// --- log ---

type logEntry struct {
	Time     string             `json:"time"`
	Source   string             `json:"source"`
	State    string             `json:"state"`
	Scores   map[string]float64 `json:"scores,omitempty"`
	Injected []string           `json:"injected,omitempty"`
	MS       int64              `json:"ms"`
	Error    string             `json:"error,omitempty"`
}

// logDecision appends to lore/.local/log.jsonl; failures to log are ignored.
func (p *Project) logDecision(e logEntry, started time.Time) {
	e.Time = time.Now().Format(time.RFC3339)
	e.MS = time.Since(started).Milliseconds()
	e.State = truncate(e.State, 500)
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	dir := p.LocalDir()
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "log.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}
