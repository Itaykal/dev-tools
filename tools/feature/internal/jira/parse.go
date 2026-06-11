package jira

import (
	"encoding/json"
	"fmt"
	"strings"

	"feature/internal/tracker"
)

// parseList turns the jira CLI's `--plain --delimiter` table into issues. The
// first row is the column header (TYPE|KEY|SUMMARY|STATUS) and is skipped.
// Summaries are kept whole; display truncation is the TUI's job.
func parseList(raw, delim string) []tracker.Issue {
	var issues []tracker.Issue
	for i, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, delim)
		if len(cols) < 4 {
			continue
		}
		for j := range cols {
			cols[j] = strings.TrimSpace(cols[j])
		}
		// Skip the header row wherever it lands.
		if i == 0 && cols[0] == "TYPE" {
			continue
		}
		iss := tracker.Issue{
			Type:    tracker.IssueType(cols[0]),
			Key:     cols[1],
			Summary: cols[2],
			Status:  cols[3],
		}
		if len(cols) >= 5 {
			iss.Assignee = cols[4]
		}
		issues = append(issues, iss)
	}
	return issues
}

// rawIssue is the slice of `jira issue view --raw` JSON we care about.
type rawIssue struct {
	Fields struct {
		Summary   string `json:"summary"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee *struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Description any `json:"description"`
	} `json:"fields"`
}

// renderIssue builds the Markdown preview body for one issue from its --raw
// JSON: the summary as an H1, then the ADF-flattened description (or a
// placeholder when there is none). Issue metadata (type/status/assignee) is
// rendered separately by the TUI from list data, not here.
func renderIssue(rawJSON []byte) (string, error) {
	var iss rawIssue
	if err := json.Unmarshal(rawJSON, &iss); err != nil {
		return "", fmt.Errorf("decode issue JSON: %w", err)
	}

	body := "_No description_"
	if iss.Fields.Description != nil {
		if md := adfToMarkdown(iss.Fields.Description); md != "" {
			body = md
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", iss.Fields.Summary)
	b.WriteString(body)
	return b.String(), nil
}
