// Package jira implements tracker.Provider by shelling out to the `jira` CLI
// (ankitpokhrel/jira-cli). It is the only package that knows jira exists; the
// rest of the app talks to it through tracker.Provider. To support a different
// tracker, write a sibling package that satisfies the same interface.
package jira

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"feature/internal/config"
	"feature/internal/tracker"
)

const listDelim = "|"

// keyRe scrapes a Jira issue key (e.g. DRM-1234) from create output.
var keyRe = regexp.MustCompile(`[A-Z][A-Z0-9_]+-[0-9]+`)

// Provider is the jira-CLI-backed tracker.Provider.
type Provider struct {
	cfg config.Config
}

// New returns a Provider configured from cfg.
func New(cfg config.Config) *Provider {
	return &Provider{cfg: cfg}
}

var _ tracker.Provider = (*Provider)(nil)

// List runs `jira issue list` with the configured filters and parses the table.
func (p *Provider) List(ctx context.Context) ([]tracker.Issue, error) {
	args := []string{"issue", "list"}
	for _, s := range p.cfg.List.ExcludeStatuses {
		args = append(args, "-s~"+s)
	}
	for _, t := range p.cfg.List.ExcludeTypes {
		args = append(args, "-t~"+t)
	}
	if p.cfg.Assignee != "" {
		args = append(args, "-a", p.cfg.Assignee)
	}
	args = append(args, "--plain", "--columns", "TYPE,KEY,SUMMARY,STATUS,ASSIGNEE", "--delimiter", listDelim)

	out, err := p.run(ctx, nil, args...)
	if err != nil {
		return nil, err
	}
	return parseList(out, listDelim), nil
}

// Describe fetches one issue as --raw JSON and renders it to Markdown.
func (p *Provider) Describe(ctx context.Context, key string) (string, error) {
	out, err := p.run(ctx, nil, "issue", "view", key, "--raw")
	if err != nil {
		return "", err
	}
	return renderIssue([]byte(out))
}

// Create creates an issue, applies the configured custom fields, then moves it
// to the configured status (best-effort), and returns the new key.
func (p *Provider) Create(ctx context.Context, req tracker.CreateRequest) (string, error) {
	if req.Type == tracker.TypeSubtask {
		return "", fmt.Errorf("cannot create a Sub-task without a parent")
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		return "", fmt.Errorf("empty summary")
	}
	typ := req.Type
	if typ == "" {
		typ = tracker.TypeTask
	}

	args := []string{"issue", "create", "--no-input", "-t", string(typ), "-s", summary}
	if p.cfg.Assignee != "" {
		args = append(args, "-a", p.cfg.Assignee)
	}
	for _, kv := range sortedCustom(p.cfg.Create.Custom) {
		args = append(args, "--custom", kv)
	}

	// stdin from /dev/null so create never blocks waiting on a tty.
	devnull, _ := os.Open(os.DevNull)
	if devnull != nil {
		defer devnull.Close()
	}
	out, err := p.run(ctx, devnull, args...)
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}

	key := keyRe.FindString(out)
	if key == "" {
		return "", fmt.Errorf("could not parse issue key from create output:\n%s", out)
	}

	if p.cfg.Create.MoveTo != "" {
		// Best-effort, matching the old tool: a created issue that can't be
		// transitioned is still usable.
		_, _ = p.run(ctx, devnull, "issue", "move", key, p.cfg.Create.MoveTo)
	}
	return key, nil
}

// run executes the jira CLI and returns combined stdout+stderr.
func (p *Provider) run(ctx context.Context, stdin *os.File, args ...string) (string, error) {
	bin := p.cfg.JiraBin
	if bin == "" {
		bin = "jira"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return strings.TrimRight(buf.String(), "\n"), err
}

// sortedCustom renders the custom-field map as deterministic "key=value" args.
func sortedCustom(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+m[k])
	}
	return out
}
