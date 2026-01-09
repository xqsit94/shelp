#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

REPO="xqsit94/shelp"
BINARY_NAME="shelp"
INSTALL_DIR="$HOME/.local/bin"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "darwin"
            ;;
        Linux*)
            echo "linux"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
}

get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl &> /dev/null; then
        version=$(curl -sL "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget &> /dev/null; then
        version=$(wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    if [ -z "$version" ]; then
        log_error "Failed to fetch latest version from GitHub"
        exit 1
    fi

    echo "$version"
}

check_binary_exists() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local binary_name="${BINARY_NAME}-${os}-${arch}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"

    if command -v curl &> /dev/null; then
        local status=$(curl -sL -o /dev/null -w "%{http_code}" "$download_url")
        [ "$status" = "200" ] || [ "$status" = "302" ]
    else
        wget --spider -q "$download_url" 2>/dev/null
    fi
}

install_binary() {
    local version="$1"
    local os="$2"
    local arch="$3"
    local binary_name="${BINARY_NAME}-${os}-${arch}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"
    local temp_file=$(mktemp)

    log_info "Downloading ${BINARY_NAME} ${version} for ${os}/${arch}..."

    if command -v curl &> /dev/null; then
        curl -sL -o "$temp_file" "$download_url"
    else
        wget -q -O "$temp_file" "$download_url"
    fi

    mkdir -p "$INSTALL_DIR"

    mv "$temp_file" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    log_success "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

verify_installation() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        log_warning "${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo ""
        echo "Then restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"
        echo ""
    fi

    if command -v shelp &> /dev/null; then
        log_success "Installation verified! Run 'shelp --help' to get started."
    else
        log_info "Installation complete. Please update your PATH and restart your terminal."
    fi
}

main() {
    echo ""
    echo "🚀 shelp Installer"
    echo "=================="
    echo ""

    local os=$(detect_os)
    local arch=$(detect_arch)

    log_info "Detected OS: ${os}"
    log_info "Detected Architecture: ${arch}"

    local version=$(get_latest_version)
    log_info "Latest version: ${version}"

    if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
        local current_version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>/dev/null | head -1 | awk '{print $NF}')
        if [ "$current_version" = "${version#v}" ]; then
            log_success "You already have the latest version (${version})"
            exit 0
        else
            log_info "Updating from ${current_version} to ${version}..."
        fi
    fi

    if ! check_binary_exists "$version" "$os" "$arch"; then
        log_error "Binary not found for ${os}/${arch} version ${version}"
        log_info "Please check releases at: https://github.com/${REPO}/releases"
        exit 1
    fi

    install_binary "$version" "$os" "$arch"
    verify_installation

    echo ""
    log_success "Installation complete! 🎉"
    echo ""
    echo "Get started:"
    echo "  shelp \"list all files in current directory\""
    echo ""
}

main "$@"
