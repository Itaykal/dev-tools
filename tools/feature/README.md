# feature

Pick one of your open issues (or create a new one) in a TUI, then branch from
it (`git checkout -b KEY-slug-of-summary`). Invoked via the `feature` shell
function (`functions/feature.zsh`), which auto-builds this binary on demand.

## Layout

```
main.go                 wire config + provider + TUI, then git checkout
internal/
  tracker/   the swap point — Issue + Provider interface, no Jira/UI deps
  jira/      Provider impl: shells out to the `jira` CLI, parses --raw, ADF→md
  config/    TOML config + defaults (see feature.example.toml)
  tui/       Bubble Tea picker: fuzzy list, type aliases, async preview, create
  vcs/       branch slug + git checkout
```

## Swapping the tracker

`internal/tracker` defines everything the app needs:

```go
type Provider interface {
    List(ctx) ([]Issue, error)
    Describe(ctx, key) (markdown string, error)
    Create(ctx, CreateRequest) (key string, error)
}
```

To move off Jira, write a new package that satisfies `tracker.Provider` (e.g. a
`gh`-backed one), then change the one line in `main.go` that does
`jira.New(cfg)`. Nothing in `tui/` knows the tracker exists.

## Config

Copy `feature.example.toml` to `~/.config/feature/config.toml` (or point
`$FEATURE_CONFIG` at it). Controls the assignee, list filters, create defaults
(custom fields, the status to move to), and the type-alias map. With no file,
built-in defaults mirror the original zsh tool.

## Develop

```
go build ./...
go test ./...
```

The `feature` wrapper rebuilds `bin/feature` whenever a `.go` file or `go.mod`
is newer than the binary, so just edit and run `feature`.
