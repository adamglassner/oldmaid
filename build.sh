#!/usr/bin/env bash
# Builds OldMaid.exe and the standalone zip. Version comes from VERSION;
# add a matching entry to web/CHANGELOG.md for every release.
set -euo pipefail
cd "$(dirname "$0")"
VERSION=$(tr -d ' \r\n' < VERSION)
grep -q "^## \[$VERSION\]" web/CHANGELOG.md || { echo "web/CHANGELOG.md has no entry for $VERSION"; exit 1; }
grep -q "const VERSION = '$VERSION'" web/game.js || { echo "web/game.js VERSION does not match $VERSION"; exit 1; }
OUT=$PWD/dist; STAGE=$OUT/OldMaid
rm -rf "$OUT" app/web && mkdir -p "$STAGE"
cp -r web app/web && rm -f app/web/preview.html
( cd tools/mkres && GOFLAGS=-mod=vendor go build -o "$OUT/mkres" . )
"$OUT/mkres" -version "$VERSION" -icon assets/icon.png -name "Old Maid" -desc "Old Maid - A Victorian Parlor Game" \
  -exe OldMaid.exe -internal OldMaid -identity Ironite.OldMaid -out app/rsrc_windows_amd64.syso
( cd app && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w -X main.version=$VERSION" -o "$STAGE/OldMaid.exe" . )
for f in README.txt CREDITS.txt; do sed "s/@VERSION@/$VERSION/g; s/\$/\r/" "$f" > "$STAGE/$f"; done
sed 's/$/\r/' web/CHANGELOG.md > "$STAGE/CHANGELOG.txt"
cp web/fonts/OFL.txt "$STAGE/OFL.txt"
( cd "$OUT" && zip -qr "OldMaid-$VERSION.zip" OldMaid )
ls -la "$STAGE" "$OUT"/*.zip
