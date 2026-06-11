package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// themedMarkdown adapts Glamour's dark theme to the fzf fuchsia palette: the
// stock dark style renders H1 as yellow-on-blue and other headings cyan, which
// clashes. Recolor the headings to the accent (H1 keeps a filled bar, on the
// selection purple, to stay readable).
func themedMarkdown() ansi.StyleConfig {
	s := styles.DarkStyleConfig // value copy; we only reassign pointer fields
	fuchsia := "#f0abfc"
	ink := "#1c1420" // near-black with a purple tint, for text on the accent bar
	// H1 is a bold fuchsia bar with dark text so the title pops against the
	// muted body, while staying on-palette.
	s.H1.Color = &ink
	s.H1.BackgroundColor = &fuchsia
	s.Heading.Color = &fuchsia
	s.H2.Color = &fuchsia
	s.H3.Color = &fuchsia
	s.Link.Color = &fuchsia
	return s
}

// descLoadedMsg carries the result of an async provider.Describe call.
type descLoadedMsg struct {
	key      string
	markdown string
	err      error
}

// loadDescCmd fetches one issue's rendered Markdown off the UI goroutine so the
// list never stalls while a description is fetched.
func (m *Model) loadDescCmd(key string) tea.Cmd {
	return func() tea.Msg {
		md, err := m.provider.Describe(m.ctx, key)
		return descLoadedMsg{key: key, markdown: md, err: err}
	}
}

// preview owns the right-hand pane: a viewport plus a Glamour renderer sized to
// the pane, and a cache of rendered Markdown keyed by issue key.
type preview struct {
	vp       viewport.Model
	renderer *glamour.TermRenderer
	cache    map[string]string // issue key -> raw markdown (pre-glamour)
	width    int
}

func newPreview() preview {
	return preview{cache: map[string]string{}}
}

// setSize rebuilds the viewport and Glamour renderer for a new pane size.
func (p *preview) setSize(w, h int) {
	p.width = w
	p.vp = viewport.New(w, h)
	// WordWrap to the pane width; auto style adapts to the terminal background
	// and gives the modern, syntax-aware look.
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(themedMarkdown()),
		glamour.WithWordWrap(w),
	)
	if err == nil {
		p.renderer = r
	}
}

// show renders markdown into the viewport (falling back to raw on any error).
func (p *preview) show(markdown string) {
	content := markdown
	if p.renderer != nil {
		if out, err := p.renderer.Render(markdown); err == nil {
			content = out
		}
	}
	p.vp.SetContent(content)
	p.vp.GotoTop()
}

// showPlain sets unrendered text (used for "loading…" / error placeholders).
func (p *preview) showPlain(s string) {
	p.vp.SetContent(s)
	p.vp.GotoTop()
}
