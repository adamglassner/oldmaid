#!/usr/bin/env bash
# Builds OldMaid.exe (a native Windows game written in Go) and the standalone zip.
# The version comes from VERSION; add a matching entry to CHANGELOG.md for every release.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=$(tr -d ' \r\n' < VERSION)
grep -q "^## \[$VERSION\]" CHANGELOG.md || { echo "CHANGELOG.md has no entry for $VERSION"; exit 1; }
OUT=$PWD/dist; STAGE=$OUT/OldMaid
rm -rf "$OUT" && mkdir -p "$STAGE"
# Windows icon and version info, written into the .exe
( cd tools/mkres && GOFLAGS=-mod=vendor go build -o "$OUT/mkres" . )
"$OUT/mkres" -version "$VERSION" -icon assets/icon.png -name "Old Maid" -desc "Old Maid - A Victorian Parlor Game" \
  -exe OldMaid.exe -internal OldMaid -identity Ironite.OldMaid -out rsrc_windows_amd64.syso
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w -X main.version=$VERSION" -o "$STAGE/OldMaid.exe" .
rm -f rsrc_windows_amd64.syso "$OUT/mkres"
for f in README.txt CREDITS.txt; do sed "s/@VERSION@/$VERSION/g; s/\$/\r/" "$f" > "$STAGE/$f"; done
sed 's/$/\r/' CHANGELOG.md > "$STAGE/CHANGELOG.txt"
cp assets/fonts/OFL.txt "$STAGE/OFL.txt"
echo "$VERSION" > "$STAGE/VERSION"
( cd "$OUT" && zip -qr "OldMaid-$VERSION.zip" OldMaid )
ls -la "$STAGE" "$OUT"/*.zip
