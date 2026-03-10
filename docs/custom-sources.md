# Custom Sources Guide

This guide shows how to create a custom font source repository and add it to FontGet.

## Overview

A custom source is a hosted `fonts.json` file that FontGet can read.  
That JSON has two main parts:

- `source_info` (metadata about your source)
- `fonts` (font IDs, variants, and download URLs)

Once your file is publicly accessible, add it with `fontget sources add`.

This example uses a custom GitHub repo, but you can host your source file and fonts anywhere public on the internet.

---

## Step 1: Create a new GitHub repository

Create a **public** repository on your GitHub account (example name: `fontget-source`).

Suggested structure:

```text
fontget-source/
  fonts.json
  fonts/
    Iosevka-Regular.ttf
```

The `fonts/` folder is optional. You can also point to font files hosted elsewhere.

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
            "ttf": "REPLACE_WITH_FONT_URL"
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

## Step 4: Commit and publish

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

- Source URL is private or requires login.
- `files` uses `zip` key instead of `ttf`/`otf`.
- Using `.tar.gz` archive URLs.
- `source_info.total_fonts` does not match entries in `fonts`.

## Troubleshooting checklist

- Source URL is valid JSON and publicly reachable.
- `files` URLs are publicly reachable.
- `files` keys are `ttf` or `otf`.
- Archive URLs use supported formats (`.zip` or `.tar.xz`).
- `source_info.total_fonts` matches the number of font entries.
- Font IDs are unique and stable.


## Notes

- Use trusted URLs only.
- Ensure font licensing allows redistribution and installation.
- Keep `source_info.total_fonts` in sync with your `fonts` entries.
- Use stable URLs for font files to avoid broken installs.
- Confirm links are publicly accessible (no auth required), otherwise installs will fail.
