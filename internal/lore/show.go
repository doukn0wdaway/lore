package lore

import (
	"fmt"
	"io"
	"strings"
)

// Show prints a zone followed by the ADRs that apply to it (its own and global).
func (p *Project) Show(w io.Writer, zoneID string) error {
	zones, err := p.LoadZones()
	if err != nil {
		return err
	}
	z := FindZone(zones, zoneID)
	if z == nil {
		return fmt.Errorf("no zone %q; zones: %s", zoneID, strings.Join(zoneIDs(zones), ", "))
	}
	adrs, err := p.LoadADRs()
	if err != nil {
		return err
	}

	fmt.Fprint(w, strings.TrimRight(z.Src, "\n")+"\n")

	var accepted, proposed []*ADR
	for _, a := range adrs {
		if !a.AppliesTo(z.Key()) {
			continue
		}
		switch a.Status {
		case "accepted":
			accepted = append(accepted, a)
		case "proposed":
			proposed = append(proposed, a)
		}
	}

	fmt.Fprintf(w, "\n--- Active decisions (%s, %s) ---\n", z.Key(), GlobalZone)
	if len(accepted) == 0 {
		fmt.Fprintln(w, "(none)")
	}
	for _, a := range accepted {
		writeADRSummary(w, a)
	}
	if len(proposed) > 0 {
		fmt.Fprintln(w, "\n--- Proposed, not binding yet ---")
		for _, a := range proposed {
			writeADRSummary(w, a)
		}
	}
	return nil
}

func writeADRSummary(w io.Writer, a *ADR) {
	tag := ""
	for _, z := range a.Zones {
		if z == GlobalZone {
			tag = "  [global]"
		}
	}
	fmt.Fprintf(w, "%s%s\n", a.Key(), tag)
	if t := strings.TrimSpace(a.TLDR); t != "" {
		fmt.Fprintf(w, "  %s\n", t)
	}
	for _, r := range a.Rules {
		fmt.Fprintf(w, "  %s\n", r)
	}
	fmt.Fprintf(w, "  → %s\n", a.File)
}
