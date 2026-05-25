# Pick a Jira issue from my open work and create a branch named KEY-lowercased-summary.
function feature() {
  local branch_name
  branch_name=$(jira issue list \
      -s~Done -s~Archived -s~"To Do" -s~"In Review" \
      -t~Epic \
      -a itayka@dreamgroup.com \
      --columns TYPE,KEY,SUMMARY,STATUS --csv \
    | column -s, -t \
    | fzf \
        --header-lines=1 \
        --preview 'key=$(echo {} | awk "{print \$2}"); printf "\033[2;37m  loading %s...\033[0m\n" "$key"; out=$(jira issue view "$key" --raw 2>/dev/null); clear; echo "$out" | jq -r "
          \"\(.fields.issuetype.name)  •  \(.fields.status.name)  •  \(.fields.assignee.displayName // \"Unassigned\")\n\",
          \"# \(.fields.summary)\n\",
          (.fields.description // \"_No description_\")
        "' \
        --preview-window='right:35%:wrap,border-rounded' \
        --preview-label=' issue ' \
    | awk '{
        key=$2
        summary=""
        for(i=3;i<=NF-3;i++) summary = summary " " $i
        gsub(/[^a-zA-Z0-9 -]/, "", summary)
        gsub(/  +/, " ", summary)
        sub(/^ +/, "", summary); sub(/ +$/, "", summary)
        gsub(/ /, "-", summary)
        print toupper(key) "-" tolower(summary)
      }')

  [[ -z "$branch_name" ]] && return 1
  git checkout -b "$branch_name"
}
