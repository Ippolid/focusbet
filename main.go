// Command focusbet is a terminal pomodoro timer with a rest-currency mini-casino:
// focus to earn rest, then bank the fair break or gamble the surplus. This file
// is the composition root — it wires the store, app core and TUI, then runs the
// bubbletea program. No business logic lives here.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Ippolid/focusbet/internal/app"
	"github.com/Ippolid/focusbet/internal/consts"
	"github.com/Ippolid/focusbet/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "focusbet:", err)
		os.Exit(1)
	}
}

func run() error {
	now := time.Now().Unix()

	core, err := app.New(consts.GetStorageDir(), now)
	if err != nil {
		return fmt.Errorf("init app: %w", err)
	}

	p := tea.NewProgram(tui.New(core), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}

	// The TUI saves on quit; save once more as a backstop in case of an unclean exit.
	if err := core.Save(time.Now().Unix()); err != nil {
		return fmt.Errorf("final save: %w", err)
	}
	return nil
}
