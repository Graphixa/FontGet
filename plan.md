# fontget — Development Blueprint
*A tiny, cross‑platform font package manager written in Go*

---

## 1  Project Purpose

Deliver **fontget**, a single‑file CLI tool for Windows, macOS and Linux that lets users install fonts from the command line. Example:

```
fontget add "open‑sans"         # alias: install
fontget remove "open‑sans"      # alias: uninstall
fontget list                    # list installed fonts
fontget search "noto"           # search the index for fonts based on the user query.
fontget import                  # imports a .json file of fonts and adds all listed fonts
fontget export                  # exports all installed fonts on the system to a json file
  fontget export --google         # exports only fonts that are referenced in the google repo to .json file
fontget repo                    # lists the repositories of FontGet (Google by default)
  fontget repo --update           # refresh font index
  fontget repo --add              # append a new index URL to the repositories file
  fontget repo --remove           # remove an existing index URL the repositories file
fontget prune                   # clear unused downloads
```

`fontget` draws its catalogue from Google Fonts → OFL (plus any user repos), verifies
SHA‑256, copies fonts into the correct system or user directory, and keeps an on‑disk
cache for offline installs.

---

## 2  Technology Stack

| Layer            | Choice & Rationale                                                        |
|------------------|---------------------------------------------------------------------------|
| Language         | **Go ≥ 1.22** — fast compile, tiny statically‑linked binaries             |
| CLI Parsing      | **Cobra** — abundant examples, man/completion generators                  |
| Config / Env     | **Viper** — YAML/TOML/JSON plus env‑var override                          |
| Packaging        | **GoReleaser** — multi‑platform archives, .deb, .msi, Homebrew, winget    |
| CI / Hosting     | **GitHub Actions** — test, lint, cross‑build, release                     |
| Cache Directory  | `os.UserCacheDir()` → …/fontget/ per XDG / macOS / Windows                |

---

## 3  Repository Layout

```
fontget/
├── cmd/                 # Cobra commands
│   ├── root.go          # global flags, config bootstrap
│   ├── add.go           # install (alias: add)
│   ├── remove.go        # uninstall
│   ├── list.go
│   ├── repo.go          # repo update/add/remove
│   ├── import.go
│   ├── export.go
│   └── cache.go         # prune
├── internal/
│   ├── cache/           # index + font blob cache
│   ├── index/           # JSON schema + helpers
│   ├── platform/        # windows.go, linux.go, macos.go
│   └── repo/            # HTTP fetch, checksum, retries
├── indexer/             # stand‑alone generator (CI only)
│   └── main.go
├── .goreleaser.yml
├── .github/workflows/
│   ├── ci.yaml
│   └── release.yaml
├── go.mod
└── README.md
```

### Key internal packages

| Package            | Responsibility                                                                    |
|--------------------|-----------------------------------------------------------------------------------|
| `internal/cache`   | Locate cache dir, ETag refresh, content‑addressable storage, `Prune()`            |
| `internal/index`   | Parse `index.json`, resolve family → style → SHA‑256                              |
| `internal/platform`| Copy font bytes to OS dir + post‑install refresh (`fc‑cache`, `AddFontResourceEx`)|
| `internal/repo`    | Download `index.json` (ETag), verify SHA‑256 on payloads                          |

---

## 4  Local Cache Layout

```
$CACHE/fontget/
├── index.json            # catalogue + ETag
└── fonts/
    └── <sha256>.ttf      # content‑addressable blobs
```

* **Auto‑refresh:** if `index.json` is > 7 days old, `fontget` issues a conditional `GET` (ETag)
  before any command.  
* **Offline:** previously downloaded fonts install without internet.  
* **Prune:** `fontget cache prune` deletes blobs not used for *N* days (default 90).

---

## 5  Index Builder (runs in CI)

1. `GET https://api.github.com/repos/google/fonts/contents/ofl/`  
2. For each family dir (`type:"dir"`) fetch `METADATA.pb` and all `.ttf`/`.ttc` files.  
3. Emit compact `index.json`, e.g.:

```json
{
  "open-sans": {
    "version": "1.10",
    "license": "OFL",
    "styles": [
      {
        "file": "OpenSans-Regular.ttf",
        "sha256": "7e41…",
        "url": "https://raw.githubusercontent.com/google/fonts/abcdef123/ofl/opensans/OpenSans-Regular.ttf",
        "weight": 400,
        "italic": false
      }
    ]
  }
}
```

4. Upload `index.json` as release asset (or to GitHub Pages/CDN).

---

## 6  GitHub Actions

### `ci.yaml`

```yaml
name: ci
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./...
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v3
```

### `release.yaml`

```yaml
name: release
on:
  push:
    tags: ['v*.*.*']

jobs:
  build-index:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go run ./indexer > index.json
      - uses: actions/upload-artifact@v4
        with: { name: index.json, path: index.json }

  goreleaser:
    needs: build-index
    runs-on: ubuntu-latest
    permissions: { contents: write }
    steps:
      - uses: actions/checkout@v4
      - name: Download index
        uses: actions/download-artifact@v4
        with: { name: index.json, path: . }
      - uses: goreleaser/goreleaser-action@v5
        with: { version: latest, args: release --clean }
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### `.goreleaser.yml` (excerpt)

```yaml
project_name: fontget

builds:
  - main: ./cmd/fontget
    goos: [windows, linux, darwin]
    goarch: [amd64, arm64]
    flags: ['-trimpath']

archives:
  - format: tar.gz
    builds: [default]

checksum:
  name_template: 'checksums.txt'

brews:
  - github:
      owner: yourname
      name: homebrew-tap
    test: "fontget --help"

winget:
  package_identifier: yourname.fontget
  publisher: Your Name
  description: "Cross-platform font installer"
```

---

## 7  Milestones & Example Cursor Prompts

| Phase | Goal | Prompt |
|-------|------|--------|
| 0 | Scaffold project | “Create a Go module 'fontget' with Cobra root command and empty sub‑commands listed in section 3.” |
| 1 | Build indexer | “Implement indexer/main.go to scan Google Fonts OFL and output schema defined in section 5. Include unit tests.” |
| 2 | Cache package | “Add internal/cache with functions LoadIndex, RefreshIndex, GetFont, Prune. Use os.UserCacheDir().” |
| 3 | First install | “Make cmd/add.go resolve Open Sans via cache, download, verify sha256, save blob (no OS install yet).” |
| 4 | Platform install | “Write internal/platform/{windows,linux,macos}.go that copies font bytes to the correct dir and refreshes cache. Use build tags.” |
| 5 | Release pipeline | “Add .goreleaser.yml and workflows in section 6; make 'goreleaser check' pass.” |

---

## 8  Done Criteria

* Binaries run `fontget --help` on Windows 10, macOS 13+, Ubuntu 22.04.  
* `add` installs Open Sans Regular and it appears in the font picker without reboot.  
* Cache supports offline re‑installs; `fontget prune` frees space.  
* Tag ⇒ GitHub Release with checksums, `index.json`, archives; Homebrew & winget PRs open automatically.  
* Documentation includes README, man‑pages, usage examples.
