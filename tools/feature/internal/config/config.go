// Package config loads feature's TOML configuration and supplies defaults.
// Everything the tool treats as a knob — who you are, which issues to list,
// what to set on new issues, the type aliases — lives here rather than being
// hardcoded, which keeps the Jira specifics out of the code.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the whole tunable surface of the tool.
type Config struct {
	// Assignee is the account whose issues are listed and who new issues are
	// assigned to. For Jira this is an email/account; empty means "current user".
	Assignee string `toml:"assignee"`

	// JiraBin is the jira CLI executable name (allows pinning a path).
	JiraBin string `toml:"jira_bin"`

	List    ListConfig        `toml:"list"`
	Create  CreateConfig      `toml:"create"`
	Aliases map[string]string `toml:"aliases"` // query prefix -> issue type name
}

// ListConfig controls which issues show up in the picker.
type ListConfig struct {
	ExcludeStatuses []string `toml:"exclude_statuses"`
	ExcludeTypes    []string `toml:"exclude_types"`
}

// CreateConfig controls what happens when you create an issue with ctrl-n.
type CreateConfig struct {
	// MoveTo is the status to transition a freshly created issue to (best-effort).
	MoveTo string `toml:"move_to"`
	// Custom are extra fields passed on create, e.g. {"squad": "Detection"}.
	Custom map[string]string `toml:"custom"`
}

// Default is the out-of-the-box config: generic enough to work for any user
// with no config file. Assignee is left empty on purpose — the Jira provider
// resolves it to the currently authenticated user (`jira me`), so everyone
// sees their own open issues without configuring anything.
func Default() Config {
	return Config{
		Assignee: "",
		JiraBin:  "jira",
		List: ListConfig{
			ExcludeStatuses: []string{"Done", "Archived"},
			ExcludeTypes:    []string{"Epic"},
		},
		Create: CreateConfig{
			MoveTo: "In Progress",
			// No custom fields by default — team-specific fields (e.g. a squad)
			// are opt-in via [create.custom] in the config file.
			Custom: nil,
		},
		Aliases: map[string]string{
			"b": "Bug", "bug": "Bug",
			"t": "Task", "task": "Task",
			"s": "Story", "story": "Story",
			"st": "Sub-task", "sub": "Sub-task", "subtask": "Sub-task",
		},
	}
}

// Load resolves a config file, layering anything it finds over the defaults.
// Resolution order: explicit path arg, then $FEATURE_CONFIG, then
// $XDG_CONFIG_HOME/feature/config.toml (~/.config/feature/config.toml). A
// missing file is not an error — defaults are returned.
func Load(explicitPath string) (Config, error) {
	cfg := Default()

	path := resolvePath(explicitPath)
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("stat config %s: %w", path, err)
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func resolvePath(explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}
	if p := os.Getenv("FEATURE_CONFIG"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "feature", "config.toml")
}
