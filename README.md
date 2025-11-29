# pkgmate

[![Build](https://github.com/fuad-daoud/pkgmate/workflows/Build/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/build.yaml)
[![Release](https://github.com/fuad-daoud/pkgmate/workflows/Release/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/release.yaml)
[![CodeQL](https://github.com/fuad-daoud/pkgmate/workflows/CodeQL%20Security%20Scan/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/codeql.yaml)
[![Security Scan](https://github.com/fuad-daoud/pkgmate/workflows/Security%20Monitoring/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/security-scan.yml)
[![Publish APT](https://github.com/fuad-daoud/pkgmate/workflows/Publish%20APT%20Repository/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/publish-apt-repo.yml)

![Demo](demo.gif)

A fast, cross-platform Terminal User Interface (TUI) package manager. Manage packages across multiple backends with a unified, responsive interface.

## Features

- ⚡ **Fast startup** - Direct file parsing and efficient CLI wrapping
- 🔍 **Telescope-style search** - Live fuzzy search through packages
- 📦 **Multi-backend support** - pacman/AUR, dpkg/apt, Homebrew, Flatpak, Snap, npm
- 🔒 **Security-first** - GPG signing, automated vulnerability scanning
- 📊 **Clean interface** - Table view with virtual scrolling, visual indicators for updates and frozen packages
- ⚡ **Privilege escalation** - PolicyKit integration with sudo fallback
- 🎨 **Command palette** - Quick access to all features (Ctrl+k/Ctrl+k)

## Supported Platforms

| Platform | Backend |
|----------|---------|
| Arch Linux | pacman/AUR |
| Debian/Ubuntu | dpkg/apt |
| macOS | Homebrew (formulae + casks) |
| Linux | Flatpak |
| Linux | Snap |
| All platforms | npm (global) |

## Installation

### Arch Linux

**Via AUR:**
```bash
yay -S pkgmate-bin
# or
paru -S pkgmate-bin
```

**Manual:**
```bash
curl -LO https://github.com/fuad-daoud/pkgmate/releases/latest/download/pkgmate-linux-amd64.tar.gz
tar -xzf pkgmate-linux-amd64.tar.gz
sudo install -m 755 pkgmate /usr/local/bin/
```

### Debian/Ubuntu

**Via APT repository:**
```bash
# Add repository
curl -fsSL https://fuad-daoud.github.io/pkgmate/public-key.asc | sudo gpg --dearmor -o /usr/share/keyrings/pkgmate.gpg
echo "deb [signed-by=/usr/share/keyrings/pkgmate.gpg] https://fuad-daoud.github.io/pkgmate/ stable main" | sudo tee /etc/apt/sources.list.d/pkgmate.list

# Install
sudo apt update
sudo apt install pkgmate
```

**Manual .deb:**
```bash
curl -LO https://github.com/fuad-daoud/pkgmate/releases/latest/download/pkgmate_linux_amd64.deb
sudo dpkg -i pkgmate_*_linux_amd64.deb
```

### macOS

**Via Homebrew:**
```bash
brew tap fuad-daoud/pkgmate
brew install pkgmate
```

**Manual:**
```bash
curl -LO https://github.com/fuad-daoud/pkgmate/releases/latest/download/pkgmate-darwin-universal.tar.gz
tar -xzf pkgmate-darwin-universal.tar.gz
sudo install -m 755 pkgmate /usr/local/bin/
```

### From Source

**Requirements:** Go 1.25+, build tools (base-devel on Arch, build-essential on Debian)

```bash
git clone https://github.com/fuad-daoud/pkgmate.git
cd pkgmate

# Build (detects platform automatically)
go build -o pkgmate ./main

sudo install -m 755 pkgmate /usr/local/bin/
```

## Usage

```bash
# Launch pkgmate
pkgmate

# Show version
pkgmate --version
```

**Keyboard shortcuts:**
- `/` or `Ctrl+F` - Search packages
- `Tab` / `Shift+Tab` - Switch between tabs
- `↑↓` or `j/k` - Navigate packages
- `Ctrl+P` or `Ctrl+k` - Command palette
- `Ctrl+C` - Quit

## Verifying Downloads

All releases are GPG signed:

```bash
# Download release and signature
curl -LO https://github.com/fuad-daoud/pkgmate/releases/latest/download/pkgmate-linux-amd64.tar.gz
curl -LO https://github.com/fuad-daoud/pkgmate/releases/latest/download/pkgmate-linux-amd64.tar.gz.sig

# Import public key
curl https://raw.githubusercontent.com/fuad-daoud/pkgmate/main/public-key.asc | gpg --import

# Verify signature
gpg --verify pkgmate-linux-amd64.tar.gz.sig pkgmate-linux-amd64.tar.gz
```

## macOS Code Signature Verification

macOS binaries are signed with Apple Developer ID and notarized:

```bash
curl -LO https://github.com/fuad-daoud/pkgmate/releases/latest/download/pkgmate-darwin-universal.tar.gz
tar -xzf pkgmate-darwin-universal.tar.gz

# Verify code signature
codesign --verify --verbose pkgmate
# Check notarization and Gatekeeper approval
spctl -a -vv -t install pkgmate
# View signing details
codesign -dvvv pkgmate
```

**Expected output:**
```
pkgmate: valid on disk
pkgmate: satisfies its Designated Requirement
pkgmate: accepted
source=Notarized Developer ID
```

## Recording Demo

The demo GIF is created using [VHS](https://github.com/charmbracelet/vhs):

```bash
# Install VHS
go install github.com/charmbracelet/vhs@latest

# Record demo
make demo

# Or record the full demo
make demo-full
```

## Development

```bash
# Run with hot reload in containers
make arch    # Test in Arch Linux container
make debian  # Test in Debian container
make brew    # Test in Homebrew container

# Build all variants
make build
```

## Security

All releases undergo automated security scanning with CodeQL, gosec, and govulncheck. View scan results in [Security Monitoring](https://github.com/fuad-daoud/pkgmate/actions/workflows/security-scan.yml).

Report security issues via [GitHub Security Advisories](https://github.com/fuad-daoud/pkgmate/security/advisories).

## License

MIT License - see [LICENSE](LICENSE) for details.
