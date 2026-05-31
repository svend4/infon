//go:build experimental

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/svend4/infon/experimental/games/snake"
	"github.com/svend4/infon/pkg/color"
	"github.com/svend4/infon/pkg/terminal"
)

// runGame is a real-time, interactive Snake rendered with the flicker-free diff
// renderer. Unlike the turn-based AI games, this is a live game loop: a ticker
// advances the world while keystrokes steer the snake. It demonstrates that the
// terminal graphics pipeline supports real-time games (Question 1).
//
//	tvcp game            # default 32x20 board
//
// Controls: arrow keys or WASD to steer, q to quit.
func runGame() {
	width, height := 32, 20

	g := snake.New(width, height, func(n int) int { return rand.Intn(n) })

	// Put the terminal into raw mode so keystrokes arrive immediately without
	// echo. Best-effort via stty (no extra deps); restored on exit.
	restore := makeRaw()
	defer restore()

	fmt.Print(terminal.HideCursor, color.ClearScreen)
	defer fmt.Print(terminal.ShowCursor, color.Reset, color.ClearScreen)

	// Read keystrokes in the background; the loop never blocks on input.
	keys := make(chan byte, 16)
	go readKeys(keys)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	dr := terminal.NewDiffRenderer()
	tick := time.NewTicker(110 * time.Millisecond) // game speed
	defer tick.Stop()

	render := func() {
		fmt.Print(dr.Render(drawGame(g, width, height)))
	}
	render()

	for {
		select {
		case <-sig:
			return
		case b := <-keys:
			switch b {
			case 'q', 'Q', 3: // q or Ctrl-C
				return
			case 'w', 'W', 'A': // 'A' is the final byte of the Up arrow escape
				g.Turn(snake.Up)
			case 's', 'S', 'B': // Down arrow
				g.Turn(snake.Down)
			case 'a', 'D': // Left arrow ('D' final byte); note lowercase 'a'
				g.Turn(snake.Left)
			case 'd', 'C': // Right arrow ('C' final byte)
				g.Turn(snake.Right)
			}
		case <-tick.C:
			if g.Tick() != snake.Playing {
				render()
				time.Sleep(900 * time.Millisecond)
				return
			}
			render()
		}
	}
}

// drawGame renders the current game state into a Frame (board + border + HUD).
func drawGame(g *snake.Game, w, h int) *terminal.Frame {
	// +2 for the border, +1 row for the HUD.
	f := terminal.NewFrame(w+2, h+3)
	bg := color.NewRGB(12, 14, 22)
	wall := color.NewRGB(70, 80, 110)
	f.Fill(' ', color.White, bg)

	f.DrawBox(0, 0, w+2, h+2, wall, bg)

	// Food.
	f.SetBlock(g.Food.X+1, g.Food.Y+1, '●', color.NewRGB(240, 90, 90), bg)

	// Snake: bright head, cooler body.
	for i, p := range g.Body {
		glyph := '█'
		c := color.NewRGB(120, 230, 140)
		if i == 0 {
			c = color.NewRGB(220, 255, 220)
		}
		f.SetBlock(p.X+1, p.Y+1, glyph, c, bg)
	}

	hud := fmt.Sprintf(" score %d   len %d   (arrows/WASD, q to quit) ", g.Score, len(g.Body))
	f.DrawText(0, h+2, hud, color.NewRGB(180, 190, 210), bg)
	if g.State == snake.Lost {
		f.DrawText(2, h/2+1, " GAME OVER ", color.NewRGB(255, 230, 230), color.NewRGB(160, 40, 40))
	} else if g.State == snake.Won {
		f.DrawText(2, h/2+1, " YOU WIN! ", color.NewRGB(230, 255, 230), color.NewRGB(40, 140, 60))
	}
	return f
}

// readKeys streams single bytes from stdin into ch. Arrow-key escape sequences
// (ESC [ A/B/C/D) arrive as separate bytes; the loop keys on the final letter.
func readKeys(ch chan<- byte) {
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			close(ch)
			return
		}
		if n > 0 {
			ch <- buf[0]
		}
	}
}

// makeRaw switches the controlling TTY to raw mode using stty and returns a
// function that restores the previous settings. If stty is unavailable (e.g. a
// non-Unix host or no TTY), it is a no-op so the command still runs.
func makeRaw() func() {
	saved, err := exec.Command("stty", "-F", "/dev/tty", "-g").Output()
	if err != nil {
		// Try the form that reads stdin's TTY (some shells/OSes).
		saved, err = sttyState()
		if err != nil {
			return func() {}
		}
	}
	state := string(saved)
	_ = runStty("-F", "/dev/tty", "raw", "-echo")
	return func() { _ = runStty("-F", "/dev/tty", state) }
}

func sttyState() ([]byte, error) {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	return cmd.Output()
}

func runStty(args ...string) error {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
