#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 VERSION ARTIFACT_ARCH OUTPUT" >&2
  echo "ARTIFACT_ARCH must be amd, amd64, or arm64" >&2
  exit 2
fi

version=$1
artifact_arch=$2
output=$3

case "$artifact_arch" in
  amd)
    deb_arch=i386
    go_arch=386
    ;;
  amd64)
    deb_arch=amd64
    go_arch=amd64
    ;;
  arm64)
    deb_arch=arm64
    go_arch=arm64
    ;;
  *)
    echo "unsupported artifact architecture: $artifact_arch" >&2
    exit 2
    ;;
esac

if [[ ! -d bin/$go_arch ]]; then
  echo "missing build directory: bin/$go_arch" >&2
  exit 1
fi

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

mkdir -p \
  "$stage/DEBIAN" \
  "$stage/usr/bin" \
  "$stage/usr/share/amaquet" \
  "$stage/usr/share/doc/amaquet"

for binary in amaquet amaquet-cli amaquet-keygen amaquet-restore; do
  install -m 0755 "bin/$go_arch/$binary" "$stage/usr/bin/$binary"
done

install -m 0644 amaquet.example.json "$stage/usr/share/amaquet/amaquet.example.json"
install -m 0644 README.md "$stage/usr/share/doc/amaquet/README.md"
install -m 0644 LICENSE "$stage/usr/share/doc/amaquet/LICENSE"

cat > "$stage/DEBIAN/control" <<EOF
Package: amaquet
Version: $version
Section: database
Priority: optional
Architecture: $deb_arch
Maintainer: Newfoundcodes <jonathaneldy.baldivicio@newfoundcodes.com>
Description: Amaquet typed in-memory database
 Amaquet is a standalone in-memory database with a custom binary protocol,
 typed values, optional append-only persistence, and an HTTP administration API.
EOF

mkdir -p "$(dirname "$output")"
dpkg-deb --build --root-owner-group "$stage" "$output"
