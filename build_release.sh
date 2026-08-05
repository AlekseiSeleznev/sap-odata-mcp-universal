#!/bin/bash

set -euo pipefail

# Build release binaries for OData MCP Bridge v1.5.1

VERSION="v1.5.1"
BINARY_NAME="sap-odata-mcp-universal"
BUILD_DIR="dist"

echo "Building OData MCP Bridge ${VERSION} binaries..."

# Darwin release artifacts require the Go linker LC_UUID load command.
bash "$(dirname "${BASH_SOURCE[0]}")/scripts/require-go-toolchain.sh" go

# Clean previous builds
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Build function
build_platform() {
    local GOOS=$1
    local GOARCH=$2
    local EXT=$3
    local OUTPUT="${BUILD_DIR}/${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}${EXT}"

    echo "Building for ${GOOS}/${GOARCH}..."
    GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags="-s -w" -o ${OUTPUT} cmd/sap-odata-mcp-universal/main.go

    if [ $? -eq 0 ]; then
        echo "✅ Built ${OUTPUT}"
        # Create a tar.gz archive for non-Windows platforms
        if [ "${GOOS}" != "windows" ]; then
            tar -czf "${OUTPUT}.tar.gz" -C "${BUILD_DIR}" "$(basename ${OUTPUT})"
            rm ${OUTPUT}
            echo "📦 Archived as ${OUTPUT}.tar.gz"
        else
            # Create a zip archive for Windows
            zip -j "${OUTPUT}.zip" "${OUTPUT}"
            rm ${OUTPUT}
            echo "📦 Archived as ${OUTPUT}.zip"
        fi
    else
        echo "❌ Failed to build for ${GOOS}/${GOARCH}"
    fi
}

# Build for multiple platforms
build_platform "darwin" "amd64" ""        # macOS Intel
build_platform "darwin" "arm64" ""        # macOS Apple Silicon
build_platform "linux" "amd64" ""         # Linux 64-bit
build_platform "linux" "arm64" ""         # Linux ARM64
build_platform "linux" "arm" ""           # Linux ARM
build_platform "windows" "amd64" ".exe"   # Windows 64-bit
build_platform "windows" "386" ".exe"     # Windows 32-bit
build_platform "windows" "arm64" ".exe"   # Windows ARM64

echo ""
echo "Build complete! Binaries are in the ${BUILD_DIR} directory."
echo ""
echo "Release notes for v1.5.0:"
echo "========================="
echo "✨ New Features:"
echo "- Streamable HTTP transport support (modern MCP protocol 2024-11-05)"
echo "- SAP GUID filtering fix: Automatically formats GUID values as guid'...' for SAP services"
echo ""
echo "🐛 Bug Fixes:"
echo "- Fixed GUID filtering for SAP OData services"
echo "- Added regression tests to prevent binary ENOENT errors"
echo ""
echo "📚 Documentation:"
echo "- Added comprehensive implementation guide"
echo "- Added GitHub Discussions link"
echo ""
echo "Next steps:"
echo "1. Test the binaries"
echo "2. Create GitHub release with: gh release create ${VERSION} ${BUILD_DIR}/* --title 'OData MCP Bridge v1.5.0' --notes-file RELEASE_NOTES.md"
