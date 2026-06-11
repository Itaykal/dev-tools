package tui

import (
	"testing"

	"feature/internal/tracker"
)

func aliases() map[string]string {
	return map[string]string{
		"b": "Bug", "bug": "Bug",
		"t":  "Task",
		"s":  "Story",
		"st": "Sub-task", "sub": "Sub-task",
	}
}

func TestParseQuery(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		wantType   tracker.IssueType
		wantSearch string
	}{
		{"plain text", "login", "", "login"},
		{"empty", "", "", ""},
		{"bare alias no space", "/b", tracker.TypeBug, ""},
		{"alias with space only", "/b ", tracker.TypeBug, ""},
		{"alias with search", "/b login", tracker.TypeBug, "login"},
		{"long alias", "/sub thing", tracker.TypeSubtask, "thing"},
		{"st before s does not collide", "/st x", tracker.TypeSubtask, "x"},
		{"unknown alias is literal", "/x foo", "", "/x foo"},
		{"slash with no alias char", "/ foo", "", "/ foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseQuery(tc.raw, aliases())
			if got.activeType != tc.wantType || got.search != tc.wantSearch {
				t.Errorf("parseQuery(%q) = {%q, %q}, want {%q, %q}",
					tc.raw, got.activeType, got.search, tc.wantType, tc.wantSearch)
			}
		})
	}
}

func TestFilterIssues(t *testing.T) {
	issues := []tracker.Issue{
		{Key: "DRM-1", Summary: "Fix login redirect", Type: tracker.TypeBug},
		{Key: "DRM-2", Summary: "Add SSO login", Type: tracker.TypeTask},
		{Key: "DRM-3", Summary: "Login rate limit", Type: tracker.TypeStory},
		{Key: "DRM-4", Summary: "Unrelated cleanup", Type: tracker.TypeBug},
	}

	idxOf := func(rows []rowMatch) []int {
		out := make([]int, len(rows))
		for i, r := range rows {
			out[i] = r.issueIdx
		}
		return out
	}

	t.Run("type filter only keeps original order", func(t *testing.T) {
		got := idxOf(filterIssues(issues, parsedQuery{activeType: tracker.TypeBug}))
		if !equalInts(got, []int{0, 3}) {
			t.Errorf("got %v, want [0 3]", got)
		}
	})

	t.Run("search ranks matches", func(t *testing.T) {
		got := idxOf(filterIssues(issues, parsedQuery{search: "login"}))
		if len(got) != 3 {
			t.Fatalf("got %d matches, want 3: %v", len(got), got)
		}
		for _, idx := range got {
			if idx == 3 {
				t.Errorf("unrelated issue should not match 'login'")
			}
		}
	})

	t.Run("type filter plus search", func(t *testing.T) {
		got := idxOf(filterIssues(issues, parsedQuery{activeType: tracker.TypeBug, search: "login"}))
		if !equalInts(got, []int{0}) {
			t.Errorf("got %v, want [0]", got)
		}
	})

	t.Run("match positions map into summary coords", func(t *testing.T) {
		// "DRM-3" / "Login rate limit": searching "login" matches summary
		// positions 0..4, not the key.
		rows := filterIssues(issues, parsedQuery{activeType: tracker.TypeStory, search: "login"})
		if len(rows) != 1 {
			t.Fatalf("got %d rows, want 1", len(rows))
		}
		if !equalInts(rows[0].sumMatched, []int{0, 1, 2, 3, 4}) {
			t.Errorf("sumMatched = %v, want [0 1 2 3 4]", rows[0].sumMatched)
		}
	})
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
