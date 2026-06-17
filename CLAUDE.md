# kalfon-dotfiles

Personal zsh dotfiles repo. All shell customisations live here — never edit `~/.p10k.zsh`, `~/.zshrc` aliases, or similar files directly.

## Structure

```
.entry              # dynamic loader — sources all *.zsh from each dir in order
init/               # initialization scripts that must run before everything else (third-party integrations: fzf, zoxide, direnv, …)
vars/               # exported env vars
aliases/            # aliases only, one file per tool (no functions)
functions/          # shell functions, one file per function
p10k/               # Powerlevel10k overrides (files are numbered for load order)
k9s/                # k9s config (views.yaml, …) — symlinked into ~/Library/Application Support/k9s/
macos/              # macOS-only artifacts (LaunchAgents, AppleScripts) — not sourced by .entry
```

## Rules

- **All changes go in this repo.** Never patch `~/.p10k.zsh` or `~/.zshrc` directly.
- New aliases → new file `aliases/<tool>.zsh`. New functions → new file `functions/<name>.zsh`.
- Aliases and functions must not be mixed in the same file.
- `init/` is for anything that must run before the rest of the dotfiles load — typically `source` lines for third-party shell integrations (fzf key-bindings, zoxide, direnv hooks). Never put those `source` lines in `aliases/` or `functions/`.
- `.entry` dynamically sources `*.zsh` from each category dir in order: `init → vars → aliases → functions → p10k`.
- `p10k/` files are numbered (`1-`, `2-`, …) because load order matters: kubernetes layout must be set before the AWS block inserts relative to it.
- `.p10k` overrides are sourced after `~/.p10k.zsh`, so they win. Use array manipulation to reorder prompt elements rather than redefining the full array.
- The AWS SSO session is always named `session`. The active profile is always `default`. Do not introduce new profile names.
- `AWS_ACCOUNT_NAME`, `AWS_ACCOUNT_ID`, `AWS_ROLE_NAME`, `AWS_DEFAULT_REGION` are exported by `aws-switch` and referenced in the p10k content expansion — keep them in sync if either side changes.
- `jq` and `fzf` are assumed to be installed. Do not add fallbacks for them.
- `k9s/` is not sourced by `.entry` (it's not shell). Files there are symlinked into `~/Library/Application Support/k9s/` (e.g. `views.yaml`). Edit only via this repo. Bad column expressions in `views.yaml` fail silently in the UI — check `~/Library/Application Support/k9s/k9s.log` after changes.
- `macos/` is also not sourced by `.entry`. Each subdir is a self-contained macOS artifact. `install.sh` symlinks LaunchAgent plists into `~/Library/LaunchAgents/` and (re)bootstraps them via `launchctl bootstrap gui/$(id -u)`. Agents that drive UI via System Events need Accessibility permission granted to `/usr/bin/osascript` — macOS will prompt on first run.

## Releasing the Rust tools (`tools/`)

The Rust CLIs under `tools/` (`feature`, `aws-switch`, …) ship from **one tag → one release**, but each tool is a **standalone package** (its own tarball asset). One version covers all tools.

To cut a release, push a `v<version>` tag:

```sh
git tag v0.2.0
git push origin v0.2.0
```

What happens: `.github/workflows/release.yml` triggers on `v*` tags, builds the whole workspace (`cargo build --release` on a `macos-14`/`aarch64-apple-darwin` runner), then packages **every** `[[bin]]` crate under `tools/crates/` into its own `<tool>-aarch64-apple-darwin.tar.gz` and attaches all of them to a single GitHub Release for that tag.

**New tools need no workflow edits** — add the binary crate under `tools/crates/<tool>/`; the packaging loop discovers any crate with a `[[bin]]` automatically (library crates like `common` are skipped).

`install.sh` auto-discovers every `[[bin]]` crate under `tools/crates/` and, on Apple Silicon macOS with `gh` available, downloads each tool's tarball from the latest release into `tools/bin/` via `gh release download`. The repo is **private**, so the download is authenticated through `gh` — if `gh` is missing or not logged in (e.g. a fresh machine before `gh auth login`), it falls back to building everything from source via `make -C tools build`.

Caveats:
- Releases are built only for `aarch64-apple-darwin`; other platforms always build locally.
- Because the repo is private, asset downloads need `gh` authenticated for an account with access (`gh` is in the Brewfile).
- Pushing a commit/tag that adds or changes `.github/workflows/*` requires a GitHub token (or SSH key) with the `workflow` scope.
