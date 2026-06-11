// Command feature is a tracker-to-branch picker: choose one of your open
// issues (or create a new one) in a Bubble Tea TUI, then it cuts a git branch
// named KEY-slug. The issue tracker lives behind tracker.Provider; today that's
// Jira (via the jira CLI), but swapping it is a one-package change.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"feature/internal/config"
	"feature/internal/jira"
	"feature/internal/tui"
	"feature/internal/vcs"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	configPath := flag.String("config", "", "path to config TOML (overrides $FEATURE_CONFIG / XDG)")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, "feature:", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	provider := jira.New(cfg)
	model := tui.New(ctx, provider, cfg)

	// No alt-screen: render inline below the prompt like fzf, rather than
	// taking over the whole terminal.
	prog := tea.NewProgram(model)
	if _, err := prog.Run(); err != nil {
		return err
	}

	// The TUI has fully exited here, so git owns the terminal cleanly.
	res := model.Result()
	if res == nil {
		return nil // user quit without choosing
	}
	branch := vcs.Branch(res.Key, res.Summary)
	return vcs.Checkout(branch)
}
