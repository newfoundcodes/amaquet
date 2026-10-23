#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 VERSION ARCH OUTPUT" >&2
  echo "ARCH must be amd64, arm64, or universal" >&2
  exit 2
fi

version=$1
arch=$2
output=$3

case "$arch" in
  amd64|arm64|universal) ;;
  *)
    echo "unsupported macOS architecture: $arch" >&2
    exit 2
    ;;
esac

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

payload="$stage/payload"
resources="$stage/resources"
component="$stage/amaquet-component.pkg"
distribution="$stage/distribution.xml"

mkdir -p \
  "$payload/usr/local/bin" \
  "$payload/usr/local/share/amaquet" \
  "$payload/usr/local/share/doc/amaquet" \
  "$resources"

for binary in amaquet amaquet-cli amaquet-keygen amaquet-restore; do
  if [[ "$arch" == universal ]]; then
    lipo -create \
      "bin/amd64/$binary" \
      "bin/arm64/$binary" \
      -output "$payload/usr/local/bin/$binary"
    chmod 0755 "$payload/usr/local/bin/$binary"
  else
    install -m 0755 "bin/$arch/$binary" "$payload/usr/local/bin/$binary"
  fi
done

install -m 0644 amaquet.example.json "$payload/usr/local/share/amaquet/amaquet.example.json"
install -m 0644 README.md "$payload/usr/local/share/doc/amaquet/README.md"
install -m 0644 LICENSE "$payload/usr/local/share/doc/amaquet/LICENSE"
install -m 0644 assets/amaquet-logo.png "$resources/amaquet-logo.png"

pkgbuild \
  --root "$payload" \
  --identifier com.newfoundcodes.amaquet \
  --version "$version" \
  --install-location / \
  "$component"

cat > "$distribution" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
  <title>Amaquet</title>
  <options customize="never" require-scripts="false" />
  <background file="amaquet-logo.png" mime-type="image/png" scaling="proportional" alignment="center" />
  <domains enable_anywhere="false" enable_currentUserHome="false" enable_localSystem="true" />
  <choices-outline>
    <line choice="default" />
  </choices-outline>
  <choice id="default" title="Amaquet">
    <pkg-ref id="com.newfoundcodes.amaquet" />
  </choice>
  <pkg-ref id="com.newfoundcodes.amaquet" version="$version">amaquet-component.pkg</pkg-ref>
</installer-gui-script>
EOF

mkdir -p "$(dirname "$output")"
productbuild \
  --distribution "$distribution" \
  --package-path "$stage" \
  --resources "$resources" \
  --identifier com.newfoundcodes.amaquet \
  --version "$version" \
  "$output"
