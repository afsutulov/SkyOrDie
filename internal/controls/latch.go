package controls

import "skyordie/internal/game"

// Latch retains one-shot events between desktop updates and original PIT steps.
func Latch(in *game.Input, old game.Input) {
	if !in.Pointer && old.Pointer {
		in.Pointer, in.PointerX, in.PointerY = true, old.PointerX, old.PointerY
	}
	in.ViewDelta += old.ViewDelta
	in.AnyKey = in.AnyKey || old.AnyKey
	in.ActionPressed = in.ActionPressed || old.ActionPressed
	in.TrickPressed = in.TrickPressed || old.TrickPressed
	in.Confirm = in.Confirm || old.Confirm
	in.Escape = in.Escape || old.Escape
	in.Pause = in.Pause || old.Pause
	in.LeftPressed = in.LeftPressed || old.LeftPressed
	in.RightPressed = in.RightPressed || old.RightPressed
	in.UpPressed = in.UpPressed || old.UpPressed
	in.DownPressed = in.DownPressed || old.DownPressed
	in.Tab = in.Tab || old.Tab
	in.Backspace = in.Backspace || old.Backspace
	in.Text = old.Text + in.Text
	if in.Second != nil && old.Second != nil {
		Latch(in.Second, *old.Second)
	}
}
