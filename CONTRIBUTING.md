# Contributing to chriscast

Thanks for helping out. chriscast is a Go-based Linux file launcher. Keep changes small, tested, and aligned with the locked design decisions.

## 1. Filing Issues

Open issues at the project tracker. Pick a template:

- **Bug**: include OS + DE, kernel, Go version, GTK version, reproduction steps, expected vs actual, logs (`~/.local/share/chriscast/log` if present), and minimal config.
- **Feature**: describe the use case first, then proposed UX. Note whether it touches core indexing, scoring, UI, or daemon. Check the decision table (see section 9) before proposing changes that contradict it.

One issue per topic. Search existing issues first.

## 2. Dev Setup

```bash
git clone <repo-url> chriscast
cd chriscast
```

GTK3 build deps (Debian/Ubuntu):

```bash
sudo apt install -y \
  build-essential pkg-config \
  libgtk-3-dev libglib2.0-dev libcairo2-dev \
  libgdk-pixbuf-2.0-dev libpango1.0-dev \
  libx11-dev libxkbcommon-dev
```

Go 1.22+ required.

Build:

```bash
make build       # core CLI binary, no GTK
make build-ui    # full binary with GTK UI (requires -tags gtk)
```

## 3. Project Layout

```
cmd/chriscast/           # main entry point, CLI flags, wiring
internal/
  config/                # config file load/save, defaults
  scanner/               # filesystem walkers, ignore rules
  trie/                  # prefix index
  trigram/               # n-gram index for fuzzy
  score/                 # ranking, recency, frecency
  search/                # query parsing + index fan-out
  index/                 # index orchestration, persistence
  store/                 # on-disk store (bolt/badger/etc.)
  normalize/             # path + query normalization (unicode, case)
  ui/                    # GTK3 window, results list (build tag: gtk)
  watcher/               # inotify/fsnotify hooks
  hotkey/                # global hotkey binding
  daemon/                # background service, IPC
```

## 4. Coding Conventions

- Format with `gofumpt` (stricter than gofmt).
- Lint with `golangci-lint run` before pushing. Fix all warnings or justify in PR.
- Build tags: GUI code lives behind `//go:build gtk`. Core must compile without `-tags gtk`.
- Errors: wrap with `fmt.Errorf("context: %w", err)`. No `panic` outside `main`.
- Logging: structured via the project logger; no `fmt.Println` in library code.
- Exported identifiers need doc comments. Keep packages small and cohesive.

## 5. Adding a Feature — Checklist

- [ ] Unit tests for new logic; table-driven where reasonable.
- [ ] Integration test if it crosses package boundaries.
- [ ] Package `doc.go` updated (or created) with a one-paragraph summary.
- [ ] README usage section updated if user-facing.
- [ ] Config keys documented in `internal/config`.
- [ ] No regression in `make build` (core stays GTK-free).
- [ ] Benchmarks added if it touches `score`, `search`, `trie`, or `trigram`.

## 6. Running Tests

```bash
go test ./...                    # core tests
go test -tags gtk ./internal/ui/...      # UI tests
go test -tags gtk ./internal/daemon/...  # daemon tests
go test -race ./...              # race detector on hot paths
go test -bench=. ./internal/score/...    # benchmarks
```

UI and daemon tests are gated behind `-tags gtk` and require an X/Wayland session or `xvfb-run`.

## 7. Commit Message Format

Conventional Commits. Subject ≤ 72 chars, imperative mood.

```
feat: add fuzzy matching to scanner
fix: handle empty config file
refactor: extract score weights to config
docs: clarify hotkey setup
test: cover trigram edge cases
perf: cache normalized paths
chore: bump go.mod to 1.22
```

Breaking changes: append `!` (e.g. `feat!: rename --db flag`) and explain in the body.

## 8. PR Process

1. Branch from `main`: `git checkout -b feat/short-slug`.
2. Run `gofumpt`, `golangci-lint run`, `go test ./...` locally.
3. Push and open a PR. Title = Conventional Commit style.
4. Link the issue with `Closes #N` or `Refs #N`.
5. Wait for CI green. No merging red builds.
6. Squash on merge unless commit history is intentionally clean.
7. One reviewer approval minimum.

## 9. Where Decisions Live

The **locked decision table** in the initial design document is the source of truth for architecture, scoring weights, index format, and UX contracts. Do not contradict it in a PR — propose an amendment first via issue.

`session-ses_1afe.md` is **historical only**: a record of how we arrived at the locked decisions. Do not cite it as authoritative; do not update it.

New decisions get appended to the locked table with a date and rationale.

## 10. Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). See `CODE_OF_CONDUCT.md` in the repo root. Report violations to the maintainers listed there.
