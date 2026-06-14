//! Everything that touches AWS SSO, the token cache, and `~/.aws/config`.
//!
//! Account/role listing hits the SSO portal REST API directly over HTTPS using
//! the cached bearer token — the same endpoints `aws sso list-accounts` calls,
//! but without paying the ~400ms `aws` CLI (Python) startup on each one. Only
//! `aws sso login` still shells out, since it drives the browser/device flow.
//! `~/.aws/config` is written in the AWS CLI's exact `key = value` / `[default]`
//! format so both the real `aws` CLI and the pure-zsh `aws-sync-prompt` parser
//! keep working.

use std::path::{Path, PathBuf};
use std::process::Command;

use anyhow::{bail, Context, Result};
use serde::{Deserialize, Serialize};

/// An expired/invalid SSO token (the portal answered 401/403). The caller
/// detects this with `downcast_ref` and responds by logging in and retrying.
#[derive(Debug)]
pub struct Unauthorized;

impl std::fmt::Display for Unauthorized {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "SSO token is unauthorized (expired)")
    }
}

impl std::error::Error for Unauthorized {}

/// An AWS account from the SSO portal `list-accounts` API. `Serialize` so the
/// list can be cached.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Account {
    #[serde(rename = "accountId")]
    pub account_id: String,
    #[serde(rename = "accountName")]
    pub account_name: String,
}

#[derive(Deserialize)]
struct AccountsResponse {
    #[serde(rename = "accountList", default)]
    account_list: Vec<Account>,
    #[serde(rename = "nextToken", default)]
    next_token: Option<String>,
}

#[derive(Deserialize)]
struct RolesResponse {
    #[serde(rename = "roleList", default)]
    role_list: Vec<Role>,
    #[serde(rename = "nextToken", default)]
    next_token: Option<String>,
}

#[derive(Deserialize)]
struct Role {
    #[serde(rename = "roleName")]
    role_name: String,
}

fn home() -> Result<PathBuf> {
    dirs::home_dir().context("could not determine home directory")
}

/// Read `sso_region` and `sso_start_url` from the `[sso-session <session>]`
/// block of `~/.aws/config`. Either may be absent.
pub fn read_sso_session(session: &str) -> Result<(Option<String>, Option<String>)> {
    let path = home()?.join(".aws/config");
    let text = match std::fs::read_to_string(&path) {
        Ok(t) => t,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok((None, None)),
        Err(e) => return Err(e).with_context(|| format!("reading {}", path.display())),
    };

    let header = format!("[sso-session {session}]");
    let (mut region, mut start_url) = (None, None);
    let mut in_section = false;
    for line in text.lines() {
        let line = line.trim();
        if line.starts_with('[') {
            in_section = line == header;
            continue;
        }
        if !in_section {
            continue;
        }
        if let Some((k, v)) = line.split_once('=') {
            match k.trim() {
                "sso_region" => region = Some(v.trim().to_string()),
                "sso_start_url" => start_url = Some(v.trim().to_string()),
                _ => {}
            }
        }
    }
    Ok((region, start_url))
}

/// Return the cached SSO access token for `start_url` (or any session if
/// `start_url` is `None`), choosing the one with the latest `expiresAt`.
pub fn read_token(start_url: Option<&str>) -> Result<Option<String>> {
    let dir = home()?.join(".aws/sso/cache");
    let entries = match std::fs::read_dir(&dir) {
        Ok(e) => e,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(e) => return Err(e).with_context(|| format!("reading {}", dir.display())),
    };

    let mut best: Option<(String, String)> = None; // (expiresAt, accessToken)
    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        let Ok(text) = std::fs::read_to_string(&path) else {
            continue;
        };
        let Ok(value) = serde_json::from_str::<serde_json::Value>(&text) else {
            continue;
        };
        let Some(token) = value.get("accessToken").and_then(|v| v.as_str()) else {
            continue;
        };
        if let Some(want) = start_url {
            let got = value.get("startUrl").and_then(|v| v.as_str());
            if got != Some(want) {
                continue;
            }
        }
        // expiresAt is RFC3339 UTC, so lexical comparison orders it correctly.
        let expires = value
            .get("expiresAt")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();
        if best.as_ref().is_none_or(|(e, _)| expires > *e) {
            best = Some((expires, token.to_string()));
        }
    }
    Ok(best.map(|(_, token)| token))
}

/// `aws sso login --sso-session <session>`, inheriting the terminal so the
/// device-code prompt is visible.
pub fn login(session: &str) -> Result<()> {
    let status = Command::new("aws")
        .args(["sso", "login", "--sso-session", session])
        .status()
        .context("running `aws sso login`")?;
    if !status.success() {
        bail!("`aws sso login` failed");
    }
    Ok(())
}

