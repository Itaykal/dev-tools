# Wrap the `feature` CLI so that when it spins an issue out into a git worktree
# (ctrl-enter, or the worktree button in the create modal) your shell drops
# straight into the new directory. The binary draws its TUI on stderr and prints
# the worktree path on stdout — and nothing on stdout for a plain branch or a
# cancel — so we capture stdout and cd into it only when it's a real directory.
function feature() {
  local out
  out="$(command feature "$@")" || return
  local dir=${out##*$'\n'}   # last line is the path; guards against stray stdout
  [[ -n $dir && -d $dir ]] && cd "$dir"
}
