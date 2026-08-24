#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec bash "$SCRIPT_DIR/rum-dev-owner-flow-verifier.sh" "670eb7bfe646bcd4f1aca80b003323da7b801433"
