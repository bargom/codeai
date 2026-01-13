#!/bin/sh
set -e

# CodeAI Installer
# Usage: curl -sSL https://raw.githubusercontent.com/bargom/codeai/main/install.sh | sh

REPO="github.com/bargom/codeai"
BINARY_NAME="codeai"
INSTALL_DIR="/usr/local/bin"

echo "Installing CodeAI..."

# Check for Go
if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go is not installed."
    echo "Please install Go 1.24+ from https://go.dev/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)

if [ "$GO_MAJOR" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 24 ]; }; then
    echo "Error: Go 1.24+ is required (found $GO_VERSION)"
    exit 1
fi

# Install using go install from main branch
echo "Downloading and building CodeAI..."
GOPROXY=direct go install "$REPO/cmd/codeai@main"

# Find the binary
GO_BIN=$(go env GOPATH)/bin
if [ ! -f "$GO_BIN/$BINARY_NAME" ]; then
    GO_BIN=$(go env GOBIN)
fi

if [ ! -f "$GO_BIN/$BINARY_NAME" ]; then
    echo "Error: Binary not found after installation"
    exit 1
fi

# Try to install to /usr/local/bin for system-wide access
if [ -w "$INSTALL_DIR" ]; then
    cp "$GO_BIN/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    echo "Installed to $INSTALL_DIR/$BINARY_NAME"
elif command -v sudo >/dev/null 2>&1; then
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo cp "$GO_BIN/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    echo "Installed to $INSTALL_DIR/$BINARY_NAME"
else
    echo "Installed to $GO_BIN/$BINARY_NAME"
    echo ""
    echo "Add to your PATH by running:"
    echo "  echo 'export PATH=\"\$PATH:$GO_BIN\"' >> ~/.zshrc && source ~/.zshrc"
fi

echo ""
echo "CodeAI installed successfully!"
echo "Run 'codeai --help' to get started."
