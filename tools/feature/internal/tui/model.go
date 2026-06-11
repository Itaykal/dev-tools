// Package tui is the Bubble Tea picker: a fuzzy-filterable issue list with a
// live Markdown preview, type-alias filtering, and an inline create flow. It
// depends only on tracker.Provider, never on Jira directly.
package tui

import (
	"context"
	"strings"

	"feature/internal/config"
	"feature/internal/tracker"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Result is what the picker hands back to main: the issue key to branch from
// and the summary to slug the branch name from.
type Result struct {
	Key     string
	Summary string
}

// Model is the root Bubble Tea model.
type Model struct {
	ctx      context.Context
	provider tracker.Provider
	aliases  map[string]string

	issues   []tracker.Issue
	filtered []rowMatch // filtered/ranked rows, with match positions
	cursor   int        // index into filtered
	offset   int        // first visible filtered row (scroll)

	input   textinput.Model // query line
	preview preview
	spinner spinner.Model

	showHelp bool

	loading  bool // initial list fetch in flight
	creating bool // create request in flight
	status   string

	width, height int
	ready         bool
	quitting      bool

	result *Result
}

// New builds a Model wired to a provider and config.
func New(ctx context.Context, p tracker.Provider, cfg config.Config) *Model {
	in := textinput.New()
	in.Prompt = "❯ "
	in.PromptStyle = promptStyle
	in.Cursor.Style = promptStyle
	in.Placeholder = "filter issues…  (/b /t /s /st to filter by type)"
	in.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = accentStyle

	return &Model{
		ctx:      ctx,
		provider: p,
		aliases:  cfg.Aliases,
		input:    in,
		preview:  newPreview(),
		spinner:  sp,
		loading:  true,
	}
}

// Result returns the user's selection, or nil if they quit without choosing.
func (m *Model) Result() *Result { return m.result }

type issuesLoadedMsg struct {
	issues []tracker.Issue
	err    error
}

type createdMsg struct {
	key     string
	summary string
	err     error
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.listCmd(), m.spinner.Tick, textinput.Blink)
}

func (m *Model) listCmd() tea.Cmd {
	return func() tea.Msg {
		issues, err := m.provider.List(m.ctx)
		return issuesLoadedMsg{issues: issues, err: err}
	}
}

func (m *Model) createCmd(typ tracker.IssueType, summary string) tea.Cmd {
	return func() tea.Msg {
		key, err := m.provider.Create(m.ctx, tracker.CreateRequest{Type: typ, Summary: summary})
		return createdMsg{key: key, summary: summary, err: err}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		return m, m.previewCmdForCursor()

	case issuesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "list failed: " + msg.err.Error()
			return m, nil
		}
		m.issues = msg.issues
		m.applyFilter()
		return m, m.previewCmdForCursor()

	case descLoadedMsg:
		body := msg.markdown
		if msg.err != nil {
			body = "_failed to load " + msg.key + "_\n\n" + msg.err.Error()
		}
		m.preview.cache[msg.key] = body
		if !m.showHelp && msg.key == m.currentKey() {
			m.preview.show(body)
		}
		return m, nil

	case createdMsg:
		m.creating = false
		if msg.err != nil {
			m.status = "create failed: " + msg.err.Error()
			return m, nil
		}
		m.result = &Result{Key: msg.key, Summary: msg.summary}
		m.quitting = true
		return m, tea.Quit

	case spinner.TickMsg:
		if m.loading || m.creating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.updateList(msg)
	}
	return m, nil
}

// updateList handles keys in the picker.
func (m *Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While a create is in flight, swallow input except a hard quit.
	if m.creating {
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "esc":
		if m.showHelp {
			m.showHelp = false
			m.refreshPreview()
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "enter":
		if iss := m.currentIssue(); iss != nil {
			m.result = &Result{Key: iss.Key, Summary: iss.Summary}
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case "up", "ctrl+k":
		m.moveCursor(-1)
		return m, m.previewCmdForCursor()
	case "down", "ctrl+j":
		m.moveCursor(1)
		return m, m.previewCmdForCursor()

	// Preview scrolling — vim's scroll keys.
	case "ctrl+d":
		m.preview.vp.HalfPageDown()
		return m, nil
	case "ctrl+u":
		m.preview.vp.HalfPageUp()
		return m, nil
	case "ctrl+f":
		m.preview.vp.PageDown()
		return m, nil
	case "ctrl+b":
		m.preview.vp.PageUp()
		return m, nil

	case "ctrl+n":
		return m.createFromQuery()

	case "?":
		if m.input.Value() == "" {
			m.toggleHelp()
			return m, nil
		}
		// otherwise fall through to literal input
	}

	// Default: edit the query, then re-filter.
	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.applyFilter()
		return m, tea.Batch(cmd, m.previewCmdForCursor())
	}
	return m, cmd
}

// createFromQuery creates an issue immediately from the current query: the
// alias-stripped text is the summary and the active alias is the type
// (defaulting to Task). The provider also moves it to the configured status.
// No prompt — ctrl-n goes straight to create + branch.
func (m *Model) createFromQuery() (tea.Model, tea.Cmd) {
	q := parseQuery(m.input.Value(), m.aliases)
	summary := strings.TrimSpace(q.search)
	if summary == "" {
		m.status = "type a summary in the query first, then ctrl-n"
		return m, nil
	}
	typ := q.activeType
	if typ == "" {
		typ = tracker.TypeTask
	}
	if typ == tracker.TypeSubtask {
		m.status = "cannot create a Sub-task without a parent"
		return m, nil
	}
	m.creating = true
	m.status = ""
	return m, tea.Batch(m.createCmd(typ, summary), m.spinner.Tick)
}

func (m *Model) toggleHelp() {
	m.showHelp = !m.showHelp
	if m.showHelp {
		m.preview.showPlain(renderHelp(m.aliases))
	} else {
		m.refreshPreview()
	}
}

// applyFilter recomputes the filtered list from the current query and clamps
// the cursor/scroll.
func (m *Model) applyFilter() {
	q := parseQuery(m.input.Value(), m.aliases)
	m.filtered = filterIssues(m.issues, q)
	m.cursor = 0
	m.offset = 0
}

func (m *Model) moveCursor(d int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += d
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	// Keep the cursor within the visible window.
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if rows := m.listRows(); rows > 0 && m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
}

func (m *Model) currentIssue() *tracker.Issue {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	iss := m.issues[m.filtered[m.cursor].issueIdx]
	return &iss
}

func (m *Model) currentKey() string {
	if iss := m.currentIssue(); iss != nil {
		return iss.Key
	}
	return ""
}

// previewCmdForCursor shows the cached preview (or a placeholder) and, if not
// cached, returns a Cmd to fetch it.
func (m *Model) previewCmdForCursor() tea.Cmd {
	if m.showHelp {
		return nil
	}
	key := m.currentKey()
	if key == "" {
		m.preview.showPlain(dimStyle.Render("  no issue selected"))
		return nil
	}
	if md, ok := m.preview.cache[key]; ok {
		m.preview.show(md)
		return nil
	}
	m.preview.showPlain(dimStyle.Render("  loading " + key + "…"))
	return m.loadDescCmd(key)
}

// refreshPreview re-renders the current cursor's preview (used after resize or
// closing the cheatsheet).
func (m *Model) refreshPreview() {
	_ = m.previewCmdForCursor() // ignore the fetch cmd; resize re-renders cached
}
