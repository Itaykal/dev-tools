# Renders a Jira issue (--raw JSON) into the three-line preview used by feature().
# The description field is in Atlassian Document Format (ADF); walk the tree and
# emit a markdown-ish flattening, falling back to children for unknown node types.

def r:
  if type == "object" then
    if   .type == "text"        then (.text // "")
    elif .type == "hardBreak"   then "\n"
    elif .type == "paragraph"   then (([.content[]? | r] | join("")) + "\n\n")
    elif .type == "heading"     then ("\n" + ([.content[]? | r] | join("")) + "\n\n")
    elif .type == "bulletList"  then ([.content[]? | r] | join(""))
    elif .type == "orderedList" then ([.content[]? | r] | join(""))
    elif .type == "listItem"    then ("• " + ([.content[]? | r] | join("")))
    elif .type == "codeBlock"   then ("\n" + ([.content[]? | r] | join("")) + "\n")
    elif .type == "inlineCard"  then (.attrs.url // "")
    elif .type == "mention"     then (.attrs.text // "@?")
    elif .type == "rule"        then "\n---\n"
    else ([.content[]? | r] | join(""))
    end
  else "" end;

"\(.fields.issuetype.name)  •  \(.fields.status.name)  •  \(.fields.assignee.displayName // "Unassigned")\n",
"# \(.fields.summary)\n",
(if .fields.description == null then "_No description_" else (.fields.description | r) end)
