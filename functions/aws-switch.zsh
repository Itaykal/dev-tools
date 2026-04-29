# Re-auth via the existing "session" SSO session if needed, pick a real account
# from the portal with fzf, then write it into [profile default] and export env vars.
function aws-switch() {
  local sso_region sso_start_url
  sso_region=$(awk '/\[sso-session session\]/{f=1} f && /^sso_region/{print $3; exit}' ~/.aws/config)
  sso_region=${sso_region:-eu-west-1}
  sso_start_url=$(awk '/\[sso-session session\]/{f=1} f && /^sso_start_url/{print $3; exit}' ~/.aws/config)

  # Pick the token cached for this sso-session's start_url (not just the latest file).
  _aws_switch_token() {
    jq -rs --arg url "$sso_start_url" '
      [.[] | select(.accessToken != null and (.startUrl == $url or $url == ""))]
      | sort_by(.expiresAt) | last | .accessToken // empty
    ' ~/.aws/sso/cache/*.json 2>/dev/null
  }

  local token accounts
  token=$(_aws_switch_token)
  if [[ -n "$token" ]]; then
    accounts=$(aws sso list-accounts --access-token "$token" --region "$sso_region" --output json 2>/dev/null)
  fi

  # If no token, or list-accounts failed/empty, the SSO token is expired — log in and retry.
  if [[ -z "$accounts" ]] || ! jq -e '.accountList | length > 0' <<<"$accounts" >/dev/null 2>&1; then
    echo "SSO session expired — logging in..."
    aws sso login --sso-session session || return 1
    token=$(_aws_switch_token)
    [[ -z "$token" ]] && { echo "Could not read SSO token after login" >&2; return 1; }
    accounts=$(aws sso list-accounts --access-token "$token" --region "$sso_region" --output json) || return 1
  fi

  # Aligned two-column list: account name (fixed 35 chars) + account ID
  local selection
  selection=$(jq -r '.accountList[] | [.accountName, .accountId] | @tsv' <<<"$accounts" \
    | sort \
    | awk 'BEGIN{FS="\t"} {printf "%-35s %s\n", $1, $2}' \
    | fzf --prompt="AWS account: " --height=40% --reverse) || return 1
  [[ -z "$selection" ]] && return 1

  local acct_name acct_id
  acct_id=$(awk '{print $NF}' <<< "$selection")
  acct_name=$(awk '{$NF=""; gsub(/ +$/, ""); print}' <<< "$selection")

  # Pick a role (auto-select if only one)
  local roles role_name
  roles=$(aws sso list-account-roles \
    --access-token "$token" \
    --region "$sso_region" \
    --account-id "$acct_id" \
    --query 'roleList[*].roleName' \
    --output text 2>/dev/null | tr '\t' '\n' | grep -v '^$')
  if [[ $(echo "$roles" | wc -l) -gt 1 ]]; then
    role_name=$(echo "$roles" | fzf --prompt="Role: " --height=20% --reverse) || return 1
  else
    role_name="$roles"
  fi

  # Overwrite the fixed 'default' profile.
  # sso_account_name is non-standard but AWS CLI ignores unknown keys; aws-sync-prompt reads it
  # so other terminals can show the account name without depending on this shell's env.
  aws configure set sso_session     session       --profile default
  aws configure set sso_account_id  "$acct_id"    --profile default
  aws configure set sso_account_name "$acct_name" --profile default
  aws configure set sso_role_name   "$role_name"  --profile default
  aws configure set region          "$sso_region" --profile default

  export AWS_PROFILE=default
  export AWS_VAULT="$acct_name"       # p10k reads AWS_VAULT before AWS_PROFILE for display & class matching
  export AWS_ACCOUNT_ID="$acct_id"
  export AWS_ROLE_NAME="$role_name"
  export AWS_DEFAULT_REGION="$sso_region"

  echo "${acct_name}  ${acct_id}  ${role_name}  ${AWS_DEFAULT_REGION}"
}
