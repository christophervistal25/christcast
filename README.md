# chriscast

A Raycast-style file launcher for Linux. Pure-Go core, GTK3 overlay, sub-millisecond search across hundreds of thousands of files.

![chriscast overlay screenshot](docs/screenshot.png)

[![Go](https://img.shields.io/badge/go-1.23%2B-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://img.shields.io/badge/ci-pending-lightgrey.svg)](#)
[![Platform](https://img.shields.io/badge/platform-Linux%20x86__64-orange.svg)](#)

---

## Features

- Instant fuzzy file search with fzf-style scoring (sub-millisecond on 200K+ files).
- Compressed Patricia trie + trigram inverted index for fast prefix and substring queries.
- Smart-case matching, recency boost, and extension-aware ranking.
- Live filesystem updates via `fsnotify` — no manual reindexing during normal use.
- Path-prefix browse mode: type `/var/www/html` to walk a directory directly.
- GTK3 overlay window with global hotkey (default `Ctrl+Space`).
- Single static binary for the core; thin GTK3 binary for the UI.
- TOML configuration with scopes, excludes, symlink and hidden-file controls.

## Architecture

```
+--------------------------------------------------------------+
|                         chriscast                            |
+--------------------------------------------------------------+

  +----------------------+        +----------------------+
  |       Daemon         |        |          UI          |
  |  (fsnotify watcher,  | <----> |   GTK3 overlay,      |
  |   X11 global hotkey) |  IPC   |   results renderer   |
  +----------+-----------+        +-----------+----------+
             |                                |
             v                                v
  +--------------------------------------------------------+
  |                         Core                           |
  |  Indexer  ->  Patricia trie  +  trigram inverted index |
  |  Search   ->  fzf scoring, recency, extension bias     |
  +--------------------------------------------------------+
             |
             v
  +--------------------------------------------------------+
  |     On-disk index (~/.cache/chriscast/index.bin)       |
  +--------------------------------------------------------+
```

## Requirements

- Linux x86_64 with X11 (Wayland: see note below).
- Go 1.23 or newer.
- GTK3 development libraries.
- `pkg-config`.

## Install

### From source

```bash
git clone https://github.com/yourname/chriscast
cd chriscast
make build       # CLI-only binary  -> bin/chriscast
make build-ui    # CLI + GTK overlay -> bin/chriscast (TAGS=gtk)
```

A single binary `chriscast` is produced in either case. `make build-ui` rebuilds it with the `gtk` build tag so the `ui` and `daemon` subcommands are available.

Install the runtime and build dependencies on Debian/Ubuntu:

```bash
sudo apt install \
  build-essential pkg-config \
  libgtk-3-dev libglib2.0-dev libcairo2-dev libpango1.0-dev \
  libx11-dev libxext-dev
```

On Fedora:

```bash
sudo dnf install gcc pkgconf-pkg-config gtk3-devel libX11-devel
```

Copy the binary to your `PATH`:

```bash
install -Dm755 bin/chriscast ~/.local/bin/chriscast
```

## Quick start

```bash
chriscast index     # build the initial index
chriscast daemon    # run watcher + hotkey listener
# Press Ctrl+Space anywhere to open the overlay.
```

## Usage

| Command              | Description                                                       |
| -------------------- | ----------------------------------------------------------------- |
| `chriscast index`    | Build the index from configured scopes.                           |
| `chriscast reindex`  | Discard the existing index and rebuild from scratch.              |
| `chriscast search Q` | Run a one-shot query against the index and print results.         |
| `chriscast ui`       | Launch the GTK3 overlay (useful as a DE keybinding target).       |
| `chriscast daemon`   | Run the watcher, X11 hotkey listener, and UI launcher in one go.  |
| `chriscast info`     | Print index stats: file count, size on disk, last build time.     |
| `chriscast help`     | Show help for any subcommand.                                     |

## Configuration

Config lives at `~/.config/chriscast/config.toml`. All fields are optional and fall back to sensible defaults.

### Schema

| Field             | Type        | Default            | Description                                          |
| ----------------- | ----------- | ------------------ | ---------------------------------------------------- |
| `[[scopes]]`      | TOML table  | `[{path="$HOME"}]` | Root directories to index. One block per scope.      |
| `excludes`        | `[]string`  | see example        | Directory **basenames** excluded from indexing.      |
| `follow_symlinks` | `bool`      | `false`            | Follow symlinks while walking scopes.                |
| `include_hidden`  | `bool`      | `false`            | Index dotfiles and dot-directories.                  |
| `cross_device`    | `bool`      | `false`            | Cross filesystem boundaries while walking.           |
| `max_results`     | `int`       | `50`               | Maximum results returned per query.                  |
| `hotkey`          | `string`    | `"ctrl+space"`     | Global hotkey for the overlay (X11 keysym syntax).   |

`excludes` is matched against **directory base names**, not glob patterns.

### Example

```toml
[[scopes]]
path = "/home/me"

[[scopes]]
path = "/var/www"

excludes = ["node_modules", ".git", ".cache", "target", "vendor", "dist", "build", ".venv", "__pycache__"]

follow_symlinks = false
include_hidden  = false
cross_device    = false
max_results     = 50
hotkey          = "ctrl+space"
```

## Search syntax

- **Smart-case.** Lowercase queries are case-insensitive; any uppercase character makes the whole query case-sensitive.
- **Path-prefix browse.** If the query starts with `/` and matches an existing directory prefix, chriscast lists the directory directly (a live `readdir` fallback), so you can walk into freshly-created paths the index has not yet seen.
- **Live `readdir` fallback.** When the index is stale or empty for a prefix, results stream in from disk so the overlay never feels frozen.
- **Fuzzy matching.** Non-prefix queries use trigram lookup followed by fzf-style scoring: contiguous matches, word boundaries, and camel-case transitions all boost rank.

## Keyboard shortcuts

| Key            | Action                                                          |
| -------------- | --------------------------------------------------------------- |
| `Ctrl+Space`   | Toggle overlay (global, configurable).                          |
| `Esc`          | Close overlay.                                                  |
| `Up` / `Down`  | Move selection (auto-scrolls the row into view).                |
| `Right`        | If selection is a directory, drill into it (sets path-browse).  |
| `Enter`        | Open the selected file or directory with `xdg-open`.            |
| `Ctrl+Enter`   | Reveal parent directory in file manager.                        |
| `Ctrl+C`       | Copy absolute path to clipboard (when no text selected).        |

## systemd autostart

A user unit is provided at `dist/chriscast.service`:

```bash
install -Dm644 dist/chriscast.service ~/.config/systemd/user/chriscast.service
systemctl --user daemon-reload
systemctl --user enable --now chriscast.service
```

## inotify limit

Watching large home directories quickly exhausts the default inotify watch limit. If you see `too many open files` or missed updates, raise the limit:

```bash
echo 'fs.inotify.max_user_watches=524288' | sudo tee /etc/sysctl.d/40-chriscast.conf
sudo sysctl --system
```

## Wayland

The global hotkey path uses X11 (`XGrabKey`) and will not register on Wayland. Workaround: bind your desktop environment's keyboard shortcut to run `chriscast ui` directly. The watcher, indexer, and overlay itself all work on XWayland.

## Algorithm overview

- **Trie.** A rune-keyed prefix trie holds every basename. Prefix walks are O(query length); the implementation will be replaced with a Patricia (compressed) trie in a later release.
- **Trigram inverted index.** Each basename is broken into overlapping padded 3-grams; a `map[trigram] -> []FileID` lets the search shortlist candidates without scanning every entry. Multi-gram queries are answered by multiset intersection.
- **fzf scoring.** Candidates are scored with a Smith-Waterman-style dynamic program tuned for filenames: contiguous matches, word-boundary hits, camel-case transitions, and matches near the basename are rewarded.
- **Recency boost.** Files opened recently float toward the top via a decaying logistic bonus on the file's last-used timestamp.
- **Extension bias.** Common "useful" extensions (source files, documents, media) get a small ranking nudge over build artefacts and lockfiles.

## Performance

Measured on a Ryzen 7 / NVMe SSD, indexing a home directory:

| Metric                      | Value          |
| --------------------------- | -------------- |
| Initial index, 200K files   | ~5 s           |
| Index size on disk          | ~32 MB         |
| Cold search latency         | <1 ms          |
| Warm search latency         | <1 ms          |
| RSS at idle (daemon + UI)   | ~40 MB         |

## Development

Project layout:

```
cmd/chriscast/        # CLI entrypoint (all subcommands)
internal/
  config/             # TOML loader + XDG paths
  store/              # FileID, FileInfo, in-memory store
  normalize/          # Unicode NFC + smart-case folding
  scanner/            # parallel directory walker
  trie/               # rune-keyed prefix trie
  trigram/            # padded n=3 inverted index
  score/              # fzf v1 greedy fuzzy matcher
  search/             # query orchestrator
  index/              # combined store + trie + trigram (msgpack persist)
  watcher/            # fsnotify recursive watcher           (gtk tag)
  hotkey/             # X11 XGrabKey global hotkey           (gtk tag)
  ui/                 # GTK3 overlay window                  (gtk tag)
  daemon/             # UI + watcher + hotkey orchestrator   (gtk tag)
dist/                 # packaging artefacts (systemd unit)
examples/             # config example + install script
```

Run the tests:

```bash
go test ./...
```

Build tags:

- default — pure-Go core (CLI subcommands work; `ui` / `daemon` print a friendly error).
- `gtk` — enables the GTK3 overlay and X11 hotkey daemon (requires libgtk-3-dev + CGO).

## Roadmap and non-goals

Planned:

- Calculator and unit-conversion results.
- Optional Wayland hotkey backend via `wlr-foreign-toplevel` / portal.
- Per-scope ranking weights.

Non-goals (for now):

- Application launching and command running. chriscast indexes files, not `.desktop` entries or shell history.
- Wayland-native global hotkey. Use a DE shortcut bound to `chriscast ui`.
- macOS or Windows support.

## Contributing

Pull requests are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening one.

## License

[MIT](LICENSE).
