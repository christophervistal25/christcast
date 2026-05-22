#!/usr/bin/env bash
#
# chriscast installer
#
# Builds chriscast with the GTK UI, installs the binary to ~/.local/bin,
# and registers the systemd --user service so it starts on login.
#
# Usage:
#   chmod +x examples/install.sh
#   ./examples/install.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

BIN_DIR="${HOME}/.local/bin"
UNIT_DIR="${HOME}/.config/systemd/user"
CONFIG_DIR="${HOME}/.config/chriscast"

echo ">> Checking build dependencies..."
if ! command -v pkg-config >/dev/null 2>&1; then
  echo "ERROR: pkg-config is not installed." >&2
  echo "  Debian/Ubuntu: sudo apt install pkg-config" >&2
  echo "  Fedora:        sudo dnf install pkgconf-pkg-config" >&2
  echo "  Arch:          sudo pacman -S pkgconf" >&2
  exit 1
fi

if ! pkg-config --exists gtk+-3.0; then
  echo "ERROR: GTK 3 development headers not found (libgtk-3-dev)." >&2
  echo "  Debian/Ubuntu: sudo apt install libgtk-3-dev" >&2
  echo "  Fedora:        sudo dnf install gtk3-devel" >&2
  echo "  Arch:          sudo pacman -S gtk3" >&2
  exit 1
fi

echo ">> Building UI binary (make build-ui)..."
make build-ui

if [[ ! -x "bin/chriscast" ]]; then
  echo "ERROR: bin/chriscast was not produced by 'make build-ui'." >&2
  exit 1
fi

echo ">> Installing binary to ${BIN_DIR}..."
mkdir -p "$BIN_DIR"
install -m 0755 bin/chriscast "${BIN_DIR}/chriscast"

echo ">> Installing systemd user unit to ${UNIT_DIR}..."
mkdir -p "$UNIT_DIR"
install -m 0644 dist/chriscast.service "${UNIT_DIR}/chriscast.service"

echo ">> Reloading systemd user units..."
systemctl --user daemon-reload

echo ">> Enabling and starting chriscast.service..."
systemctl --user enable chriscast.service
systemctl --user restart chriscast.service

mkdir -p "$CONFIG_DIR"

cat <<EOF

chriscast installed successfully.

Next steps:
  1. Create / edit your config at:
       ${CONFIG_DIR}/config.toml
     (See examples/config.example.toml for a starting point.)

  2. Make sure ${BIN_DIR} is on your PATH.

  3. Press Ctrl+Space to summon the launcher.

  Service status: systemctl --user status chriscast.service
  Service logs:   journalctl --user -u chriscast.service -f

EOF
