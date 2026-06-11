package tui

import (
	"context"
	"testing"

	"feature/internal/config"
	"feature/internal/tracker"
	tea "github.com/charmbracelet/bubbletea"
)

type fakeProvider struct{ issues []tracker.Issue }

func (f fakeProvider) List(context.Context) ([]tracker.Issue, error) { return f.issues, nil }
func (f fakeProvider) Describe(_ context.Context, key string) (string, error) {
	return "# " + key + " summary\n\nSome **bold** description with:\n\n- a bullet\n- another", nil
}
func (f fakeProvider) Create(_ context.Context, r tracker.CreateRequest) (string, error) {
	return "DRM-9999", nil
}

// TestRenderSnapshot drives the model through resize + load and prints the
// view so layout can be eyeballed (run with -v). Not an assertion test.
func TestRenderSnapshot(t *testing.T) {
	fp := fakeProvider{issues: []tracker.Issue{
		{Type: tracker.TypeBug, Key: "DRM-43930", Summary: "Dijo2 Backup timeouts", Status: "In Review"},
		{Type: tracker.TypeStory, Key: "DRM-43616", Summary: "Research ClickHouse best practices for event processing", Status: "Selected for Development"},
		{Type: tracker.TypeSubtask, Key: "DRM-42399", Summary: "Enable CD via Jenkins", Status: "In Review"},
		{Type: tracker.TypeBug, Key: "DRM-41468", Summary: "Postgresql-HA wastes 32GB of RAM", Status: "To Do"},
	}}
	m := New(context.Background(), fp, config.Default())

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 110, Height: 20})
	model, _ = model.Update(issuesLoadedMsg{issues: fp.issues})
	model, _ = model.Update(descLoadedMsg{key: "DRM-43930", markdown: mustDescribe(fp, "DRM-43930")})

	t.Log("\n" + model.View())
}

// TestRenderHelp drives the help panel so it is exercised (eyeball with -v).
func TestRenderHelp(t *testing.T) {
	fp := fakeProvider{issues: []tracker.Issue{
		{Type: tracker.TypeBug, Key: "DRM-1", Summary: "Fix login redirect", Status: "To Do"},
	}}
	m := New(context.Background(), fp, config.Default())
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	model, _ = model.Update(issuesLoadedMsg{issues: fp.issues})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	t.Log("\n--- help ---\n" + model.View())
}

// TestCreateInstant verifies ctrl-n creates from the query with no prompt.
func TestCreateInstant(t *testing.T) {
	fp := fakeProvider{issues: []tracker.Issue{
		{Type: tracker.TypeBug, Key: "DRM-1", Summary: "x", Status: "To Do"},
	}}

	t.Run("empty query sets a status, does not create", func(t *testing.T) {
		m := New(context.Background(), fp, config.Default())
		var model tea.Model = m
		model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
		model, _ = model.Update(issuesLoadedMsg{issues: fp.issues})
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
		mm := model.(*Model)
		if mm.creating {
			t.Error("should not be creating with an empty query")
		}
		if mm.status == "" {
			t.Error("expected a guidance status for empty-query ctrl-n")
		}
	})

	t.Run("typed query triggers create", func(t *testing.T) {
		m := New(context.Background(), fp, config.Default())
		var model tea.Model = m
		model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
		model, _ = model.Update(issuesLoadedMsg{issues: fp.issues})
		for _, r := range "/b login form" {
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
		if !model.(*Model).creating {
			t.Error("expected creating=true after ctrl-n with a query")
		}
	})
}

func mustDescribe(f fakeProvider, k string) string {
	s, _ := f.Describe(context.Background(), k)
	return s
}
