# feature — pick an open Jira issue (or create one) and branch from it.
#
# This is a thin wrapper around the Go program in tools/feature. It auto-builds
# the binary on first use and whenever the sources change, so there is no
# separate install step and no committed binary. Implementation, the Jira
# specifics, and all UI live in tools/feature (see its README/source).
function feature() {
  # %x is the file this function was defined in; tools/feature sits beside the
  # functions/ dir that holds this file.
  local root="${${(%):-%x}:A:h}/../tools/feature"
  root="${root:A}"
  local bin="$root/bin/feature"

  # Rebuild if the binary is missing or any source is newer. (.om) sorts a glob
  # by modification time, newest first, so [1] is the most recently touched.
  local -a go_src
  go_src=( $root/**/*.go(N.om) )
  local need_build=0
  if [[ ! -x "$bin" ]]; then
    need_build=1
  elif [[ -n "${go_src[1]}" && "${go_src[1]}" -nt "$bin" ]]; then
    need_build=1
  elif [[ "$root/go.mod" -nt "$bin" ]] || [[ -f "$root/go.sum" && "$root/go.sum" -nt "$bin" ]]; then
    need_build=1
  fi

  if (( need_build )); then
    print -u2 "feature: building…"
    ( cd "$root" && go build -o bin/feature . ) || {
      print -u2 "feature: build failed"
      return 1
    }
  fi

  # The binary runs the picker, then runs `git checkout -b` itself; that touches
  # .git on disk, so the current shell ends up on the new branch.
  "$bin" "$@"
}
