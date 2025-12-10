#!/usr/bin/env bash
set -euo pipefail

# Generate man pages for terraci
# This script is called by goreleaser before building

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MANPAGES_DIR="${PROJECT_ROOT}/manpages"

# Build terraci for the current platform
echo "Building terraci for man page generation..."
TERRACI_BIN="${PROJECT_ROOT}/terraci"
go build -o "$TERRACI_BIN" "${PROJECT_ROOT}/cmd/terraci"

# Generate man pages
echo "Generating man pages..."
"$TERRACI_BIN" man -d "$MANPAGES_DIR"

# Clean up temporary binary
rm -f "$TERRACI_BIN"

echo "Man pages generated successfully!"
