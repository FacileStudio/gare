#!/usr/bin/env bash
set -euo pipefail
BOOTSTRAP="https://get.facile.studio"
BOOTSTRAP_FALLBACK="https://raw.githubusercontent.com/FacileStudio/kit/main/install.sh"
command -v kit >/dev/null 2>&1 ||
  { curl -fsSL "$BOOTSTRAP" | bash || curl -fsSL "$BOOTSTRAP_FALLBACK" | bash
    export PATH="${KIT_BIN_DIR:-$HOME/.local/bin}:$PATH"; }
exec kit add gare "$@"
