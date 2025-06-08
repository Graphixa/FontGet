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

`fontget` queries the Google Fonts repository for each font on demand, verifies SHA‑256, copies fonts into the correct system or user directory, and keeps an on‑disk cache for offline installs.

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
│   ├── cache/           # font blob cache
│   ├── platform/        # windows.go, linux.go, macos.go
│   └── repo/            # HTTP fetch, checksum, retries
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
| `internal/cache`   | Locate cache dir, content‑addressable storage, `Prune()`                          |
| `internal/platform`| Copy font bytes to OS dir + post‑install refresh (`fc‑cache`, `AddFontResourceEx`)|
| `internal/repo`    | Download font files, verify SHA‑256 on payloads                                   |

---

## 4  Local Cache Layout

```
$CACHE/fontget/
└── fonts/
    └── <sha256>.ttf      # content‑addressable blobs
```

* **Offline:** previously downloaded fonts install without internet.  
* **Prune:** `fontget cache prune` deletes blobs not used for *N* days (default 90).

---

## 5  Font Installation Process

1. `GET https://api.github.com/repos/google/fonts/contents/ofl/{font-name}`  
2. If the response is valid, extract the font files and download them.  
3. Verify SHA‑256 and install the fonts.

### Platform-Specific Implementation

#### Windows
- Uses `AddFontResource` and `RemoveFontResource` from GDI32
- Sends `WM_FONTCHANGE` message to notify applications
- Font directory: `%WINDIR%\Fonts`

#### Linux
- Uses `fc-cache` to update font cache
- Font directory: `~/.local/share/fonts`
- Requires `fontconfig` package

#### macOS
- Uses `atsutil` to manage font cache
- Font directory: `~/Library/Fonts`
- Requires administrative privileges for system-wide installation

---

## 6  GitHub Actions

### `ci.yaml`

```yaml
name: ci
on: [push, pull_request]

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - name: Install dependencies
        run: |
          if [ "${{ runner.os }}" = "Linux" ]; then
            sudo apt-get update
            sudo apt-get install -y fontconfig
          fi
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
  goreleaser:
    runs-on: ubuntu-latest
    permissions: { contents: write }
    steps:
      - uses: actions/checkout@v4
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
| 0 | Scaffold project | "Create a Go module 'fontget' with Cobra root command and empty sub‑commands listed in section 3." |
| 1 | Font installation | "Implement cmd/add.go to query the GitHub API for a specific font and download it." |
| 2 | Cache package | "Add internal/cache with functions GetFont, Prune. Use os.UserCacheDir()." |
| 3 | Platform install | "Write internal/platform/{windows,linux,macos}.go that copies font bytes to the correct dir and refreshes cache. Use build tags." |
| 4 | Release pipeline | "Add .goreleaser.yml and workflows in section 6; make 'goreleaser check' pass." |

---

## 8  Done Criteria

* Binaries run `fontget --help` on Windows 10, macOS 13+, Ubuntu 22.04.  
* `add` installs a font and it appears in the font picker without reboot.  
* Cache supports offline re‑installs; `fontget prune` frees space.  
* Tag ⇒ GitHub Release with checksums, archives; Homebrew & winget PRs open automatically.  
* Documentation includes README, man‑pages, usage examples.

### Testing Requirements

#### Windows
- Test on Windows 10/11
- Verify font installation in `%WINDIR%\Fonts`
- Check font appears in applications without reboot
- Test font removal and cache updates

#### Linux
- Test on Ubuntu 22.04
- Verify font installation in `~/.local/share/fonts`
- Check `fc-cache` updates
- Test font removal and cache updates
- Verify font appears in applications

#### macOS
- Test on macOS 13+
- Verify font installation in `~/Library/Fonts`
- Check `atsutil` cache updates
- Test font removal and cache updates
- Verify font appears in applications
