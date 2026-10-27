// Command lore keeps a project's knowledge for AI agents: zones (what the
// system does), ADRs (why it is built this way) and changes (what we do now).
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"lore/internal/lore"
)

const version = "0.3.1 (stage 3)"

const usage = `lore — project knowledge for AI agents

Usage:
  lore init                         set up lore in the current directory (idempotent)
  lore index                        regenerate lore/INDEX.md
  lore lint [--json]                check zones, ADRs, index, product, code anchors
  lore show <zone>                  print a zone and the ADRs that apply to it
  lore adr new <slug> --zones a,b [--title "..."] [--supersedes id,...] [--change name]
                                    create an ADR (status proposed); zones may be "global";
                                    --change puts a draft into that change
  lore adr accept <id>              owner approved: proposed → accepted, replaced ADRs → superseded
  lore adr list [--zone z] [--status s] [--json]
  lore change new <name> [--from-diff [--base ref]]
                                    start a change: spec-first, or code-first from the git diff
  lore change list                  active changes
  lore change status [name]         next task + deltas + decisions of the touched zones
  lore change archive [name]        pre-merge checks for the gates
  lore change archive [name] --finish
                                    after merge + owner's yes: verify, move ADR drafts, archive
                                    (name may be omitted when one change is active)
  lore ctx "<text>"                 pick knowledge for a text with the decisions model
  lore ctx --hook                   Claude Code UserPromptSubmit hook (reads JSON on stdin)
  lore change check [name]          facts of the change vs zones and accepted ADRs (model)
  lore agents sync                  link lore/workflows/* into .claude/skills as lore-*
  lore upgrade [--force]            update workflows/prompts and the AGENTS.md block
  lore version
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		var exit exitError
		if errors.As(err, &exit) {
			os.Exit(int(exit))
		}
		fmt.Fprintln(os.Stderr, "lore:", err)
		os.Exit(1)
	}
}

type exitError int

func (e exitError) Error() string { return fmt.Sprintf("exit %d", int(e)) }

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			fmt.Print(usage)
			return nil
		}
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help":
		fmt.Print(usage)
		return nil
	case "version", "--version":
		fmt.Println("lore", version)
		return nil
	case "init":
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		if _, err := lore.Init(dir, os.Stdout); err != nil {
			return err
		}
		fmt.Println("lore is set up. Next: run /lore-bootstrap in Claude Code to document the project.")
		return nil
	}

	if cmd == "ctx" && len(rest) == 1 && rest[0] == "--hook" {
		lore.RunHook(os.Stdin, os.Stdout, func(p *lore.Project) (lore.Decider, error) { return p.NewDecider() })
		return nil
	}

	p, err := lore.FindProject(".")
	if err != nil {
		return err
	}
	switch cmd {
	case "index":
		changed, err := p.WriteIndex()
		if err != nil {
			return err
		}
		if changed {
			fmt.Println("INDEX.md updated")
		} else {
			fmt.Println("INDEX.md is up to date")
		}
		return nil
	case "lint":
		return cmdLint(p, rest)
	case "show":
		f, err := parseFlags(rest, nil, nil)
		if err != nil {
			return err
		}
		if len(f.args) != 1 {
			return errors.New("usage: lore show <zone>")
		}
		return p.Show(os.Stdout, f.args[0])
	case "adr":
		return cmdADR(p, rest)
	case "change":
		return cmdChange(p, rest)
	case "ctx":
		text := strings.TrimSpace(strings.Join(rest, " "))
		if text == "" {
			return errors.New(`usage: lore ctx "<text>"`)
		}
		d, err := p.NewDecider()
		if err != nil {
			return err
		}
		return p.Ctx(os.Stdout, text, d)
	case "agents":
		if len(rest) != 1 || rest[0] != "sync" {
			return errors.New("usage: lore agents sync")
		}
		return p.SyncAgents(os.Stdout)
	case "upgrade":
		f, err := parseFlags(rest, nil, []string{"force"})
		if err != nil {
			return err
		}
		return p.Upgrade(os.Stdout, f.bools["force"])
	}
	return fmt.Errorf("unknown command %q — see `lore help`", cmd)
}

func cmdLint(p *lore.Project, args []string) error {
	f, err := parseFlags(args, nil, []string{"json"})
	if err != nil {
		return err
	}
	issues, err := p.Lint()
	if err != nil {
		return err
	}
	if f.bools["json"] {
		if issues == nil {
			issues = []lore.Issue{}
		}
		if err := printJSON(issues); err != nil {
			return err
		}
	} else {
		printIssues(issues)
	}
	if lore.CountErrors(issues) > 0 {
		return exitError(1)
	}
	return nil
}

func cmdADR(p *lore.Project, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: lore adr new|accept|list")
	}
	switch args[0] {
	case "new":
		f, err := parseFlags(args[1:], []string{"zones", "title", "supersedes", "change"}, nil)
		if err != nil {
			return err
		}
		if len(f.args) != 1 {
			return errors.New(`usage: lore adr new <slug> --zones a,b [--title "..."] [--supersedes id,...]`)
		}
		path, err := p.NewADR(f.args[0], splitList(f.values["zones"]), f.values["title"], splitList(f.values["supersedes"]), f.values["change"], time.Now())
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	case "accept":
		if len(args) != 2 {
			return errors.New("usage: lore adr accept <id>")
		}
		changed, err := p.AcceptADR(args[1])
		if err != nil {
			return err
		}
		for _, f := range changed {
			fmt.Println("updated", f)
		}
		return nil
	case "list":
		f, err := parseFlags(args[1:], []string{"zone", "status"}, []string{"json"})
		if err != nil {
			return err
		}
		adrs, err := p.ListADRs(f.values["zone"], f.values["status"])
		if err != nil {
			return err
		}
		if f.bools["json"] {
			type row struct {
				ID     string   `json:"id"`
				Status string   `json:"status"`
				Zones  []string `json:"zones"`
				TLDR   string   `json:"tldr"`
				File   string   `json:"file"`
			}
			rows := []row{}
			for _, a := range adrs {
				rows = append(rows, row{a.Key(), a.Status, a.Zones, a.TLDR, a.File})
			}
			return printJSON(rows)
		}
		if len(adrs) == 0 {
			fmt.Println("no ADRs")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSTATUS\tZONES\tTLDR")
		for _, a := range adrs {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Key(), a.Status, strings.Join(a.Zones, ","), a.TLDR)
		}
		return w.Flush()
	}
	return fmt.Errorf("unknown adr command %q", args[0])
}

func cmdChange(p *lore.Project, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: lore change new|list|status|check|archive")
	}
	switch args[0] {
	case "new":
		f, err := parseFlags(args[1:], []string{"base"}, []string{"from-diff"})
		if err != nil {
			return err
		}
		if len(f.args) != 1 {
			return errors.New("usage: lore change new <name> [--from-diff [--base ref]]")
		}
		if f.values["base"] != "" && !f.bools["from-diff"] {
			return errors.New("--base only makes sense with --from-diff")
		}
		dir, err := p.NewChange(f.args[0], f.bools["from-diff"], f.values["base"], time.Now())
		if err != nil {
			return err
		}
		fmt.Println(dir)
		return nil
	case "list":
		return p.ListChanges(os.Stdout)
	case "status":
		f, err := parseFlags(args[1:], nil, nil)
		if err != nil {
			return err
		}
		c, err := p.FindChange(first(f.args))
		if err != nil {
			return err
		}
		d, _ := p.NewDecider() // optional: without a key status shows deltas and ADRs only
		return p.Status(os.Stdout, c, d)
	case "check":
		f, err := parseFlags(args[1:], nil, nil)
		if err != nil {
			return err
		}
		c, err := p.FindChange(first(f.args))
		if err != nil {
			return err
		}
		d, err := p.NewDecider()
		if err != nil {
			return err
		}
		reports, err := p.CheckFacts(c, d)
		if err != nil {
			return err
		}
		lore.WriteFactReports(os.Stdout, c, reports)
		return nil
	case "archive":
		f, err := parseFlags(args[1:], nil, []string{"finish"})
		if err != nil {
			return err
		}
		c, err := p.FindChange(first(f.args))
		if err != nil {
			return err
		}
		if f.bools["finish"] {
			issues, err := p.ArchiveFinish(c, time.Now(), os.Stdout)
			if err != nil {
				return err
			}
			if len(issues) > 0 {
				printIssues(issues)
				fmt.Println("not archived: fix the errors above and run --finish again")
				return exitError(1)
			}
			return nil
		}
		issues, err := p.ArchiveCheck(c)
		if err != nil {
			return err
		}
		printIssues(issues)
		if lore.CountErrors(issues) > 0 {
			fmt.Println("fix the errors above before merging")
			return exitError(1)
		}
		fmt.Printf(`ready to merge %s. Next (see /lore-archive):
  1. reconcile deltas with the code, check facts against ADRs, settle ADR candidates with the owner
  2. merge deltas into lore/zones/ (and product.md / architecture.md)
  3. show the owner the diff of lore/, get a yes
  4. lore change archive %s --finish
