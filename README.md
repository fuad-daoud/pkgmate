# PKGMATE your new terminal mate for better tux (Terminal User Experience)


## Verifying Downloads

All releases are GPG signed. Verify authenticity:

Replace `PKGMATE_VERSION` with versions v0.9.0 and higher

```bash
export PKGMATE_VERSION="v0.9.0"
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$PKGMATE_VERSION/pkgmate-arch-linux-amd64.tar.gz
curl -LO https://github.com/fuad-daoud/pkgmate/releases/download/$PKGMATE_VERSION/pkgmate-arch-linux-amd64.tar.gz.sig

# Import public key
curl https://raw.githubusercontent.com/fuad-daoud/pkgmate/main/public-key.asc | gpg --import

# Verify
gpg --verify pkgmate-arch-linux-amd64.tar.gz.sig pkgmate-arch-linux-amd64.tar.gz
```
