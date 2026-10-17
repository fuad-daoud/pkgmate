# pkgmate

[![Build](https://github.com/fuad-daoud/pkgmate/workflows/Build/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/build.yaml)
[![Release](https://github.com/fuad-daoud/pkgmate/workflows/Release/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/release.yaml)
[![CodeQL](https://github.com/fuad-daoud/pkgmate/workflows/CodeQL%20Security%20Scan/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/codeql.yaml)
[![Security Scan](https://github.com/fuad-daoud/pkgmate/workflows/Security%20Monitoring/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/security-scan.yml)
[![Publish APT](https://github.com/fuad-daoud/pkgmate/workflows/Publish%20APT%20Repository/badge.svg)](https://github.com/fuad-daoud/pkgmate/actions/workflows/publish-apt-repo.yml)

![Demo](demo.gif)

A fast, cross-platform Terminal User Interface (TUI) for browsing the packages installed through multiple package managers, in one unified, responsive interface.

## What it does / does not do yet

pkgmate is a viewer today. It does:

- List installed packages from every supported source that is present on the machine, in four tabs: direct, dependency, all and orphan packages.
- Mark packages that are frozen (held or pinned) and packages with a newer version available. For pacman, the newer version of AUR packages is looked up from the AUR.
- Search the list with live fuzzy matching.
- Refresh package databases ("update") for pacman, apt and Homebrew, running the privileged commands as described in [Privileged commands](#privileged-commands). The update flow is implemented in the backends but is not yet listed in the command palette.

pkgmate does not yet install, remove or upgrade packages. Flatpak, Snap, npm, pip and Go have no update support either: they are skipped when an update runs.

## Features

- ⚡ **Fast startup** - Direct file parsing and efficient CLI wrapping
- 🔍 **Telescope-style search** - Live fuzzy search through packages
- 📦 **Multi-backend support** - pacman/AUR, dpkg/apt, Homebrew, Flatpak, Snap, npm, pip/pipx, Go
- 🔒 **Security-first** - GPG signing, automated vulnerability scanning
- 📊 **Clean interface** - Table view with virtual scrolling, visual indicators for available updates and frozen packages, plus an orphan tab
- ⚡ **Privilege escalation** - pkexec (PolicyKit) with sudo fallback
- 🎨 **Command palette** - Quick access to all features (Ctrl+k or Ctrl+p)

## Supported Sources

A source is used when it is detected on the machine at startup. Sources that are not detected are left out silently.

| Source | Lists | Detected when | Build tag |
|--------|-------|---------------|-----------|
| pacman/AUR | Arch Linux packages, AUR packages marked `pacman/AUR` | `/var/lib/pacman` exists and `pacman` is on `PATH` | `pacman` |
| dpkg/apt | Debian/Ubuntu packages | `/var/lib/dpkg/status` exists and `apt` or `apt-get` is on `PATH` | none, always built |
| Homebrew | Formulae and casks, on macOS or Linux | `brew` is on `PATH` | `brew` |
| Flatpak | Linux Flatpak applications | `flatpak` is on `PATH` | `flatpak` |
| Snap | Linux Snap packages | `snap` is on `PATH` | none, always built |
| npm | Global npm packages | `npm` is on `PATH` | `npm` |
| pip | pip and pipx packages in site-packages | `pip3` or `pip` is on `PATH` and can list packages | `pip` |
| Go | Binaries installed with `go install`, read from `$GOBIN`, `$GOPATH/bin` or `~/go/bin` | `go` is on `PATH` | `golang` |

Release binaries are built with all of these. See [From Source](#from-source) to build a smaller binary.

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

# Build with every backend, as release builds do
go build -tags=all_backends -o pkgmate ./main

sudo install -m 755 pkgmate /usr/local/bin/
```

**Build tags:** which backends are compiled in is decided at build time.

- `all_backends` - every backend below. Release builds use `-tags=all_backends` with `CGO_ENABLED=0`.
- `pacman`, `brew`, `flatpak`, `npm`, `pip`, `golang` - one tag per backend, combinable, e.g. `-tags=pacman,npm`.
- No tag - only the dpkg/apt and Snap backends, which have no tag of their own and are always built.
- `dummy` - registers a placeholder backend for UI development. It does not compile against the current backend interface, so it is not usable at the moment.

```bash
# Only the backends you use
go build -tags=pacman,pip -o pkgmate ./main
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

## Privileged commands

Refreshing a package database needs root, so pkgmate itself refuses to start as root and escalates only for the command that needs it. For each privileged command it:

1. Resolves the command to an absolute path.
2. Runs it through `pkexec <path> <args>`. If `pkexec` is not installed, cannot be started, or exits with 126 or 127 (authentication dismissed or not authorized), it retries with `sudo -- <path> <args>`.
3. Runs the argv directly, without a shell, so no argument is ever interpreted by one.

The exact argv is written to the application log (Debug Console). The command's output is shown live in the terminal while it runs, and kept so pkgmate can show it again afterwards.

The commands pkgmate runs today:

| Source | Command | Escalation |
|--------|---------|------------|
| pacman | `pacman -Syy --noconfirm` | pkexec, then sudo |
| dpkg/apt | `apt-get update`, or `apt update` when `apt-get` is missing | pkexec, then sudo |
| Homebrew | `brew update` | none, runs as your user |

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

# Record demo (writes demo.gif)
vhs demo.tape
```

## Development

Run the tests with every backend compiled in:

```bash
go test -tags=all_backends ./...
```

Try a snapshot build in a container, with the binary mounted at `/pkgmate`:

```bash
make arch    # Arch Linux container
make debian  # Debian container
make brew    # Homebrew container

# Build snapshot binaries with goreleaser only
make build
```

## Security

All releases undergo automated security scanning with CodeQL, gosec, and govulncheck. View scan results in [Security Monitoring](https://github.com/fuad-daoud/pkgmate/actions/workflows/security-scan.yml).

Report security issues via [GitHub Security Advisories](https://github.com/fuad-daoud/pkgmate/security/advisories).

## License

MIT License - see [LICENSE](LICENSE) for details.
