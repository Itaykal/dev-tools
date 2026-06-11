// Package tracker defines the tracker-agnostic domain types and the Provider
// interface that any issue tracker (Jira today, something else tomorrow) must
// implement. It has zero dependencies on Jira or the TUI on purpose: this is
// the seam you swap to change backends.
package tracker

import "context"

// IssueType is a normalized issue category. Concrete providers map their own
// type names onto these (or pass them through, in Jira's case).
type IssueType string

const (
	TypeBug     IssueType = "Bug"
	TypeTask    IssueType = "Task"
	TypeStory   IssueType = "Story"
	TypeSubtask IssueType = "Sub-task"
)

// Issue is one row in the picker. List() populates the cheap fields; the long
// description is fetched lazily via Describe() so listing stays fast.
type Issue struct {
	Key      string
	Summary  string // full, untruncated — the TUI truncates only for display
	Status   string
	Assignee string
	Type     IssueType
}

// CreateRequest is the minimal, user-supplied part of creating an issue. The
// provider fills in the rest (assignee, custom fields, the status to move to)
// from its own configuration.
type CreateRequest struct {
	Type    IssueType
	Summary string
}

// Provider is the single seam between the app and a concrete issue tracker.
type Provider interface {
	// List returns the user's open issues (cheap fields only; Description is
	// fetched separately via Describe).
	List(ctx context.Context) ([]Issue, error)

	// Describe returns a rendered Markdown view of one issue, ready to hand to
	// a Markdown renderer. The caller never sees tracker-specific formats.
	Describe(ctx context.Context, key string) (string, error)

	// Create creates an issue, applies the provider's create defaults, performs
	// any best-effort post-create transition, and returns the new issue key.
	Create(ctx context.Context, req CreateRequest) (key string, err error)
}
