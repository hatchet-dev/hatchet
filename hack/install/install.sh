#!/bin/bash

# Hatchet CLI Installation Script
# Supports macOS (Darwin) and Linux on x86_64 and ARM64 architectures
# Usage: curl -fsSL https://install.hatchet.run | bash
# Or with specific version: curl -fsSL https://install.hatchet.run | bash -s -- v0.73.10

set -e

# Configuration
REPO="hatchet-dev/hatchet"
BINARY_NAME="hatchet"
INSTALL_DIR="/usr/local/bin"
GITHUB_API="https://api.github.com/repos/${REPO}"
GITHUB_RELEASES="https://github.com/${REPO}/releases/download"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Cleanup function
cleanup() {
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        # Delete specific files we created (no recursive deletion)
        rm -f "$TMP_DIR/$ARCHIVE_NAME" 2>/dev/null || true
        rm -f "$TMP_DIR/checksums.txt" 2>/dev/null || true
        rm -f "$TMP_DIR/$BINARY_NAME" 2>/dev/null || true
        # Remove directory only if empty
        rmdir "$TMP_DIR" 2>/dev/null || true
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Check prerequisites
check_prereqs() {
    log_info "Checking prerequisites..."

    # Check for curl or wget
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
        DOWNLOAD_CMD="curl -fsSL"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
        DOWNLOAD_CMD="wget -qO-"
    else
        log_error "Either curl or wget is required to install Hatchet CLI"
        exit 1
    fi

    # Check for tar
    if ! command -v tar >/dev/null 2>&1; then
        log_error "tar is required to install Hatchet CLI"
        exit 1
    fi

    # Check for shasum or sha256sum for checksum verification
    if command -v shasum >/dev/null 2>&1; then
        CHECKSUM_CMD="shasum -a 256"
    elif command -v sha256sum >/dev/null 2>&1; then
        CHECKSUM_CMD="sha256sum"
    else
        log_error "shasum or sha256sum is required for checksum verification"
        exit 1
    fi

    log_info "Using ${DOWNLOADER} for downloads"
}

# Detect platform and architecture
detect_platform() {
    log_info "Detecting platform and architecture..."

    # Detect OS
    case "$(uname -s)" in
        Darwin*)
            OS="Darwin"
            ARCHIVE_EXT="tar.gz"
            ;;
        Linux*)
            OS="Linux"
            ARCHIVE_EXT="tar.gz"
            # Check for WSL
            if uname -a | grep -qi 'microsoft'; then
                log_warn "WSL detected. Installation should work but may have issues with file permissions."
            fi
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            log_error "Hatchet CLI supports macOS and Linux only"
            exit 1
            ;;
    esac

    # Detect architecture
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)
            ARCH="x86_64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            log_error "Hatchet CLI supports x86_64 and arm64 only"
            exit 1
            ;;
    esac

    log_info "Detected platform: ${OS} ${ARCH}"
}

