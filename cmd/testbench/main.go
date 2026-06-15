// Command testbench is the technical test-bench for the tvcp-ai layer. It runs a
// battery of real in-process capabilities and reports them three ways from one
// engine (pkg/bench):
//
//	go run ./cmd/testbench          # an ANSI/Unicode table in the terminal (humans)
//	go run ./cmd/testbench -json    # the same Report as JSON (tools, dashboards, models)
//	go run ./cmd/testbench -plain   # no colour (logs / CI)
//
// Exit code is non-zero if any capability is red, so it doubles as a smoke test.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/svend4/infon/pkg/bench"
)

var groupOrder = []string{"protocol", "games", "substrate", "security", "directory", "agent-world"}

func main() {
	asJSON := flag.Bool("json", false, "emit the report as JSON")
	plain := flag.Bool("plain", false, "disable ANSI colour")
	flag.Parse()

	r := bench.Run()

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		if r.Passed != r.Total {
			os.Exit(1)
		}
		return
	}
	render(r, !*plain)
	if r.Passed != r.Total {
		os.Exit(1)
	}
}

func pad(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

func render(r bench.Report, color bool) {
	col := func(code, s string) string {
		if !color {
			return s
		}
		return "\x1b[" + code + "m" + s + "\x1b[0m"
	}
	bold := func(s string) string { return col("1", s) }
	dim := func(s string) string { return col("2", s) }
	green := func(s string) string { return col("32", s) }
	red := func(s string) string { return col("31", s) }
	cyan := func(s string) string { return col("36", s) }

	title := fmt.Sprintf(" TVCP testbench · %s · %s ", r.Protocol, r.When)
	bar := strings.Repeat("─", len([]rune(title)))
	fmt.Println("┌" + bar + "┐")
	fmt.Println("│" + bold(title) + "│")
	fmt.Println("└" + bar + "┘")

	for _, g := range groupOrder {
		var rows []bench.Check
		for _, ch := range r.Checks {
			if ch.Group == g {
				rows = append(rows, ch)
			}
		}
		if len(rows) == 0 {
			continue
		}
		fmt.Println(" " + cyan(strings.ToUpper(g)))
		for _, ch := range rows {
			mark := green("✓")
			if !ch.Pass {
				mark = red("✗")
			}
			fmt.Printf("   %s %s %s %s  %s\n",
				mark, pad(ch.Name, 14), pad(ch.Metric, 26),
				dim(pad(ch.Detail, 26)), dim(fmt.Sprintf("%dms", ch.Millis)))
		}
	}

	verdict := fmt.Sprintf(" %d/%d PASS ", r.Passed, r.Total)
	if r.Passed == r.Total {
		fmt.Println(green("▣" + verdict + "— all capabilities green"))
	} else {
		fmt.Println(red("▣" + verdict + "— see ✗ above"))
	}
}
