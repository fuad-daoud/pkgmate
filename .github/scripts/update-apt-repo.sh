#!/bin/bash
set -e

# Script to update APT repository on GitHub Pages
# This script should be run after GoReleaser creates .deb packages

REPO_DIR="apt-repo"
DIST="stable"
COMPONENT="main"
ARCH="amd64 arm64"

echo "Setting up APT repository structure..."

# Create directory structure
mkdir -p "${REPO_DIR}/pool/main/p/pkgmate"
mkdir -p "${REPO_DIR}/dists/${DIST}/${COMPONENT}/binary-amd64"
mkdir -p "${REPO_DIR}/dists/${DIST}/${COMPONENT}/binary-arm64"

# Copy .deb files from dist directory to pool
for arch in $ARCH; do
    echo "  Copying ${arch} packages..."
    cp dist/*_linux_${arch}.deb "${REPO_DIR}/pool/main/p/pkgmate/" 2>/dev/null || true
done

# Generate Packages files for each architecture
echo "Generating Packages files..."
for arch in $ARCH; do
    echo "  Processing architecture: $arch"
    cd "${REPO_DIR}"
    dpkg-scanpackages --arch ${arch} pool/main/p/pkgmate > "dists/${DIST}/${COMPONENT}/binary-${arch}/Packages"
    gzip -9c "dists/${DIST}/${COMPONENT}/binary-${arch}/Packages" > "dists/${DIST}/${COMPONENT}/binary-${arch}/Packages.gz"
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

# Generate checksums manually
{
    echo "MD5Sum:"
    find . -type f -name "Packages*" -exec md5sum {} \; | sed 's/\.\///'
    echo "SHA256:"
    find . -type f -name "Packages*" -exec sha256sum {} \; | sed 's/\.\///'
} >> Release

cd -

# Sign the Release file with GPG
echo "Signing Release file..."
if [ -n "$GPG_PRIVATE_KEY" ]; then
    echo "$GPG_PRIVATE_KEY" | gpg --batch --import
    gpg --batch --yes --armor --detach-sign --output "${REPO_DIR}/dists/${DIST}/Release.gpg" "${REPO_DIR}/dists/${DIST}/Release"
    gpg --batch --yes --armor --detach-sign --clearsign --output "${REPO_DIR}/dists/${DIST}/InRelease" "${REPO_DIR}/dists/${DIST}/Release"
else
    echo "Warning: GPG_PRIVATE_KEY not set, skipping signing"
fi

# Export public key
echo "Exporting public key..."
if [ -n "$GPG_PRIVATE_KEY" ]; then
    gpg --armor --export > "${REPO_DIR}/public.key"
fi

echo "APT repository updated successfully!"
echo "Repository location: ${REPO_DIR}"
