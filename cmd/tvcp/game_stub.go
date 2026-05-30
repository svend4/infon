//go:build !experimental

package main

import (
	"fmt"
	"os"
)

// runGame is a stub in non-experimental builds. The interactive Snake game lives
// behind the `experimental` build tag (it brings in raw-TTY handling); build
// with `-tags experimental` to enable it.
func runGame() {
	fmt.Fprintln(os.Stderr, "The 'game' command requires an experimental build:")
	fmt.Fprintln(os.Stderr, "  go run -tags experimental ./cmd/tvcp game")
	os.Exit(1)
}
