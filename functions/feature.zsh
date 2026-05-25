# Pick a Jira issue from my open work and create a branch named KEY-lowercased-summary.
# Press ctrl-n inside the picker to create a new Task with the typed query as the
# summary, transition it to In Progress, and use it for the branch.
function feature() {
  local fzf_out
  fzf_out=$(jira issue list \
      -s~Done -s~Archived -s~"To Do" -s~"In Review" \
      -t~Epic \
      -a itayka@dreamgroup.com \
      --columns TYPE,KEY,SUMMARY,STATUS --csv \
    | column -s, -t \
    | fzf \
        --print-query \
        --expect=ctrl-n \
        --header 'ctrl-n: create new task with the typed summary' \
        --header-lines=1 \
        --preview 'key=$(echo {} | awk "{print \$2}"); printf "\033[2;37m  loading %s...\033[0m\n" "$key"; out=$(jira issue view "$key" --raw 2>/dev/null); clear; echo "$out" | jq -r "
          \"\(.fields.issuetype.name)  •  \(.fields.status.name)  •  \(.fields.assignee.displayName // \"Unassigned\")\n\",
          \"# \(.fields.summary)\n\",
          (.fields.description // \"_No description_\")
        "' \
        --preview-window='right:35%:wrap,border-rounded' \
        --preview-label=' issue ')
  # Note: do not bail on non-zero exit — fzf returns 1 when ctrl-n is pressed with no
  # match, which is exactly the case we want to handle. Decide on parsed content below.

  local -a lines
  lines=("${(@f)fzf_out}")
  local query=${lines[1]-}
  local pressed=${lines[2]-}
  local selection=${lines[3]-}

  local branch_name
  if [[ "$pressed" == "ctrl-n" ]]; then
    local summary
    summary=$(print -r -- "$query" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
    [[ -z "$summary" ]] && { print -u2 "feature: empty summary, aborting"; return 1 }

    print -u2 "feature: creating Jira task: $summary"
    local create_out key
    create_out=$(jira issue create --no-input -t Task -s "$summary" -a itayka@dreamgroup.com --custom squad=Detection 2>&1) || {
      print -u2 "$create_out"
      return 1
    }

    key=$(print -r -- "$create_out" | grep -oE '[A-Z][A-Z0-9_]+-[0-9]+' | head -1)
    [[ -z "$key" ]] && { print -u2 "feature: could not parse issue key from:\n$create_out"; return 1 }

    print -u2 "feature: created $key, moving to In Progress"
    jira issue move "$key" "In Progress" >/dev/null 2>&1 \
      || print -u2 "feature: created $key but could not move to In Progress (continuing)"

    local slug
    slug=$(print -r -- "$summary" \
      | tr -c 'a-zA-Z0-9 -' ' ' \
      | tr -s ' ' \
      | sed 's/^ *//; s/ *$//' \
      | tr ' ' '-' \
      | tr '[:upper:]' '[:lower:]')
    branch_name="${key}-${slug}"
  elif [[ -n "$selection" ]]; then
    branch_name=$(print -r -- "$selection" | awk '{
        key=$2
        summary=""
        for(i=3;i<=NF-3;i++) summary = summary " " $i
        gsub(/[^a-zA-Z0-9 -]/, "", summary)
        gsub(/  +/, " ", summary)
        sub(/^ +/, "", summary); sub(/ +$/, "", summary)
        gsub(/ /, "-", summary)
        print toupper(key) "-" tolower(summary)
      }')
  else
    return 1
  fi

  [[ -z "$branch_name" ]] && return 1
  git checkout -b "$branch_name"
}
