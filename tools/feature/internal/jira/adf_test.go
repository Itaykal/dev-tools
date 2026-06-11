package jira

import (
	"encoding/json"
	"testing"
)

// decode parses an ADF JSON snippet the way Describe would receive it.
func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad test JSON: %v", err)
	}
	return v
}

func TestAdfToMarkdown(t *testing.T) {
	cases := []struct {
		name string
		adf  string
		want string
	}{
		{
			name: "nil description",
			adf:  `null`,
			want: "",
		},
		{
			name: "single paragraph",
			adf:  `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`,
			want: "Hello world",
		},
		{
			name: "heading then paragraph",
			adf:  `{"type":"doc","content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Goals"}]},{"type":"paragraph","content":[{"type":"text","text":"Body."}]}]}`,
			want: "## Goals\n\nBody.",
		},
		{
			name: "bullet list",
			adf:  `{"type":"doc","content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"two"}]}]}]}]}`,
			want: "- one\n- two",
		},
		{
			name: "ordered list",
			adf:  `{"type":"doc","content":[{"type":"orderedList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"second"}]}]}]}]}`,
			want: "1. first\n2. second",
		},
		{
			name: "marks",
			adf:  `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"bold","marks":[{"type":"strong"}]},{"type":"text","text":" and "},{"type":"text","text":"code","marks":[{"type":"code"}]}]}]}`,
			want: "**bold** and `code`",
		},
		{
			name: "link mark",
			adf:  `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"site","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]}]}]}`,
			want: "[site](https://example.com)",
		},
		{
			name: "code block",
			adf:  `{"type":"doc","content":[{"type":"codeBlock","content":[{"type":"text","text":"go build ./..."}]}]}`,
			want: "```\ngo build ./...\n```",
		},
		{
			name: "rule between paragraphs",
			adf:  `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]},{"type":"rule"},{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}`,
			want: "a\n\n\n---\n\nb",
		},
		{
			name: "mention and inline card",
			adf:  `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"text":"@Itay"}},{"type":"text","text":" see "},{"type":"inlineCard","attrs":{"url":"https://jira/X-1"}}]}]}`,
			want: "@Itay see https://jira/X-1",
		},
		{
			name: "hard break",
			adf:  `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"line1"},{"type":"hardBreak"},{"type":"text","text":"line2"}]}]}`,
			want: "line1\nline2",
		},
		{
			name: "unknown node descends into children",
			adf:  `{"type":"doc","content":[{"type":"panel","content":[{"type":"paragraph","content":[{"type":"text","text":"inside panel"}]}]}]}`,
			want: "inside panel",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := adfToMarkdown(decode(t, tc.adf))
			if got != tc.want {
				t.Errorf("adfToMarkdown mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
