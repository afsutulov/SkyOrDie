package main

import (
	"strings"

	"skyordie/internal/controls"
	"skyordie/internal/game"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (a *application) input() game.Input {
	// Original keypad-one scan order. Cardinal keys do not synthesize a
	// diagonal; the dedicated keypad corners have priority as in 0x6D0C.
	in := game.Input{AnyKey: len(inpututil.AppendJustPressedKeys(nil)) > 0,
		Action: down(ebiten.KeyEnter, ebiten.KeyNumpadEnter), ActionPressed: pressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter),
		Trick: down(ebiten.KeyShiftRight), TrickPressed: pressed(ebiten.KeyShiftRight),
		Confirm: pressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter), Escape: pressed(ebiten.KeyEscape), Pause: pressed(ebiten.KeyP),
		UpPressed: pressed(ebiten.KeyArrowUp, ebiten.KeyNumpad8), DownPressed: pressed(ebiten.KeyArrowDown, ebiten.KeyNumpad2),
		LeftPressed: pressed(ebiten.KeyArrowLeft, ebiten.KeyNumpad4), RightPressed: pressed(ebiten.KeyArrowRight, ebiten.KeyNumpad6),
		Tab: pressed(ebiten.KeyTab), Backspace: pressed(ebiten.KeyBackspace)}
	setDevice(&in, 0)

	device := a.model.Save.Controls[0]
	if a.model.Phase == game.Resort && device != 0 {
		// Original 0x240 accepts the selected device and keypad one together;
		// a nonzero keypad-one direction takes priority.
		selected := game.Input{}
		setDevice(&selected, device)
		if !in.Up && !in.Down && !in.Left && !in.Right {
			in.Up, in.Down, in.Left, in.Right = selected.Up, selected.Down, selected.Left, selected.Right
		}
		in.Action = in.Action || selected.Action
		in.ActionPressed = in.ActionPressed || selected.ActionPressed
		in.Trick = in.Trick || selected.Trick
	}
	if a.model.Phase == game.Playing {
		setDevice(&in, device)
		in.Confirm = in.ActionPressed
	}
	second := game.Input{}
	setDevice(&second, a.model.Save.Controls[1])
	if in.Action {
		in.Trick = false
		in.TrickPressed = false
	}
	if second.Action {
		second.Trick = false
		second.TrickPressed = false
	}
	in.Second = &second
	in.Text = string(ebiten.AppendInputChars(nil))
	if a.model.Phase == game.Signup && a.model.Front.Editing {
		// Space is now a confirmation key, including while entering a name.
		in.Text = strings.ReplaceAll(in.Text, " ", "")
	}
	x, y := ebiten.CursorPosition()
	mouse := controls.Mouse{X: x, Y: y * 5 / 6, Inside: x >= 0 && x < 320 && y >= 0 && y < 240,
		Focused: ebiten.IsFocused(), Left: ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft), Right: ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		LeftPressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft), RightPressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight), MiddlePressed: inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle)}
	_, mouse.Wheel = ebiten.Wheel()
	return a.controls.Apply(a.model, in, modernKeyboard(a.model), mouse)
}

func modernKeyboard(g *game.Game) controls.Keyboard {
	up := []ebiten.Key{ebiten.KeyI, ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9}
	downKeys := []ebiten.Key{ebiten.KeyK, ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3}
	left := []ebiten.Key{ebiten.KeyJ, ebiten.KeyDigit1, ebiten.KeyDigit4, ebiten.KeyDigit7}
	right := []ebiten.Key{ebiten.KeyL, ebiten.KeyDigit3, ebiten.KeyDigit6, ebiten.KeyDigit9}
	if controls.WASDEnabled(g) {
		up = append(up, ebiten.KeyW)
		downKeys = append(downKeys, ebiten.KeyS)
		left = append(left, ebiten.KeyA)
		right = append(right, ebiten.KeyD)
	}
	return controls.Keyboard{DirectionUsed: down(ebiten.KeyDigit5), Up: down(up...), Down: down(downKeys...), Left: down(left...), Right: down(right...),
		UpPressed: pressed(up...), DownPressed: pressed(downKeys...), LeftPressed: pressed(left...), RightPressed: pressed(right...),
		Primary: down(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace), PrimaryPressed: pressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace),
		Secondary: down(ebiten.KeyShiftLeft), SecondaryPressed: pressed(ebiten.KeyShiftLeft)}
}

func setDevice(in *game.Input, device int) {
	if device == 0 {
		in.Up = false
		in.Down = false
		in.Left = false
		in.Right = false
		switch {
		case down(ebiten.KeyNumpad7):
			in.Up = true
			in.Left = true
		case down(ebiten.KeyArrowUp, ebiten.KeyNumpad8):
			in.Up = true
		case down(ebiten.KeyNumpad9):
			in.Up = true
			in.Right = true
		case down(ebiten.KeyArrowLeft, ebiten.KeyNumpad4):
			in.Left = true
		case down(ebiten.KeyArrowRight, ebiten.KeyNumpad6):
			in.Right = true
		case down(ebiten.KeyNumpad1):
			in.Down = true
			in.Left = true
		case down(ebiten.KeyArrowDown, ebiten.KeyNumpad2):
			in.Down = true
		case down(ebiten.KeyNumpad3):
			in.Down = true
			in.Right = true
		}
		in.Action = down(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace)
		in.ActionPressed = pressed(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace)
		in.Trick = down(ebiten.KeyShiftRight) && !in.Action
		in.TrickPressed = pressed(ebiten.KeyShiftRight) && !in.Action
		return
	}
	in.Up = false
	in.Down = false
	in.Left = false
	in.Right = false
	if device == 1 {
		switch {
		case down(ebiten.KeyR):
			in.Up = true
			in.Left = true
		case down(ebiten.KeyT):
			in.Up = true
		case down(ebiten.KeyY):
			in.Up = true
			in.Right = true
		case down(ebiten.KeyF):
			in.Left = true
		case down(ebiten.KeyH):
			in.Right = true
		case down(ebiten.KeyV):
			in.Down = true
			in.Left = true
		case down(ebiten.KeyB):
			in.Down = true
		case down(ebiten.KeyN):
			in.Down = true
			in.Right = true
		}
		in.Action = down(ebiten.KeyA)
		in.ActionPressed = pressed(ebiten.KeyA)
		in.Trick = down(ebiten.KeyZ)
		in.TrickPressed = pressed(ebiten.KeyZ)
	} else {
		in.Action = false
		in.ActionPressed = false
		in.Trick = false
		in.TrickPressed = false
		ids := ebiten.AppendGamepadIDs(nil)
		i := device - 2
		if i < 0 || i >= len(ids) {
			return
		}
		id := ids[i]
		x, y := ebiten.GamepadAxisValue(id, 0), ebiten.GamepadAxisValue(id, 1)
		in.Left = x < -0.25
		in.Right = x > .25
		in.Up = y < -0.25
		in.Down = y > .25
		in.Action = ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton0)
		in.Trick = ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton1)
		in.ActionPressed = inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton0)
		in.TrickPressed = inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton1)
	}
}
