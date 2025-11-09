# pkgmate

[![Build](https://github.com/fuad-daoud/pkgmate/workflows/Build/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/build.yaml)
[![Release](https://github.com/fuad-daoud/pkgmate/workflows/Release/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/release.yaml)
[![CodeQL](https://github.com/fuad-daoud/pkgmate/workflows/CodeQL%20Security%20Scan/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/codeql.yaml)
[![Security Scan](https://github.com/fuad-daoud/pkgmate/workflows/Security%20Monitoring/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/security-scan.yml)
[![Publish APT](https://github.com/fuad-daoud/pkgmate/workflows/Publish%20APT%20Repository/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/publish-apt-repo.yml)


A fast, cross-platform Terminal User Interface (TUI) package manager for Linux and macOS. Manage packages from **Arch Linux** (pacman), **Debian/Ubuntu** (dpkg/apt), and **macOS** (Homebrew) with a unified, responsive interface.

## Features

- 🚀 **Blazing fast** - Direct library integration, instant startup
- 🔍 **Telescope-style search** - Fuzzy search through packages
- 📦 **Multi-platform** - Arch, Debian/Ubuntu, and macOS support
- 🔒 **Security-first** - GPG signing, automated vulnerability scanning
- 📊 **Clean interface** - Table view with virtual scrolling, visual indicators for updates and frozen packages
- ⚡ **Privilege escalation** - PolicyKit integration with sudo fallback

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
export VERSION="v0.9.0"
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$VERSION/pkgmate-arch-linux-amd64.tar.gz
tar -xzf pkgmate-arch-linux-amd64.tar.gz
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
export VERSION="v0.9.0"
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$VERSION/pkgmate_${VERSION#v}_linux_amd64.deb
sudo dpkg -i pkgmate_${VERSION#v}_linux_amd64.deb
```

### macOS

**Via Homebrew:**
```bash
brew tap fuad-daoud/pkgmate
brew install pkgmate
```

**Manual:**
```bash
export VERSION="v0.9.0"
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$VERSION/pkgmate-brew-darwin-amd64.tar.gz
tar -xzf pkgmate-brew-darwin-amd64.tar.gz
sudo install -m 755 pkgmate /usr/local/bin/
```

### From Source

**Requirements:** Go 1.25+, base-devel (Arch), build-essential (Debian)

```bash
git clone https://github.com/fuad-daoud/pkgmate.git
cd pkgmate

# For Arch Linux
CGO_ENABLED=1 go build -tags=arch -o pkgmate ./main

# For Debian/Ubuntu
CGO_ENABLED=0 go build -tags=dpkg -o pkgmate ./main

# For macOS/Homebrew
CGO_ENABLED=0 go build -tags=brew -o pkgmate ./main

sudo install -m 755 pkgmate /usr/local/bin/
```

## Usage

```bash
# Launch pkgmate
pkgmate

# Show version
pkgmate --version
```

## Verifying Downloads

All releases are GPG signed. Verify authenticity:

```bash
export PKGMATE_VERSION="v0.9.0"
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$PKGMATE_VERSION/pkgmate-arch-linux-amd64.tar.gz
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$PKGMATE_VERSION/pkgmate-arch-linux-amd64.tar.gz.sig

# Import public key
curl https://raw.githubusercontent.com/fuad-daoud/pkgmate/main/public-key.asc | gpg --import

# Verify signature
gpg --verify pkgmate-arch-linux-amd64.tar.gz.sig pkgmate-arch-linux-amd64.tar.gz
```

## Development

```bash
# Run with hot reload
make arch    # Test in Arch container
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
