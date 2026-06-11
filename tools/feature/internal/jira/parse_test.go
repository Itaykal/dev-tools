package jira

import (
	"strings"
	"testing"

	"feature/internal/tracker"
)

func TestParseList(t *testing.T) {
	// Shape mirrors real `jira issue list --plain --delimiter '|'` output:
	// a header row plus pipe-delimited rows, statuses with spaces.
	raw := strings.Join([]string{
		"TYPE|KEY|SUMMARY|STATUS|ASSIGNEE",
		"Bug|DRM-43930|Backup timeouts|In Review|Itay Kalfon",
		"Story|DRM-43616|Research ClickHouse best practices|Selected for Development|Itay Kalfon",
		"Sub-task|DRM-42399|Enable CD via Jenkins|In Review|Itay Kalfon",
		"", // trailing blank line
	}, "\n")

	got := parseList(raw, "|")
	want := []tracker.Issue{
		{Type: tracker.TypeBug, Key: "DRM-43930", Summary: "Backup timeouts", Status: "In Review", Assignee: "Itay Kalfon"},
		{Type: tracker.TypeStory, Key: "DRM-43616", Summary: "Research ClickHouse best practices", Status: "Selected for Development", Assignee: "Itay Kalfon"},
		{Type: tracker.TypeSubtask, Key: "DRM-42399", Summary: "Enable CD via Jenkins", Status: "In Review", Assignee: "Itay Kalfon"},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d issues, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("issue %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRenderIssue(t *testing.T) {
	t.Run("with description", func(t *testing.T) {
		raw := `{"fields":{"summary":"Fix login","issuetype":{"name":"Bug"},"status":{"name":"In Progress"},"assignee":{"displayName":"Itay Kalfon"},"description":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Steps to repro."}]}]}}}`
		got, err := renderIssue([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		want := "# Fix login\n\nSteps to repro."
		if got != want {
			t.Errorf("got %q\nwant %q", got, want)
		}
	})

	t.Run("null description", func(t *testing.T) {
		raw := `{"fields":{"summary":"Backup timeouts","issuetype":{"name":"Bug"},"status":{"name":"To Do"},"assignee":null,"description":null}}`
		got, err := renderIssue([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		want := "# Backup timeouts\n\n_No description_"
		if got != want {
			t.Errorf("got %q\nwant %q", got, want)
		}
	})
}
