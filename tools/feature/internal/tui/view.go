package tui

import (
	"fmt"
	"sort"
	"strings"

	"feature/internal/tracker"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Column widths inside the left pane (summary flexes to fill the rest).
const (
	ptrW    = 2
	typeW   = 9
	keyW    = 11
	statusW = 14
)

// layout computes pane sizes and (re)sizes the preview viewport/renderer.
func (m *Model) layout() {
	leftW, rightW := m.paneWidths()
	rightInner := rightW - 4
	if rightInner < 1 {
		rightInner = 1
	}
	vpH := m.innerHeight() - 1 // 1 line for the in-pane label
	if vpH < 1 {
		vpH = 1
	}
	m.preview.setSize(rightInner, vpH)
	m.input.Width = leftW - 4 - 3 // pane minus border/padding minus prompt
}

func (m *Model) paneWidths() (left, right int) {
	left = m.width * 60 / 100
	if left < 30 {
		left = min(30, m.width)
	}
	right = m.width - left
	if right < 0 {
		right = 0
	}
	return left, right
}

// pickerHeight is the inline height the picker occupies — a fraction of the
// terminal (à la fzf's --height) rather than the whole screen, with a usable
// minimum so the preview pane has room.
func (m *Model) pickerHeight() int {
	h := m.height * 40 / 100
	if h < 16 {
		h = 16
	}
	if h > m.height {
		h = m.height
	}
	return h
}

// innerHeight is the height inside a pane's rounded border.
func (m *Model) innerHeight() int {
	h := m.pickerHeight() - 1 /*footer*/ - 1 /*meta line*/ - 2 /*top+bottom border*/
	if h < 1 {
		h = 1
	}
	return h
}

// listRows is how many issue rows fit (inner height minus the query line).
func (m *Model) listRows() int {
	r := m.innerHeight() - 1
	if r < 1 {
		r = 1
	}
	return r
}

func (m *Model) View() string {
	// On exit, render nothing so Bubble Tea clears the inline frame (like fzf
	// vanishing on accept/cancel) and git output starts on a clean line.
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "  " + m.spinner.View() + " loading…"
	}

	left := m.leftPane()
	right := m.rightPane()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return body + "\n" + m.metaLine() + "\n" + m.footer()
}

// metaLine renders the current issue's type/status/assignee, greyed and
// indented to sit just under the preview pane's border.
func (m *Model) metaLine() string {
	iss := m.currentIssue()
	if iss == nil || m.showHelp {
		return ""
	}
	parts := []string{string(iss.Type), iss.Status}
	if iss.Assignee != "" {
		parts = append(parts, iss.Assignee)
	}
	leftW, _ := m.paneWidths()
	indent := strings.Repeat(" ", leftW+2) // align under the preview content
	return indent + mutedStyle.Render(strings.Join(parts, "  •  "))
}

// leftPane renders the query line and issue rows.
func (m *Model) leftPane() string {
	leftW, _ := m.paneWidths()
	inner := leftW - 4
	var b strings.Builder

	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(m.rowsBlock(inner))

	// Width(inner+2): lipgloss Width is the inter-border region (padding
	// included); +2 covers the horizontal padding so the text area is `inner`.
	return leftFrame.Width(inner + 2).Height(m.innerHeight()).Render(b.String())
}