/// The SSO portal endpoint for a region (the host the AWS CLI talks to under
/// the hood for `sso list-accounts` / `list-account-roles`).
fn portal(region: &str) -> String {
    format!("https://portal.sso.{region}.amazonaws.com")
}

const BEARER_HEADER: &str = "x-amz-sso_bearer_token";
const PAGE_SIZE: &str = "100";

/// Send an SSO portal request, turning a 401/403 into [`Unauthorized`] (so the
/// caller can log in and retry) and any other failure into a contextual error.
fn send(req: ureq::Request, what: &str) -> Result<ureq::Response> {
    match req.call() {
        Ok(resp) => Ok(resp),
        Err(ureq::Error::Status(401 | 403, _)) => Err(anyhow::Error::new(Unauthorized)),
        Err(e) => Err(anyhow::Error::new(e)).with_context(|| format!("{what} request")),
    }
}

/// List accounts the token can access, following pagination. A non-2xx
/// response (e.g. an expired token → 401) surfaces as an `Err`, which the
/// caller treats as "log in and retry".
pub fn list_accounts(token: &str, region: &str) -> Result<Vec<Account>> {
    let url = format!("{}/assignment/accounts", portal(region));
    let mut accounts = Vec::new();
    let mut next: Option<String> = None;
    loop {
        let mut req = ureq::get(&url)
            .set(BEARER_HEADER, token)
            .query("max_result", PAGE_SIZE);
        if let Some(n) = &next {
            req = req.query("next_token", n);
        }
        let page: AccountsResponse = send(req, "SSO list-accounts")?
            .into_json()
            .context("parsing list-accounts response")?;
        accounts.extend(page.account_list);
        match page.next_token {
            Some(n) if !n.is_empty() => next = Some(n),
            _ => break,
        }
    }
    Ok(accounts)
}

/// List the role names available for an account, following pagination.
pub fn list_roles(token: &str, region: &str, account_id: &str) -> Result<Vec<String>> {
    let url = format!("{}/assignment/roles", portal(region));
    let mut roles = Vec::new();
    let mut next: Option<String> = None;
    loop {
        let mut req = ureq::get(&url)
            .set(BEARER_HEADER, token)
            .query("account_id", account_id)
            .query("max_result", PAGE_SIZE);
        if let Some(n) = &next {
            req = req.query("next_token", n);
        }
        let page: RolesResponse = send(req, "SSO list-account-roles")?
            .into_json()
            .context("parsing list-account-roles response")?;
        roles.extend(page.role_list.into_iter().map(|r| r.role_name));
        match page.next_token {
            Some(n) if !n.is_empty() => next = Some(n),
            _ => break,
        }
    }
    Ok(roles)
}

/// Write the given key/values into `profile`'s section of `~/.aws/config`,
/// preserving every other line.
///
/// This replaces five sequential `aws configure set` calls — each of which pays
/// the full `aws` CLI (Python) startup cost (~1s total). We update the file
/// directly instead, matching the exact `key = value` / `[default]` format the
/// AWS CLI writes, so both the real `aws` CLI and the pure-zsh `aws-sync-prompt`
/// parser keep working. Existing keys are updated in place; missing ones are
/// appended to the section; unrelated profiles are untouched.
///
/// The new contents are written atomically (temp file + rename), so a crash or
/// full disk mid-write can never truncate the user's `~/.aws/config` — they
/// keep the previous file intact.
pub fn write_profile(profile: &str, entries: &[(&str, &str)]) -> Result<()> {
    let path = home()?.join(".aws/config");
    let original = match std::fs::read_to_string(&path) {
        Ok(t) => t,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => String::new(),
        Err(e) => return Err(e).with_context(|| format!("reading {}", path.display())),
    };

    // The default profile is `[default]`; others are `[profile NAME]`.
    let header = if profile == "default" {
        "[default]".to_string()
    } else {
        format!("[profile {profile}]")
    };

    let out = update_ini(&original, &header, entries);
    write_atomic(&path, &out)
}

/// Replace `path`'s contents with `contents` atomically: write a sibling temp
/// file, then `rename` it over the target (an atomic swap on the same
/// filesystem). The temp file is removed if the rename fails. The temp lives in
/// the target's own directory so the rename never crosses a filesystem boundary
/// (a cross-device rename isn't atomic and would fail).
fn write_atomic(path: &Path, contents: &str) -> Result<()> {
    let dir = path
        .parent()
        .with_context(|| format!("{} has no parent directory", path.display()))?;
    let name = path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("config");
    // Per-pid temp name so concurrent invocations don't clobber each other.
    let tmp = dir.join(format!(".{name}.tmp.{}", std::process::id()));

    std::fs::write(&tmp, contents).with_context(|| format!("writing {}", tmp.display()))?;
    std::fs::rename(&tmp, path)
        .with_context(|| format!("replacing {}", path.display()))
        .inspect_err(|_| {
            let _ = std::fs::remove_file(&tmp);
        })
}

