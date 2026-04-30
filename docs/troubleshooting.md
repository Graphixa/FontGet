# Troubleshooting Guide

---

## Downloads fail with HTTP 202 (blocked by upstream challenge)

**What this means:** The host did not return the font file. Often this is an automated-client challenge from the CDN/WAF (you may see headers like `x-amzn-waf-action: challenge`). FontGet treats that as **not** a successful download.

**What you can try:**

- Run with **`--verbose`** or **`--debug`** to see whether the standard downloader or an external tool ran, and to see the full HTTP status message that was returned.
- In **`~/.fontget/config.yaml`**, adjust **`Network.DownloadUserAgent`** if a host is picky about User-Agents (defaults match `internal/config/default_config.yaml`; override as needed).
- FontGet’s standard downloader follows redirects and stores cookies for the request (closer to how browsers behave on some CDNs).
- FontGet **serializes downloads per host** by default to avoid looking like parallel “bot” traffic.
- Keep **`Network.EnableExternalDownloadFallback: true`** if you want FontGet to try **`curl` / `wget` / PowerShell`** when the plain HTTP path fails. FontGet only runs what is installed.
- **Retry later** or try another network — blocking rules can change by IP and time.
- See if the font is available from another source: `fontget search <font-name>`. 
    - Alternatively host the font files yourself (if licensing allows this) and setup your own custom hosted source, see [Custom sources](custom-sources.md).

**Note:** FontGet validates downloads so HTML or junk contents should not get installed as fonts.

---

## Downloads fail with HTTP 404 / “not found”

**What this means:** The URL may be wrong, the upstream file may have moved/been removed, or your local source catalog may be stale.

**What you can try:**

1. `fontget sources update`
2. Retry the install.
3. If it still fails: `fontget add <font-id> --debug` and note the exact URL and HTTP status.

---

## Downloads fail with HTTP 403 / “forbidden”

**What this means:** The host refused the request (rate limits, geo rules, permission rules, etc.).

**What you can try:**

- The same practical knobs as HTTP 202 (`--verbose` / `--debug`, `DownloadUserAgent`, external fallbacks), plus retry later.
- **Note:** FontGet cannot bypass a host’s policy.

---

## Font installs to the wrong place / wrong “scope”

**What this means:** FontGet installs to **user** or **machine** scope depending on flags and permissions.

**What you can try:**

- Use **`--scope user`** or **`--scope machine`** on **`fontget add`**, **`fontget browse`**, **`fontget import`**, etc.
- **Machine scope** usually needs elevation (Administrator / `sudo`).
- Confirm where fonts landed: **`fontget list --scope all`** (or filter by scope).

---

## Config reset doesn’t match docs

**What this means:** The FontGet binary you are running is older than the docs you’re reading. `fontget config reset` regenerates **`~/.fontget/config.yaml`** from the defaults embedded in from the installed version.

**What you can try:**

1. Update FontGet (see [Installation](installation.md) if `fontget update` isn’t available).
2. `fontget config reset`
3. `fontget config validate`

---

## `--debug` looks broken inside the TUI

**What this means:** Bubble Tea uses stdout for rendering; mixing unrelated stdout lines into the UI looks broken.

**What you can try:** Nothing — this is expected. Debug output goes to **stderr** so it does not corrupt fullscreen rendering.

---

## “command not found” / PATH

**What this means:** Your shell cannot find the `fontget` executable on `PATH` (or it points at an unexpected install location).

**What you can try:** See **Troubleshooting** in [Installation](installation.md).

---

## Terminal rendering / colors / completions

**What this means:** Your terminal, shell, or theme may affect colors, fonts, and completion loading.

**What you can try:** See [Terminal setup](terminal-setup.md).
