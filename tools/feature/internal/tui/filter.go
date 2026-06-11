package tui

import (
	"strings"

	"feature/internal/tracker"
	"github.com/sahilm/fuzzy"
)

// parsedQuery is the query line split into an explicit type filter and the
// remaining fuzzy search text. Decoupling these (instead of the old
// "^Bug ..."-in-the-query hack) is what makes the type aliases clean.
type parsedQuery struct {
	activeType tracker.IssueType // empty = no type filter
	search     string
}

// parseQuery interprets a leading "/<alias>" token against the alias map.
// "/b" or "/b login" filters to the aliased type (search = "login"); an
// unknown "/x ..." is treated as literal fuzzy text.
func parseQuery(raw string, aliases map[string]string) parsedQuery {
	if !strings.HasPrefix(raw, "/") {
		return parsedQuery{search: raw}
	}
	body := raw[1:]
	token, rest, hasSpace := cut(body, " ")
	typeName, ok := aliases[token]
	if !ok {
		// Unknown alias — don't swallow the text, just search literally.
		return parsedQuery{search: raw}
	}
	search := ""
	if hasSpace {
		search = rest
	}
	return parsedQuery{activeType: tracker.IssueType(typeName), search: strings.TrimSpace(search)}
}

// cut is strings.Cut (kept local for clarity of intent).
func cut(s, sep string) (before, after string, found bool) {
	return strings.Cut(s, sep)
}

// rowMatch is one row in the filtered list: an index into the original issues
// slice plus the rune positions (within the summary) that matched the search,
// so the row can highlight them.
type rowMatch struct {
	issueIdx   int
	sumMatched []int
}

// filterIssues applies the type filter then fuzzy-ranks by the search text.
// With no search text, the type-filtered list is returned in its original
// (server) order with no match positions.
func filterIssues(issues []tracker.Issue, q parsedQuery) []rowMatch {
	// Type filter first.
	candidates := make([]int, 0, len(issues))
	for i, iss := range issues {
		if q.activeType == "" || iss.Type == q.activeType {
			candidates = append(candidates, i)
		}
	}
	if q.search == "" {
		out := make([]rowMatch, len(candidates))
		for j, idx := range candidates {
			out[j] = rowMatch{issueIdx: idx}
		}
		return out
	}

	// Fuzzy rank the candidates by "KEY summary". Matched positions land in
	// that combined string; map the ones past "KEY " back to summary offsets.
	src := haystack(make([]string, len(candidates)))
	keyLens := make([]int, len(candidates))
	for j, idx := range candidates {
		src[j] = issues[idx].Key + " " + issues[idx].Summary
		keyLens[j] = len(issues[idx].Key)
	}
	matches := fuzzy.FindFrom(q.search, src)
	out := make([]rowMatch, 0, len(matches))
	for _, m := range matches {
		base := keyLens[m.Index] + 1 // skip "KEY "
		var sm []int
		for _, pos := range m.MatchedIndexes {
			if pos >= base {
				sm = append(sm, pos-base)
			}
		}
		out = append(out, rowMatch{issueIdx: candidates[m.Index], sumMatched: sm})
	}
	return out
}

// haystack adapts a []string to fuzzy.Source.
type haystack []string

func (h haystack) String(i int) string { return h[i] }
func (h haystack) Len() int            { return len(h) }
