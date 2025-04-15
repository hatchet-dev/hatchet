#!/bin/bash

# Install typedoc and the markdown plugin
npm install -g typedoc typedoc-plugin-markdown

# Get the absolute path to the src directory
SRC_DIR="$(pwd)/src"
DOCS_DIR="$(pwd)/../../frontend/docs/pages/sdks/typescript/api"

# Remove existing docs directory if it exists
if [ -d "$DOCS_DIR" ]; then
    rm -rf "$DOCS_DIR"
fi

# Create docs directory if it doesn't exist
mkdir -p "$DOCS_DIR"

# Generate documentation
typedoc \
    --out "$DOCS_DIR" \
    --name "Hatchet TypeScript SDK" \
    --readme none \
    --theme markdown \
    --hideGenerator \
    --excludePrivate \
    --excludeProtected \
    --excludeExternals \
    "$SRC_DIR"

echo "Documentation generated at $DOCS_DIR" 