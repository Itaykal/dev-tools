# kalfon-dotfiles — agent guide

Personal zsh dotfiles repo. `CLAUDE.md` holds the full repo conventions (structure,
shell layout rules, p10k/k9s/macos notes) — read it first. This file documents the
release/versioning flow for the Rust tools.

## Releasing the Rust tools (`tools/`)

The Rust CLIs under `tools/` (`feature`, `aws-switch`, …) are released **independently —
one release per tool**, each with its own version.

To cut a release, push a tag of the form `<tool>-v<version>`:

```sh
git tag feature-v0.2.0
git push origin feature-v0.2.0
```

- `<tool>` must match a binary crate dir under `tools/crates/` (package/bin/dir names are identical).
- `<version>` is semver; the `-v` separator supplies the `v`.
- Each tool versions on its own timeline; bumping `feature` doesn't touch `aws-switch`.

What happens: `.github/workflows/release.yml` triggers on `*-v*` tags, derives the tool
name from the part before `-v`, builds **only that crate**
(`cargo build --release -p <tool>` on a `macos-14`/`aarch64-apple-darwin` runner),
packages the single binary as `<tool>-aarch64-apple-darwin.tar.gz`, and publishes a
GitHub Release for that tag.

**New tools need no workflow edits** — add the binary crate under `tools/crates/<tool>/`,
then tag `<tool>-v<version>`. The workflow validates the tag maps to a `[[bin]]` crate
and fails the run otherwise.

`install.sh` auto-discovers every `[[bin]]` crate under `tools/crates/` and, on Apple
Silicon macOS, downloads each tool's **latest** `<tool>-v*` release into `tools/bin/`.
If any tool has no release yet (or a download fails), it falls back to building
everything from source via `make -C tools build`.

Caveats:
- Releases are built only for `aarch64-apple-darwin`; other platforms always build locally.
- Pushing a commit/tag that adds or changes `.github/workflows/*` requires a GitHub token
  (or SSH key) with the `workflow` scope.
