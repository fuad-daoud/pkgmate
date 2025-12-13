#!/bin/bash
set -e

# Script to update APT repository on GitHub Pages
# This script should be run after GoReleaser creates .deb packages

REPO_DIR="apt-repo"
DIST="stable"
COMPONENT="main"
ARCH="amd64 arm64"

echo "=== Pre-flight checks ==="
echo "Current directory: $(pwd)"
echo "Contents:"
ls -la

echo ""
echo "Checking dist directory..."
if [ ! -d "dist" ]; then
    echo "Error: dist directory not found"
    exit 1
fi

echo "Contents of dist/:"
ls -lh dist/

DEB_COUNT=$(find dist -name "*.deb" | wc -l)
if [ "$DEB_COUNT" -eq 0 ]; then
    echo "Error: No .deb files found in dist/"
    exit 1
fi

echo "Found $DEB_COUNT .deb file(s)"
echo ""

echo "=== Setting up APT repository structure ==="

# Create directory structure
mkdir -p "${REPO_DIR}/pool/main/p/pkgmate"
mkdir -p "${REPO_DIR}/dists/${DIST}/${COMPONENT}/binary-amd64"
mkdir -p "${REPO_DIR}/dists/${DIST}/${COMPONENT}/binary-arm64"

# Copy .deb files from dist directory to pool
echo "Copying .deb packages to pool..."
for arch in $ARCH; do
    echo "  Copying ${arch} packages..."
    cp dist/*_linux_${arch}.deb "${REPO_DIR}/pool/main/p/pkgmate/" 2>/dev/null || true
done

# Generate Packages files for each architecture
echo "Generating Packages files..."
for arch in $ARCH; do
    echo "  Processing architecture: $arch"
    cd "${REPO_DIR}"
    dpkg-scanpackages --multiversion pool/ > "dists/${DIST}/${COMPONENT}/binary-${arch}/Packages"
    gzip -9nc "dists/${DIST}/${COMPONENT}/binary-${arch}/Packages" > "dists/${DIST}/${COMPONENT}/binary-${arch}/Packages.gz"
    cd -
done

# Generate Release file
echo "Generating Release file..."
cd "${REPO_DIR}/dists/${DIST}"

cat > Release <<EOF
Origin: pkgmate
Label: pkgmate
Suite: ${DIST}
Codename: ${DIST}
Architectures: amd64 arm64
Components: ${COMPONENT}
Description: pkgmate - TUI application to manage your dependencies
Date: $(date -Ru)
EOF

# Generate checksums for Packages files
apt-ftparchive release . >> Release

cd -

# Sign the Release file with GPG
echo "Signing Release file..."
if [ -n "$GPG_PRIVATE_KEY" ]; then
    echo "$GPG_PRIVATE_KEY" | gpg --batch --import

    # Get the key ID
    KEY_ID=$(gpg --list-secret-keys --keyid-format=long | grep sec | awk '{print $2}' | cut -d'/' -f2)

    # Sign with passphrase
    echo "$GPG_PASSPHRASE" | gpg --batch --yes --passphrase-fd 0 --pinentry-mode loopback \
        --armor --detach-sign --default-key "$KEY_ID" \
        --output "${REPO_DIR}/dists/${DIST}/Release.gpg" "${REPO_DIR}/dists/${DIST}/Release"

    echo "$GPG_PASSPHRASE" | gpg --batch --yes --passphrase-fd 0 --pinentry-mode loopback \
        --armor --clearsign --default-key "$KEY_ID" \
        --output "${REPO_DIR}/dists/${DIST}/InRelease" "${REPO_DIR}/dists/${DIST}/Release"
 
    # Export public key
    gpg --armor --export "$KEY_ID" > "${REPO_DIR}/public.key"
else
    echo "Warning: GPG_PRIVATE_KEY not set, skipping signing"
fi

echo "APT repository updated successfully!"
echo "Repository location: ${REPO_DIR}"

