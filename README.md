# Old Maid: A Victorian Parlor Game

The classic matching card game, dressed for a Victorian parlor.

An Ironite game for Windows 10 and 11.

## Play

The easiest way is **[Ironite Launcher](https://github.com/adamglassner/launcher)**: it downloads
Old Maid from this repository's releases, installs it, and updates it automatically when a
new version is released.

Or grab `OldMaid-<version>-windows.zip` from the [latest release](../../releases/latest), unzip it
anywhere and run `OldMaid.exe`.

## Build

`bash build.sh` (needs Go 1.22+) builds `dist/` with the Windows executable.

## Releasing

Bump the version, add a section at the top of the changelog, and push to `main`. The
[Release workflow](.github/workflows/release.yml) builds the game and publishes `v<version>` with
the zip that Ironite Launcher installs.

(c) 2026 Ironite.
