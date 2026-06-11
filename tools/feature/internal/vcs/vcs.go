// Package vcs holds the git side of the tool: turning an issue into a branch
// name and creating that branch. It's deliberately separate from any tracker.
package vcs

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slug lowercases summary and turns any run of non-alphanumeric characters into
// a single hyphen, trimming leading/trailing hyphens. Mirrors the old zsh
// pipeline (tr/sed) so branch names stay consistent.
func Slug(summary string) string {
	s := strings.ToLower(summary)
	s = nonSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Branch builds "KEY-slug(summary)".
func Branch(key, summary string) string {
	slug := Slug(summary)
	if slug == "" {
		return key
	}
	return key + "-" + slug
}

// Checkout runs `git checkout -b <branch>`, wiring stdio to the terminal so the
// user sees git's output directly.
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
