# kalfon-dotfiles

Personal zsh dotfiles for macOS — AWS SSO account switching, Powerlevel10k overrides, and shell aliases.

## Structure

| File | Purpose |
|------|---------|
| `.entry` | Entrypoint — sources all other files in order. Add `source ~/kalfon-dotfiles/.entry` to `~/.zshrc`. |
| `.vars` | Environment variables (e.g. `LC_TIME`). |
| `.alias` | Shell aliases: k8s shortcuts (`kk`, `kx`, `kn`), IaC shortcuts (`tf`, `tg`), AWS switcher (`awsw`). |
| `.functions` | Shell functions — currently `aws-switch`. |
| `.p10k` | Powerlevel10k overrides. Source this after `~/.p10k.zsh`. |

## Installation

```zsh
git clone https://github.com/Itaykal/kalfon-dotfiles.git ~/kalfon-dotfiles
```

Add to `~/.zshrc` (after oh-my-zsh and p10k are loaded):

```zsh
source ~/kalfon-dotfiles/.entry
```

### Dependencies

- [oh-my-zsh](https://ohmyz.sh/)
- [Powerlevel10k](https://github.com/romkatv/powerlevel10k)
- [fzf](https://github.com/junegunn/fzf) — `brew install fzf`
- [jq](https://jqlang.github.io/jq/) — `brew install jq`
- [k9s](https://k9scli.io/), [kubectx/kubens](https://github.com/ahmetb/kubectx) — for k8s aliases
- AWS CLI v2 configured with an SSO session named `session`

## AWS SSO Switcher (`awsw`)

Run `awsw` to switch AWS accounts:

1. Checks if the SSO session is valid; re-authenticates via `aws sso login --sso-session session` if expired.
2. Lists all accounts available to you in the SSO portal (real account names, not profile keys).
3. If the selected account has multiple roles, prompts to pick one.
4. Writes the selection into a single `[profile default]` in `~/.aws/config` — no profile sprawl.
5. Exports `AWS_PROFILE`, `AWS_ACCOUNT_ID`, `AWS_ACCOUNT_NAME`, `AWS_ROLE_NAME`, and `AWS_DEFAULT_REGION`.

The Powerlevel10k right prompt always shows the active account: `account-name  account-id  role  region`.

## Powerlevel10k overrides (`.p10k`)

- **kubecontext** — own line at the top-left, above `dir/vcs`. Visible only when typing k8s commands.
- **aws** — right prompt, immediately after the command timer. Always visible.

These overrides survive `p10k configure` regenerating `~/.p10k.zsh` because they live in a separate file.
