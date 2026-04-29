# Custom Sources Guide

This guide shows how to create a custom font source repository and add it to FontGet.

## Overview

A custom source is a hosted `fonts.json` file that FontGet can read.  
That JSON has two main parts:

- `source_info` (metadata about your source)
- `fonts` (font IDs, variants, and download URLs)

Once your file is publicly accessible, add it with `fontget sources manage`.

This example uses a custom GitHub repo, but you can host your source file and fonts anywhere public on the internet.

---

## Step 1: Create a new GitHub repository

1. Create a **public** repository (empty is fine).
2. Add a manifest file — call it `fonts.json` or anything you like; the name only matters for your own sanity.
3. Optionally add font binaries under something like `fonts/` in the same repo.
4. Push to your default branch (`main` or `master`).

**Manifest URL:** open the file on GitHub → **Raw** → copy the address. It looks like:

`https://raw.githubusercontent.com/<you>/<repo>/<branch>/fonts.json`

Paste that into a browser tab with **private/incognito** off any VPN: if you see raw JSON, FontGet can fetch it.

---

## Step 2: Create `fonts.json`

Create a file named `fonts.json` in the repo root and paste this template:

```json
{
  "source_info": {
    "name": "My GitHub Fonts",
    "description": "Custom FontGet source by @gh-username",
    "url": "https://github.com/<username>/<repo>",
    "version": "1.0.0",
    "last_updated": "2026-03-10T00:00:00Z",
    "total_fonts": 1
  },
  "fonts": {
    "iosevka": {
      "name": "Iosevka",
      "family": "Iosevka",
      "license": "OFL",
      "variants": [
        {
          "name": "Regular",
          "weight": 400,
          "style": "normal",
          "files": {
            "ttf": "https://github.com/<username>/<repo>/main/fonts/CoolFont-Regular.ttf"
          }
        }
      ]
    }
  }
}
```

Now replace:

- `<username>` with your GitHub username/org.
- `<repo>` with your repository name.
- `REPLACE_WITH_FONT_URL` with a real direct font URL or archive URL.

---

## Step 3: Choose where font files come from

You have two valid options.

### Option A: Host your custom font files in your GitHub repo

Put files in your repo (for example `fonts/Iosevka-Regular.ttf`) and use Raw URLs.

Example:

```json
{
  "files": {
    "ttf": "https://raw.githubusercontent.com/<username>/<repo>/main/fonts/Iosevka-Regular.ttf"
  }
}
```

### Option B: Link to externally hosted font URLs

If fonts are hosted elsewhere, point `files` to those URLs directly.

Example:

```json
{
  "files": {
    "ttf": "https://example.org/fonts/Iosevka-Regular.ttf"
  }
}
```

### Option C: Use archive links (ZIP / TAR.XZ)

You can also point to a release archive that contains font files.

Example:

```json
{
  "files": {
    "ttf": "https://github.com/<username>/<repo>/releases/download/v1.0.0/Iosevka.zip"
  }
}
```

> [!NOTE]
> - Keep `files` keys as `ttf` or `otf` - zips are still supported this way.
> - Supported archive URLs: `.zip`, `.tar.xz`.
> - `.tar.gz` is not currently supported.


---

## Register the source in FontGet

1. Commit `fonts.json` (and optional `fonts/` files).
2. Push to your default branch.
3. Copy your raw `fonts.json` URL:

```text
https://raw.githubusercontent.com/<username>/<repo>/main/fonts.json
```

4. Open that URL in your browser and confirm it loads JSON publicly.

---

## Step 5: Add the source to FontGet

Run:

```bash
fontget sources add --name "My GitHub Fonts" --url "https://raw.githubusercontent.com/<username>/<repo>/main/fonts.json" --prefix gh --priority 10
```

Then validate/update:

```bash
fontget sources info
fontget sources validate
fontget sources update
```

---

## Step 6: Install from your custom source

Use your prefix plus font ID:

```bash
fontget add "gh.iosevka"
```

---


## Common mistakes

- Using a URL that only works when you're logged in (private repo, cookies, “Sign in to view” pages).
- Copying the GitHub *file page* URL instead of the **Raw** URL.
- Pointing `files` at a webpage (HTML) instead of a direct font file or archive.
- `source_info.total_fonts` does not match entries in `fonts`.
- Changing font IDs over time (breaks scripts and saved installs).

## Troubleshooting checklist

- Open the manifest URL in a fresh browser session (no login, no VPN) and confirm it returns JSON.
- Run `fontget sources update` and `fontget sources validate`.
- If installs fail, run `fontget add <prefix>.<font-id> --debug` and look for the exact URL it tried.
- Open that URL in a browser and confirm it downloads a real file (not HTML, not a 302-to-login, not a blocked 202/403/404).
- Keep `files` keys as `ttf` / `otf`.
- Archive URLs can be `.zip`, `.tar.xz`, `.tar.gz`/`.tgz`, or `.7z` (7z extraction needs `7zz`/`7z` available).


## Notes

- Your manifest must be public. FontGet can't log in to fetch it.
- Use stable URLs. If you publish “latest.zip” and later replace it, old installs can break.
- Ensure font licensing allows redistribution and installation.
- Keep `source_info.total_fonts` in sync with your `fonts` entries.