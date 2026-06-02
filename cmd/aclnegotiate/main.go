// Command aclnegotiate shows multiagent NEGOTIATION (pkg/acl, Contract Net): an
// initiator calls for proposals on a task, agents bid or refuse, and the best bid
// is awarded — the conversation printed performative by performative. Two tasks
// show context-dependent winners (a specialist wins its kind of work). Live
// tvcp-ai brains could bid through the same Bidder interface.
//
//	go run ./cmd/aclnegotiate
package main

import (
	"fmt"
	"strings"

	"github.com/svend4/infon/pkg/acl"
)

func main() {
	bidders := []acl.Bidder{
		acl.FuncBidder{N: "fast", F: func(t acl.Task) (int, bool) { return t.Size, true }},
		acl.FuncBidder{N: "thorough", F: func(t acl.Task) (int, bool) { return t.Size * 2, true }},
		acl.FuncBidder{N: "render-specialist", F: func(t acl.Task) (int, bool) {
			if strings.Contains(t.Name, "render") {
				return t.Size / 2, true // best at rendering
			}
			return 0, false // refuses anything else
		}},
		acl.FuncBidder{N: "overloaded", F: func(acl.Task) (int, bool) { return 0, false }},
	}

	tasks := []acl.Task{
		{ID: "job-1", Name: "render the scene", Size: 8},
		{ID: "job-2", Name: "compute a checksum", Size: 6},
	}

	for _, t := range tasks {
		out, log := acl.ContractNet("director", t, bidders)
		fmt.Printf("=== negotiation for %q (size %d) ===\n", t.Name, t.Size)
		for _, m := range log {
			fmt.Printf("  %-17s -%s-> %-17s %s\n", m.From, m.Perf, m.To, m.Content)
		}
		if out.Awarded {
			fmt.Printf("  AWARD: %s at cost %d\n\n", out.Winner, out.Cost)
		} else {
			fmt.Printf("  AWARD: none (all refused)\n\n")
		}
	}
	fmt.Println("agents that NEGOTIATE, not just compete — a specialist wins its kind of work.")
}
