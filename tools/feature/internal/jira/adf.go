package jira

import (
	"encoding/json"
	"strings"
)

// adfToMarkdown walks an Atlassian Document Format (ADF) tree — Jira's rich
// description format — and flattens it to Markdown. It mirrors the node
// handling of the old feature-adf.jq, but emits real Markdown (## headings,
// fenced code, - bullets, **bold**) so a Markdown renderer can style it richly.
//
// The input is the JSON value of `fields.description`, already decoded into
// the usual map[string]any / []any shape. A nil input (the common
// null-description case) yields the empty string; callers substitute a
// placeholder.
func adfToMarkdown(node any) string {
	return strings.TrimRight(walk(node), "\n")
}

func walk(node any) string {
	n, ok := node.(map[string]any)
	if !ok {
		return ""
	}

	switch str(n["type"]) {
	case "text":
		return applyMarks(str(n["text"]), n["marks"])
	case "hardBreak":
		return "\n"
	case "paragraph":
		return children(n) + "\n\n"
	case "heading":
		level := 1
		if attrs, ok := n["attrs"].(map[string]any); ok {
			if l, ok := attrs["level"].(float64); ok && l >= 1 && l <= 6 {
				level = int(l)
			}
		}
		return strings.Repeat("#", level) + " " + children(n) + "\n\n"
	case "bulletList":
		return list(n, func(int) string { return "- " })
	case "orderedList":
		return list(n, func(i int) string { return itoa(i+1) + ". " })
	case "listItem":
		// List items hold block content (usually a paragraph); trim the block
		// spacing so each item is a single line under its marker.
		return strings.TrimSpace(children(n))
	case "codeBlock":
		return "```\n" + strings.TrimRight(children(n), "\n") + "\n```\n\n"
	case "blockquote":
		return "> " + strings.TrimSpace(children(n)) + "\n\n"
	case "inlineCard":
		if attrs, ok := n["attrs"].(map[string]any); ok {
			return str(attrs["url"])
		}
		return ""
	case "mention":
		if attrs, ok := n["attrs"].(map[string]any); ok {
			if t := str(attrs["text"]); t != "" {
				return t
			}
		}
		return "@?"
	case "rule":
		return "\n---\n\n"
	default:
		// doc and any unknown node: just descend into children.
		return children(n)
	}
}

// list renders an ordered/bullet list, prefixing each item with marker(i).
func list(n map[string]any, marker func(i int) string) string {
	content, _ := n["content"].([]any)
	var b strings.Builder
	for i, item := range content {
		b.WriteString(marker(i))
		b.WriteString(walk(item))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// children joins the rendering of every child node in order.
func children(n map[string]any) string {
	content, ok := n["content"].([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, c := range content {
		b.WriteString(walk(c))
	}
	return b.String()
}

// applyMarks wraps text in Markdown emphasis for the ADF marks it carries.
func applyMarks(text string, raw any) string {
	marks, ok := raw.([]any)
	if !ok || text == "" {
		return text
	}
	for _, m := range marks {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		switch str(mm["type"]) {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "link":
			href := ""
			if attrs, ok := mm["attrs"].(map[string]any); ok {
				href = str(attrs["href"])
			}
			if href != "" {
				text = "[" + text + "](" + href + ")"
			}
		}
	}
	return text
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