# Get latest version or use provided version
get_version() {
    local requested_version="$1"

    if [ -n "$requested_version" ]; then
        # Validate version format (should start with v)
        if [[ ! "$requested_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
            log_error "Invalid version format: $requested_version"
            log_error "Expected format: vX.Y.Z or vX.Y.Z-prerelease (e.g., v0.73.10 or v1.0.0-beta.1)"
            exit 1
        fi
        VERSION="$requested_version"
        log_info "Using requested version: $VERSION"
    else
        log_info "Fetching latest version..."
        if [ "$DOWNLOADER" = "curl" ]; then
            VERSION=$(curl -fsSL "${GITHUB_API}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        else
            VERSION=$(wget -qO- "${GITHUB_API}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        fi

        if [ -z "$VERSION" ]; then
            log_error "Failed to fetch latest version"
            exit 1
        fi

        log_info "Latest version: $VERSION"
    fi
}

# Download file with progress
download_file() {
    local url="$1"
    local output="$2"

    if [ "$DOWNLOADER" = "curl" ]; then
        if ! curl -fsSL --progress-bar "$url" -o "$output"; then
            log_error "Failed to download $url"
            return 1
        fi
    else
        if ! wget -q --show-progress "$url" -O "$output"; then
            log_error "Failed to download $url"
            return 1
        fi
    fi

    return 0
}

# Verify checksum
verify_checksum() {
    local file="$1"
    local checksums_file="$2"

    log_info "Verifying checksum..."

    # Extract expected checksum for this file
    local filename=$(basename "$file")
    local expected_checksum=$(grep "$filename" "$checksums_file" | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        log_error "Could not find checksum for $filename in checksums file"
        return 1
    fi

    # Calculate actual checksum
    local actual_checksum=$($CHECKSUM_CMD "$file" | awk '{print $1}')

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        log_error "Checksum verification failed!"
        log_error "Expected: $expected_checksum"
        log_error "Got:      $actual_checksum"
        return 1
    fi

    log_success "Checksum verified"
    return 0
}

# Download and install
install_hatchet() {
    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    log_info "Using temporary directory: $TMP_DIR"

    # Strip 'v' prefix from version for archive name
    VERSION_NO_V="${VERSION#v}"

    # Construct archive name (note: archive names don't have 'v' prefix)
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NO_V}_${OS}_${ARCH}.${ARCHIVE_EXT}"
    DOWNLOAD_URL="${GITHUB_RELEASES}/${VERSION}/${ARCHIVE_NAME}"
    CHECKSUMS_URL="${GITHUB_RELEASES}/${VERSION}/checksums.txt"

    log_info "Downloading Hatchet CLI ${VERSION}..."
    log_info "Archive: ${ARCHIVE_NAME}"

    # Download archive
    if ! download_file "$DOWNLOAD_URL" "$TMP_DIR/$ARCHIVE_NAME"; then
        log_error "Failed to download Hatchet CLI"
        log_error "URL: $DOWNLOAD_URL"
        exit 1
    fi

    # Download checksums
    log_info "Downloading checksums..."
    if ! download_file "$CHECKSUMS_URL" "$TMP_DIR/checksums.txt"; then
        log_error "Failed to download checksums"
        exit 1
    fi

    # Verify checksum
    if ! verify_checksum "$TMP_DIR/$ARCHIVE_NAME" "$TMP_DIR/checksums.txt"; then
        log_error "Checksum verification failed. Aborting installation."
        exit 1
    fi

    # Extract archive
    log_info "Extracting archive..."
    if ! tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"; then
        log_error "Failed to extract archive"
        exit 1
    fi

    # Verify binary exists
    if [ ! -f "$TMP_DIR/$BINARY_NAME" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi

    # Make binary executable
    chmod +x "$TMP_DIR/$BINARY_NAME"

    # Test binary works
    log_info "Testing binary..."
    if ! "$TMP_DIR/$BINARY_NAME" --version >/dev/null 2>&1; then
        log_warn "Binary test failed, but continuing with installation"
    fi

    # Install binary
    log_info "Installing to ${INSTALL_DIR}..."

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    else
        log_info "Installation requires administrator privileges"
        if ! sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"; then
            log_error "Failed to install binary to $INSTALL_DIR"
            log_error "You can manually copy the binary from: $TMP_DIR/$BINARY_NAME"
            exit 1
        fi
    fi

    # Verify installation
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        log_warn "Binary installed but not found in PATH"
        log_warn "Make sure ${INSTALL_DIR} is in your PATH"
        log_info "You may need to restart your terminal or run: export PATH=\"${INSTALL_DIR}:\$PATH\""
    else
        log_success "Hatchet CLI installed successfully!"

        # Show version
        log_info "Installed version:"
        "$BINARY_NAME" --version
    fi
}

# Main installation flow
main() {
    echo ""
    log_info "Installing Hatchet CLI..."
    echo ""

    check_prereqs
    detect_platform
    get_version "$1"
    install_hatchet

    echo ""
    log_success "Installation complete! 🎉"
    echo ""
    log_info "Get started by running: ${BINARY_NAME} --help"
    echo ""
}

# Run main with first argument (optional version)
main "$1"
