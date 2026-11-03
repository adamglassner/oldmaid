# Old Maid: A Victorian Parlor Game

The classic matching card game, dressed for a Victorian parlor. An Ironite game for Windows 10 and 11,
written in Go with [Ebitengine](https://ebitengine.org).

- **Single player:** play against one to three computer guests, drawn fresh at every deal, each with a
  portrait, a name to match and their own remarks.
- **Pass & Play:** 2 to 4 people share one computer. Hands stay hidden until it's your turn.
- **Play Online:** one player hosts a table and friends join with the host's address (port 47820).

## Play

The easiest way is **[Ironite Launcher](https://github.com/adamglassner/launcher)**: it downloads
Old Maid from this repository's releases, installs it, and updates it automatically when a
new version is released.

Or grab `OldMaid-<version>-windows.zip` from the [latest release](../../releases/latest), unzip it
anywhere and run `OldMaid.exe`.

Your record and settings are saved in `%APPDATA%\Ironite\Old Maid\save.json`, outside the
install folder, so updating never wipes them.

## Build

`bash build.sh` (needs Go 1.24+ and bash, e.g. Git Bash) builds `dist/OldMaid/OldMaid.exe` and a zip.
To run it straight from source on Windows: `go run .`

| File | What it is |
|---|---|
| `main.go` | Window, saving, and the hooks used by automated checks |
| `game.go` | The rules and the flow of a hand |
| `view.go`, `table.go` | Laying out and drawing the table from your seat |
| `ui.go`, `panels.go` | The top bar, title card, rules, credits and multiplayer screens |
| `faces.go` | The guests' cameo portraits, drawn in code, and their random names |
| `net.go` | Online play (host and join) |
| `sound.go`, `synth.go` | Music and sound effects, synthesized in Go |
| `assets/` | Card art, the doily, fonts, icon and promo art |
| `tools/mkres/` | Writes the Windows icon and version info into the .exe |

## Releasing

Bump `VERSION`, add a section at the top of `CHANGELOG.md`, and push to `main`. The
[Release workflow](.github/workflows/release.yml) builds the game and publishes `v<version>` with
the zip that Ironite Launcher installs.

(c) 2026 Ironite.