/// Pure core of [`write_profile`]: return `original` with `entries` set under
/// `header`, preserving all other lines. Existing keys are updated in place,
/// missing keys are appended to the section, and the section is created at EOF
/// if absent. Output always ends with a trailing newline.
fn update_ini(original: &str, header: &str, entries: &[(&str, &str)]) -> String {
    let mut lines: Vec<String> = original.lines().map(str::to_string).collect();

    match lines.iter().position(|l| l.trim() == header) {
        Some(start) => {
            // Section runs until the next section header (or EOF).
            let end = lines[start + 1..]
                .iter()
                .position(|l| l.trim_start().starts_with('['))
                .map(|i| start + 1 + i)
                .unwrap_or(lines.len());

            let mut append = Vec::new();
            for &(k, v) in entries {
                match (start + 1..end).find(|&i| line_key(&lines[i]) == Some(k)) {
                    Some(i) => lines[i] = format!("{k} = {v}"),
                    None => append.push(format!("{k} = {v}")),
                }
            }
            for (offset, line) in append.into_iter().enumerate() {
                lines.insert(end + offset, line);
            }
        }
        None => {
            lines.push(header.to_string());
            for &(k, v) in entries {
                lines.push(format!("{k} = {v}"));
            }
        }
    }

    let mut out = lines.join("\n");
    out.push('\n');
    out
}

/// The key of an `ini` `key = value` line, or `None` for headers/blanks.
fn line_key(line: &str) -> Option<&str> {
    let line = line.trim();
    if line.starts_with('[') {
        return None;
    }
    line.split_once('=').map(|(k, _)| k.trim())
}

#[cfg(test)]
mod tests {
    use super::{update_ini, write_atomic};

    #[test]
    fn write_atomic_creates_overwrites_and_leaves_no_temp() {
        let dir = std::env::temp_dir().join(format!("aws-switch-test-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let path = dir.join("config");

        write_atomic(&path, "first\n").unwrap();
        assert_eq!(std::fs::read_to_string(&path).unwrap(), "first\n");

        write_atomic(&path, "second\n").unwrap();
        assert_eq!(std::fs::read_to_string(&path).unwrap(), "second\n");

        // The temp file must not linger after a successful rename.
        let leftovers: Vec<_> = std::fs::read_dir(&dir)
            .unwrap()
            .filter_map(|e| e.ok())
            .filter(|e| e.file_name().to_string_lossy().contains(".tmp."))
            .collect();
        assert!(leftovers.is_empty(), "temp file left behind: {leftovers:?}");

        std::fs::remove_dir_all(&dir).unwrap();
    }

    #[test]
    fn updates_existing_keys_in_place_and_preserves_others() {
        let original = "\
[profile dev]
sso_account_id = 111
[default]
sso_account_id = 111
region = eu-west-1
[sso-session session]
sso_region = eu-west-1
";
        let got = update_ini(
            original,
            "[default]",
            &[("sso_account_id", "222"), ("region", "us-east-1")],
        );
        assert_eq!(
            got,
            "\
[profile dev]
sso_account_id = 111
[default]
sso_account_id = 222
region = us-east-1
[sso-session session]
sso_region = eu-west-1
"
        );
    }

    #[test]
    fn appends_missing_keys_to_the_section() {
        let original = "[default]\nregion = eu-west-1\n[other]\nx = 1\n";
        let got = update_ini(original, "[default]", &[("sso_role_name", "DevOps")]);
        assert_eq!(
            got,
            "[default]\nregion = eu-west-1\nsso_role_name = DevOps\n[other]\nx = 1\n"
        );
    }

    #[test]
    fn creates_section_when_absent() {
        let got = update_ini("[other]\nx = 1\n", "[default]", &[("region", "eu-west-1")]);
        assert_eq!(got, "[other]\nx = 1\n[default]\nregion = eu-west-1\n");
    }

    #[test]
    fn non_default_profile_uses_profile_prefix_section() {
        // write_profile picks the header; here we just confirm update_ini honors it.
        let got = update_ini("", "[profile work]", &[("region", "eu-west-1")]);
        assert_eq!(got, "[profile work]\nregion = eu-west-1\n");
    }
}