`, c.Name, c.Name)
		return nil
	}
	return fmt.Errorf("unknown change command %q", args[0])
}

func first(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func printIssues(issues []lore.Issue) {
	for _, i := range issues {
		fmt.Println(i)
	}
	errs := lore.CountErrors(issues)
	fmt.Printf("%d error(s), %d warning(s)\n", errs, len(issues)-errs)
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// flags are parsed by hand so they may come before or after positional args.
type flags struct {
	args   []string
	values map[string]string
	bools  map[string]bool
}

func parseFlags(args, valueFlags, boolFlags []string) (flags, error) {
	f := flags{values: map[string]string{}, bools: map[string]bool{}}
	isValue := map[string]bool{}
	for _, v := range valueFlags {
		isValue[v] = true
	}
	isBool := map[string]bool{}
	for _, b := range boolFlags {
		isBool[b] = true
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			f.args = append(f.args, a)
			continue
		}
		name := strings.TrimLeft(a, "-")
		val, hasVal := "", false
		if k, v, ok := strings.Cut(name, "="); ok {
			name, val, hasVal = k, v, true
		}
		switch {
		case isBool[name]:
			f.bools[name] = true
		case isValue[name]:
			if !hasVal {
				if i+1 >= len(args) {
					return f, fmt.Errorf("--%s needs a value", name)
				}
				i++
				val = args[i]
			}
			f.values[name] = val
		default:
			return f, fmt.Errorf("unknown flag --%s", name)
		}
	}
	return f, nil
}
