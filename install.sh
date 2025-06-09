#!/bin/bash

set -e

# GitHub repository information
REPO="mangelajo/jumpstarter-lab-config"
BINARY_NAME="jumpstarter-lab-config"

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    # Detect OS
    case "$(uname -s)" in
        Darwin*)
            os="darwin"
            ;;
        Linux*)
            os="linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            os="windows"
            ;;
        *)
            echo "Error: Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            echo "Error: Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release version from GitHub API
get_latest_version() {
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "${latest_url}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "${latest_url}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo "Error: Neither curl nor wget is available"
        exit 1
    fi
}

# Download and install binary
install_binary() {
    local platform="$1"
    local version="$2"
    local extension=""
    
    # Add .exe extension for Windows
    if [[ "${platform}" == windows-* ]]; then
        extension=".exe"
    fi
    
    local binary_filename="${BINARY_NAME}-${version}-${platform}${extension}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${binary_filename}"
    local install_dir="${HOME}/.local/bin"
    local install_path="${install_dir}/${BINARY_NAME}${extension}"
    
    echo "Downloading ${binary_filename}..."
    
    # Create install directory if it doesn't exist
    mkdir -p "${install_dir}"
    
    # Download the binary
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "${install_path}" "${download_url}"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "${install_path}" "${download_url}"
    else
        echo "Error: Neither curl nor wget is available"
        exit 1
    fi
    
    # Make it executable (not needed for Windows)
    if [[ "${platform}" != windows-* ]]; then
        chmod +x "${install_path}"
    fi
    
    echo "Successfully installed ${BINARY_NAME} to ${install_path}"
    echo ""
    echo "Make sure ${install_dir} is in your PATH:"
    echo "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
    echo ""
    echo "You can also add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
}


# Main function
main() {
    local version="$1"
    
    echo "Installing ${BINARY_NAME}..."
    echo ""
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    echo "Detected platform: ${platform}"
    
    # Get version (use provided version or fetch latest)
    if [[ -n "${version}" ]]; then
        echo "Using specified version: ${version}"
    else
        version=$(get_latest_version)
        if [[ -z "${version}" ]]; then
            echo "Error: Could not determine latest version"
            exit 1
        fi
        echo "Latest version: ${version}"
    fi
    echo ""
    
    # Install binary
    install_binary "${platform}" "${version}"
}

# Show usage information
usage() {
    echo "Usage: $0 [VERSION]"
    echo ""
    echo "Install ${BINARY_NAME} binary for the current platform."
    echo ""
    echo "Arguments:"
    echo "  VERSION    Optional. Specific version to install (e.g., v0.0.2)"
    echo "             If not provided, the latest version will be downloaded."
    echo ""
    echo "Examples:"
    echo "  $0           # Install latest version"
    echo "  $0 v0.0.2    # Install specific version v0.0.2"
}

# Handle command line arguments
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    usage
    exit 0
fi

# Run main function with provided arguments
main "$1"