// rowsBlock renders the visible window of filtered issue rows.
func (m *Model) rowsBlock(inner int) string {
	rows := m.listRows()

	if m.loading {
		return m.spinner.View() + " fetching issues…"
	}
	if len(m.filtered) == 0 {
		return dimStyle.Render("  no matching issues")
	}

	var lines []string
	end := min(m.offset+rows, len(m.filtered))
	for i := m.offset; i < end; i++ {
		row := m.filtered[i]
		lines = append(lines, m.renderRow(m.issues[row.issueIdx], i == m.cursor, inner, row.sumMatched))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderRow(iss tracker.Issue, selected bool, inner int, matched []int) string {
	sumW := inner - (ptrW + typeW + keyW + statusW + 3)
	if sumW < 5 {
		sumW = 5
	}

	typ := pad(string(iss.Type), typeW)
	key := pad(iss.Key, keyW)
	sumText := truncate(iss.Summary, sumW)
	st := pad(truncate(iss.Status, statusW), statusW)

	if selected {
		// Whole row painted on the selection background (the selection itself is
		// the emphasis, so no per-letter highlight here).
		line := fmt.Sprintf("▸ %s %s %s %s", typ, key, pad(sumText, sumW), st)
		return rowSelStyle.Width(inner).Render(line)
	}

	// Highlight the matched letters in the summary; pad with plain spaces after
	// styling so column widths stay exact (lipgloss styling adds zero-width ANSI).
	sum := styleMatches(sumText, matched, rowStyle, matchStyle)
	if gap := sumW - runewidth.StringWidth(sumText); gap > 0 {
		sum += strings.Repeat(" ", gap)
	}
	line := fmt.Sprintf("  %s %s %s %s",
		typeStyle.Render(typ), keyStyle.Render(key), sum, statusStyle.Render(st))
	return line
}

// styleMatches renders text with the matched rune positions in `match` style
// and the rest in `base`, coalescing runs to keep the ANSI minimal.
func styleMatches(text string, matched []int, base, match lipgloss.Style) string {
	if len(matched) == 0 {
		return base.Render(text)
	}
	set := make(map[int]bool, len(matched))
	for _, p := range matched {
		set[p] = true
	}
	runes := []rune(text)
	var b strings.Builder
	for i := 0; i < len(runes); {
		on := set[i]
		j := i
		for j < len(runes) && set[j] == on {
			j++
		}
		seg := string(runes[i:j])
		if on {
			b.WriteString(match.Render(seg))
		} else {
			b.WriteString(base.Render(seg))
		}
		i = j
	}
	return b.String()
}

// rightPane renders the preview (or cheatsheet) with an accent label.
func (m *Model) rightPane() string {
	_, rightW := m.paneWidths()
	inner := rightW - 4
	label := previewLabelStyle.Render(" issue ")
	if m.showHelp {
		label = previewLabelStyle.Render(" help ")
	}
	content := label + "\n" + m.preview.vp.View()
	return previewFrame.Width(inner + 2).Height(m.innerHeight()).Render(content)
}

// footer is the keybar / status line.
func (m *Model) footer() string {
	if m.status != "" {
		return lipgloss.NewStyle().Foreground(accent).Render("  " + m.status)
	}
	if m.creating {
		return "  " + m.spinner.View() + mutedStyle.Render(" creating issue…")
	}
	sep := footerSep.Render("  ·  ")
	keys := []string{
		footerKey.Render("↑↓") + footerLabel.Render(" move"),
		footerKey.Render("enter") + footerLabel.Render(" branch"),
		footerKey.Render("ctrl-n") + footerLabel.Render(" new"),
		footerKey.Render("/b /t /s /st") + footerLabel.Render(" type"),
		footerKey.Render("?") + footerLabel.Render(" help"),
		footerKey.Render("esc") + footerLabel.Render(" quit"),
	}
	return "  " + strings.Join(keys, sep)
}

// renderHelp builds the help panel: what the tool does, the keys, and the type
// aliases. Shown in the preview pane when the user presses '?'.
func renderHelp(aliases map[string]string) string {
	var b strings.Builder

	row := func(k, desc string) {
		b.WriteString("  " + helpKeyStyle.Render(pad(k, 13)) + mutedStyle.Render(desc) + "\n")
	}
	section := func(title string) {
		b.WriteString("\n" + sectionStyle.Render(title) + "\n")
	}

	b.WriteString(mutedStyle.Render("Pick an issue and branch from it, or") + "\n")
	b.WriteString(mutedStyle.Render("create a new one with ctrl-n.") + "\n")

	section("navigate")
	row("↑ / ctrl-k", "move up")
	row("↓ / ctrl-j", "move down")
	row("enter", "branch from issue")
	row("ctrl-n", "new issue → branch")
	row("esc / ctrl-c", "quit")

	section("preview (vim)")
	row("ctrl-d / ctrl-u", "scroll ½ page down / up")
	row("ctrl-f / ctrl-b", "scroll page down / up")

	section("type filter")
	order := []string{"Bug", "Task", "Story", "Sub-task"}
	for _, typ := range order {
		var prefixes []string
		for p, t := range aliases {
			if t == typ {
				prefixes = append(prefixes, "/"+p)
			}
		}
		if len(prefixes) == 0 {
			continue
		}
		row(strings.Join(sortShort(prefixes), " "), typ)
	}
	b.WriteString("  " + dimStyle.Render("prefix + text, e.g. /b login") + "\n")

	b.WriteString("\n" + dimStyle.Render("?  toggle this help"))
	return b.String()
}

// --- small helpers ---

// truncate shortens s to w display columns, adding an ellipsis when cut.
func truncate(s string, w int) string {
	if runewidth.StringWidth(s) <= w {
		return s
	}
	if w <= 1 {
		return runewidth.Truncate(s, w, "")
	}
	return runewidth.Truncate(s, w, "…")
}

// pad right-pads s with spaces to w display columns.
func pad(s string, w int) string {
	gap := w - runewidth.StringWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// sortShort sorts alias prefixes by length then lexically ("/b" before "/bug").
func sortShort(ss []string) []string {
	sort.Slice(ss, func(i, j int) bool {
		if len(ss[i]) != len(ss[j]) {
			return len(ss[i]) < len(ss[j])
		}
		return ss[i] < ss[j]
	})
	return ss
}
