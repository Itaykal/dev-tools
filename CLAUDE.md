# kalfon-dotfiles

Personal zsh dotfiles repo. All shell customisations live here — never edit `~/.p10k.zsh`, `~/.zshrc` aliases, or similar files directly.

## File map

- `.entry` — sources `.vars`, `.alias`, `.functions`, `.p10k` in order
- `.vars` — exported env vars
- `.alias` — aliases only (no functions)
- `.functions` — shell functions (currently `aws-switch`)
- `.p10k` — Powerlevel10k overrides only; do not touch `~/.p10k.zsh`

## Rules

- **All changes go in this repo.** Never patch `~/.p10k.zsh` or `~/.zshrc` directly.
- Aliases belong in `.alias`, functions in `.functions`. Don't mix them.
- `.p10k` is sourced after `~/.p10k.zsh`, so overrides here win. Use array manipulation to reorder prompt elements rather than redefining the full array.
- The AWS SSO session is always named `session`. The active profile is always `default`. Do not introduce new profile names.
- `AWS_ACCOUNT_NAME`, `AWS_ACCOUNT_ID`, `AWS_ROLE_NAME`, `AWS_DEFAULT_REGION` are exported by `aws-switch` and referenced in the p10k content expansion — keep them in sync if either side changes.
- `jq` and `fzf` are assumed to be installed. Do not add fallbacks for them.
