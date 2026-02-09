#!/bin/bash

# Hatchet CLI Installation Script
# Supports macOS (Darwin) and Linux on x86_64 and ARM64 architectures
#
# Usage:
#   Basic installation: curl -fsSL https://install.hatchet.run | bash
#   Specific version: curl -fsSL https://install.hatchet.run | bash -s -- v0.73.10
#   Custom directory: curl -fsSL https://install.hatchet.run | INSTALL_DIR="$HOME/bin" bash
#
# Installation directories (in order of preference):
#   1. $INSTALL_DIR (if set via environment variable)
#   2. $HOME/.local/bin (if in PATH, no sudo required)
#   3. /usr/local/bin (may require sudo)

set -e

# Configuration
REPO="hatchet-dev/hatchet"
BINARY_NAME="hatchet"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
GITHUB_API="https://api.github.com/repos/${REPO}"
GITHUB_RELEASES="https://github.com/${REPO}/releases/download"

# Colors for output - Hatchet color scheme
# Matching cmd/hatchet-cli/cli/internal/drivers/docker/hatchet_lite.go styles
BOLD_BLUE='\033[1;34m'   # Bold blue for success (matches #3392FF)
CYAN='\033[0;36m'        # Cyan for info (matches #A5C5E9)
YELLOW='\033[1;33m'      # Yellow for warnings
RED='\033[1;31m'         # Red for errors
NC='\033[0m'             # No Color

# Logging functions - colored labels with regular text
log_info() {
    printf "${CYAN}[INFO]${NC} %s\n" "$1"
}

log_success() {
    printf "${BOLD_BLUE}[SUCCESS]${NC} %s\n" "$1"
}

log_warn() {
    printf "${YELLOW}[WARNING]${NC} %s\n" "$1" >&2
}

log_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
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

# Detect best installation directory
detect_install_dir() {
    log_info "Determining installation directory..."

    # Check if INSTALL_DIR is already set via environment variable
    if [ -n "$INSTALL_DIR" ]; then
        log_info "Using custom installation directory from INSTALL_DIR env var: ${INSTALL_DIR}"
        # Ensure directory exists
        if [ ! -d "$INSTALL_DIR" ]; then
            if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
                log_error "Cannot create custom installation directory: $INSTALL_DIR"
                exit 1
            fi
        fi
        return
    fi

    # Check if ~/.local/bin is in PATH and is/can be writable
    if echo "$PATH" | grep -q "$USER_INSTALL_DIR"; then
        # ~/.local/bin is in PATH, check if it exists and is writable
        if [ -d "$USER_INSTALL_DIR" ] && [ -w "$USER_INSTALL_DIR" ]; then
            INSTALL_DIR="$USER_INSTALL_DIR"
            log_info "Using user installation directory: ${INSTALL_DIR} (no sudo required)"
            return
        elif [ ! -d "$USER_INSTALL_DIR" ]; then
            # Directory doesn't exist but is in PATH, try to create it
            if mkdir -p "$USER_INSTALL_DIR" 2>/dev/null; then
                INSTALL_DIR="$USER_INSTALL_DIR"
                log_info "Created and using user installation directory: ${INSTALL_DIR} (no sudo required)"
                return
            fi
        fi
    fi

    # Fall back to system-wide installation
    INSTALL_DIR="$DEFAULT_INSTALL_DIR"
    log_info "Will install to system directory: ${INSTALL_DIR}"

    # Check if we'll need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        log_info "System directory requires administrator privileges (sudo)"
        if ! echo "$PATH" | grep -q "$USER_INSTALL_DIR"; then
            log_info "Tip: To avoid sudo in the future, add ~/.local/bin to your PATH:"
            log_info "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
            log_info "  (or ~/.zshrc if using zsh)"
        fi
    fi
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
    local expected_checksum=$(grep -F " ${filename}" "$checksums_file" | head -1 | awk '{print $1}')

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
    log_info "Installing Hatchet CLI to: ${INSTALL_DIR}/${BINARY_NAME}"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        if ! mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"; then
            log_error "Failed to install binary to $INSTALL_DIR"
            exit 1
        fi
        log_success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
    else
        log_info "Requesting administrator privileges to install to system directory..."
        if ! sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"; then
            log_error "Failed to install binary to $INSTALL_DIR"
            log_error "You can manually copy the binary from: $TMP_DIR/$BINARY_NAME"
            exit 1
        fi
        log_success "Installed to ${INSTALL_DIR}/${BINARY_NAME} (with sudo)"
    fi

    # Verify installation
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        log_warn "Binary installed but not found in PATH"
        log_warn "Make sure ${INSTALL_DIR} is in your PATH"
        log_info "You may need to restart your terminal or run: export PATH=\"${INSTALL_DIR}:\$PATH\""
    else
        # Show version
        VERSION_OUTPUT=$("$BINARY_NAME" --version 2>&1)
        log_info "Installed version: ${VERSION_OUTPUT}"
    fi
}

# Main installation flow
main() {
    log_info "Installing Hatchet CLI..."

    check_prereqs
    detect_platform
    detect_install_dir
    get_version "$1"
    install_hatchet

    log_success "Installation complete! 🎉"
}

# Run main with first argument (optional version)
main "$1"
