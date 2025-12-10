#!/usr/bin/env bash
set -euo pipefail

# Generate shell completion scripts for terraci
# This script is called by goreleaser before building

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPLETIONS_DIR="${PROJECT_ROOT}/completions"

# Build terraci for the current platform
echo "Building terraci for completion generation..."
TERRACI_BIN="${PROJECT_ROOT}/terraci"
go build -o "$TERRACI_BIN" "${PROJECT_ROOT}/cmd/terraci"

# Create completions directory
mkdir -p "$COMPLETIONS_DIR"

echo "Generating shell completions..."

# Generate completions for each shell
"$TERRACI_BIN" completion bash > "${COMPLETIONS_DIR}/terraci.bash"
echo "  - bash: ${COMPLETIONS_DIR}/terraci.bash"

"$TERRACI_BIN" completion zsh > "${COMPLETIONS_DIR}/_terraci"
echo "  - zsh: ${COMPLETIONS_DIR}/_terraci"

"$TERRACI_BIN" completion fish > "${COMPLETIONS_DIR}/terraci.fish"
echo "  - fish: ${COMPLETIONS_DIR}/terraci.fish"

"$TERRACI_BIN" completion powershell > "${COMPLETIONS_DIR}/terraci.ps1"
echo "  - powershell: ${COMPLETIONS_DIR}/terraci.ps1"

# Clean up temporary binary
rm -f "$TERRACI_BIN"

echo "Shell completions generated successfully!"
