//! The git side: turn an issue into a branch name and create it. Ported from
//! the Go `vcs.go`, deliberately tracker-agnostic.

use std::collections::HashSet;
use std::process::Command;

use anyhow::{bail, Context, Result};

/// Lowercase `summary` and collapse any run of non-alphanumeric characters into
/// a single hyphen, trimming leading/trailing hyphens.
pub fn slug(summary: &str) -> String {
    let mut out = String::new();
    let mut pending_dash = false;
    for c in summary.to_lowercase().chars() {
        if c.is_ascii_lowercase() || c.is_ascii_digit() {
            out.push(c);
            pending_dash = false;
        } else if !pending_dash {
            out.push('-');
            pending_dash = true;
        }
    }
    out.trim_matches('-').to_string()
}

/// Build `KEY-slug(summary)` (just `KEY` when the slug is empty).
pub fn branch(key: &str, summary: &str) -> String {
    let slug = slug(summary);
    if slug.is_empty() {
        key.to_string()
    } else {
        format!("{key}-{slug}")
    }
}

/// Switch to `branch`, creating it if it doesn't exist. Re-running `feature` on
/// the same issue just checks out the branch you already made, rather than
/// failing the way a bare `git checkout -b` would. Inherits the terminal so
/// git's output shows.
pub fn checkout(branch: &str) -> Result<()> {
    let args: Vec<&str> = if branch_exists(branch)? {
        vec!["checkout", branch]
    } else {
        vec!["checkout", "-b", branch]
    };
    let status = Command::new("git")
        .args(&args)
        .status()
        .context("running git checkout")?;
    if !status.success() {
        bail!("git checkout {branch} failed");
    }
    Ok(())
}

/// The set of local branch names, for marking issues that already have a branch.
/// Returns an empty set outside a git repo (callers treat that as "none").
pub fn local_branches() -> Result<HashSet<String>> {
    let out = Command::new("git")
        .args(["for-each-ref", "--format=%(refname:short)", "refs/heads"])
        .output()
        .context("listing git branches")?;
    if !out.status.success() {
        bail!("git for-each-ref failed");
    }
    Ok(String::from_utf8_lossy(&out.stdout)
        .lines()
        .map(|l| l.trim().to_string())
        .filter(|l| !l.is_empty())
        .collect())
}

/// Whether a local branch named `branch` already exists.
fn branch_exists(branch: &str) -> Result<bool> {
    let status = Command::new("git")
        .args([
            "show-ref",
            "--verify",
            "--quiet",
            &format!("refs/heads/{branch}"),
        ])
        .status()
        .context("checking for an existing branch")?;
    Ok(status.success())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn slug_cases() {
        assert_eq!(slug("Fix login redirect"), "fix-login-redirect");
        assert_eq!(
            slug("Postgresql-HA wastes 32GB of RAM"),
            "postgresql-ha-wastes-32gb-of-ram"
        );
        assert_eq!(slug("  Trim & punctuation!! "), "trim-punctuation");
        assert_eq!(slug("Enable CD via Jenkins"), "enable-cd-via-jenkins");
        assert_eq!(slug("DIJO2 — Add AZ to VPC"), "dijo2-add-az-to-vpc");
        assert_eq!(slug(""), "");
        assert_eq!(slug("!!!"), "");
    }

    #[test]
    fn branch_cases() {
        assert_eq!(branch("DRM-1", "Fix login"), "DRM-1-fix-login");
        assert_eq!(branch("DRM-2", "!!!"), "DRM-2");
    }
}
