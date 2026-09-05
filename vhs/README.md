# FontGet VHS recordings

Terminal demos recorded with [VHS](https://github.com/charmbracelet/vhs). Tapes live in `tapes/`; rendered gif/mp4/webm files land in `exports/`.

This does **not** embed demos in the GitHub README. Re-render when the CLI UI changes.

## Tapes

| Tape | What it shows |
|------|----------------|
| `hero` | `search` → `add` → `list` → `remove` (`google.roboto`) |
| `browse` | Interactive font browser TUI |

## Record

From the **repo root** (Git Bash, WSL, macOS, or Linux). Docker is the path that works on Windows.

```sh
sh scripts/vhs-record.sh          # both tapes
sh scripts/vhs-record.sh hero
sh scripts/vhs-record.sh browse
```

The script builds a Linux `fontget`, uses a throwaway `$HOME` under `vhs/.work/`, pre-seeds config (first-run complete, updates off), and warms the font manifest before recording. Do not set `FONTGET_ACCEPT_*` during the visible tape — those flags print a terms banner on every command.

### Docker (Windows)

Requires [Docker Desktop](https://www.docker.com/products/docker-desktop/). The script pulls `ghcr.io/charmbracelet/vhs` and mounts the repo at `/vhs`.

### Native VHS (Linux / macOS / WSL)

Install [VHS](https://github.com/charmbracelet/vhs) plus **ttyd** and **ffmpeg**, then run the same script. It will use `vhs` on your `PATH` when Docker is not available.

## Outputs

```
vhs/exports/hero.gif
vhs/exports/hero.mp4
vhs/exports/hero.webm
vhs/exports/browse.gif
vhs/exports/browse.mp4
vhs/exports/browse.webm
```

`vhs/.work/` is gitignored (temp home + recorder binary).
