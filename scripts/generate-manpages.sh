#!/usr/bin/env bash
set -euo pipefail

# Generate man pages for terraci and xterraci
# This script is called by goreleaser before building

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MANPAGES_DIR="${PROJECT_ROOT}/manpages"

generate_manpages() {
  local name="$1"
  local cmd_dir="$2"

  echo "Building ${name}..."
  local bin="${PROJECT_ROOT}/${name}"
  go build -o "$bin" "${PROJECT_ROOT}/${cmd_dir}"

  echo "Generating ${name} man pages..."
  "$bin" man -d "$MANPAGES_DIR"

  rm -f "$bin"
}

generate_manpages "terraci" "cmd/terraci"
generate_manpages "xterraci" "cmd/xterraci"

echo "Man pages generated successfully!"
