# rust tools

Standalone Rust CLIs for the dotfiles, replacing the more complex shell
functions. Each tool is a self-contained binary on `PATH` (no zsh wrapper); the
shell only reflects state into the prompt where a binary can't.

## Layout

```
tools/                this directory IS the Cargo workspace
  Cargo.toml          workspace + shared dependency versions
  Makefile            build / dev / test / fmt / clippy
  crates/
    common/           shared lib: theme, fuzzy picker, config + cache loaders,
                      term guard, spinner, background Refresh (ctrl-r / SWR)
    aws-switch/        bin: pick an AWS SSO account+role → write ~/.aws/config
    feature/           bin: pick a Jira issue (or create one) → git checkout -b
  bin/                built binaries, symlinked here (gitignored, on PATH via vars/path.zsh)
```

## Build

```
make build    # release build, link binaries into bin/ (what install.sh runs)
make dev      # fast debug build, linked into bin/ (the dev loop)
make test     # cargo test
make clippy   # cargo clippy -D warnings
```

There is no build-on-invocation: edit, `make dev`, run. `install.sh` runs
`make build` once.

## Adding a tool

1. `cargo new --bin crates/<tool>` and add it to `members` in `Cargo.toml` and
   `TOOLS` in the `Makefile`.
2. Reuse `common`: `common::select(...)` for a simple fuzzy picker,
   `common::config::load(...)` for `~/.config/<tool>/config.toml` (+ `$<TOOL>_CONFIG`
   + `--config`), `common::cache` for a TTL on-disk cache, `common::spinner` for a
   loading spinner, `common::refresh::Refresh` for background/`ctrl-r` refresh, and
   `common::theme` for the fuchsia palette. `feature` shows the richer pattern: a
   bespoke two-pane TUI (list + Markdown preview) built from these primitives.
3. If the tool only needs to *persist* state (not mutate the live shell), write
   a file and let an existing zsh prompt hook read it — see how `aws-switch`
   writes `~/.aws/config` and `functions/aws-sync-prompt.zsh` exports from it.

## Conventions

- **Standalone, no wrappers.** Binaries go on `PATH`; don't add zsh functions
  that wrap them.
- **Shell state stays in zsh.** A child process can't `export` into the parent
  shell or drive the prompt. Persist to a file the shell already reads.
- **Config via `common::config`** (XDG `~/.config/<tool>/config.toml`), with a
  `#[serde(default)]` struct + real `Default` so partial files layer over defaults.
- **Pickers via `common::select`** so every tool shares the look and key map.
