# SkyOfDie

[Русский](README.ru.md)

SkyOfDie is a native Go recreation of the DOS winter sports game **Ski or Die**. Play Snowball Blast, Downhill Blitz, Acro Aerials, Innertube Thrash, and Snowboard Halfpipe in Practice or compete with up to six players taking turns. Innertube Thrash supports two simultaneous riders.

Gameplay runs in Go without a runtime emulator. Original artwork, maps, music, and effects are included as PNG, JSON, Ogg Vorbis, and WAV resources. Supports Windows, Linux, and macOS on amd64 and arm64.

| Snowball Blast | Snowboard Halfpipe |
| --- | --- |
| ![Snowball Blast](screenshots/snowball.png) | ![Snowboard Halfpipe](screenshots/halfpipe.png) |

## Controls

In Rodney's shop, choose **SIGN IN** to enter names and select each player's device, then **GO COMPETE**. Choose **GO PRACTICE** for practice. On the resort map, steer into an event entrance; **COMPETE ALL** starts all five events in competition mode.

Use arrows and Enter/Space in menus. In sign-in, confirm a setting, change it with left/right, and confirm again. Type a name and confirm; select **DONE** to return. Names have a ten-character limit.

### Keyboard and gamepad

| Action | Keypad One | Keypad Two | Additional keys for the current player |
| --- | --- | --- | --- |
| Up-left / up / up-right | Numpad 7 / 8 / 9 | R / T / Y | Top-row 7 / 8 / 9 |
| Left / right | Numpad 4 / 6 | F / H | Top-row 4 / 6 |
| Down-left / down / down-right | Numpad 1 / 2 / 3 | V / B / N | Top-row 1 / 2 / 3 |
| Button one / confirm | Enter | A | Space |
| Button two | Right Shift | Z | Left Shift |

Keypad One also accepts arrow keys for cardinal directions. **WASD** and **IJKL** provide additional directions; hold adjacent keys for diagonals. Top-row **5** is neutral. Letter bindings use physical key positions. Space also confirms name entry.

WASD is disabled during an event if the current player or a simultaneous second rider uses Keypad Two, preserving its A button. IJKL, top-row digits, Space, Left Shift, and the mouse remain available for the current player. The second rider uses their selected device. **JOYSTICK A/B** use connected gamepads' axes and first two buttons.

| Location/event | Actions |
| --- | --- |
| Resort | Left/right turn; hold button one to move. |
| Snowball Blast | Directions aim; button one throws. Button two + a cardinal direction changes the view. |
| Downhill Blitz | Down accelerates, up brakes, left/right steer. Button one + direction selects a stunt when terrain permits. |
| Acro Aerials | Alternate left/right on the approach. In flight, button one + direction selects a stunt; release the direction to leave a held pose. |
| Innertube Thrash | Directions move, accelerate, or brake. Button one + left/right rotates; button two attacks. |
| Snowboard Halfpipe | Left/right carve. Button one + direction selects an available floor, bump, or lip stunt; release the direction to leave a held pose. |

### Mouse

- **Menus:** point and left-click. In sign-in, click a row; click an active setting again or scroll to change it. Click a name to type it, then confirm with Enter/Space or a click.
- **Resort:** hold the left button to turn and move toward the pointer; release to stop. Guide the skier around buildings.
- **Snowballs:** move to aim; hold the left button to throw. Right-click cycles the view clockwise; wheel up/down selects the previous/next view.
- **Other events:** the pointer's position relative to the window center selects one of eight directions; the center is neutral. Left/right buttons act as buttons one/two.

Keyboard/gamepad directions take priority over mouse steering. Move the mouse again to resume it. Middle-click or **P** pauses. Right-click in menus acts as Escape; click **YES/NO** on the replay screen.

**Esc** opens event standings during competition, the replay question during practice, and the quit query in the shop/resort. **Ctrl+M** toggles sound, **F11** toggles fullscreen, and **F12** saves a screenshot. Settings, records, and captured screenshots are stored in the OS user configuration directory under `SkyOfDie`.

## Build and run

Install **Go 1.25 or later**. From the project root containing `main.go`:

```sh
go mod download
go run .
```

The first build requires internet access for Go dependencies. No C compiler or DOS installation is required. Linux needs a desktop graphics session (X11 or XWayland).

Build for the current system:

```sh
go run ./cmd/build
```

Build all six targets, or choose one:

```sh
go run ./cmd/build -all
go run ./cmd/build -os windows -arch arm64
```

| System | amd64 output | arm64 output |
| --- | --- | --- |
| Windows | `build/windows-amd64/SkyOfDie.exe` | `build/windows-arm64/SkyOfDie.exe` |
| Linux | `build/linux-amd64/SkyOfDie` | `build/linux-arm64/SkyOfDie` |
| macOS | `build/darwin-amd64/SkyOfDie.app` | `build/darwin-arm64/SkyOfDie.app` |

The builder embeds the Windows icon and packages the macOS app with its icon. Icon files are in `assets/icon`.

PC Speaker audio is the default. Run `go run . -adlib` for the AdLib arrangement. Other options include `-nosound`, `-fullscreen`, and `-scale 3`.

## Licenses

New Go code and the application icon are provided under the [MIT license](LICENSE). Original Ski or Die artwork, audio, font, maps, animation data, and screenshots retain their original rights and are **not** covered by that license.
