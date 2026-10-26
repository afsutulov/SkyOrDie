// Package controls maps optional desktop controls to the original game inputs.
// It contains no windowing dependencies and does not change event physics.
package controls

import (
	"math"
	"skyofdie/internal/game"
)

type Keyboard struct {
	DirectionUsed                                        bool // includes the top-row 5 neutral key
	Up, Down, Left, Right                                bool
	UpPressed, DownPressed, LeftPressed, RightPressed    bool
	Primary, PrimaryPressed, Secondary, SecondaryPressed bool
}

type Mouse struct {
	X, Y                                                  int // original 320x200 coordinates
	Inside, Focused                                       bool
	Left, Right, LeftPressed, RightPressed, MiddlePressed bool
	Wheel                                                 float64
}

type Adapter struct {
	phase                        game.Phase
	event                        game.Event
	initialized, known, steering bool
	blocked                      bool
	x, y                         int
	wheel                        float64
}

// WASD cannot share A with a live KEYPAD TWO player. IJKL and the top-row
// digit aliases are always available, including in a two-player tube race.
func WASDEnabled(g *game.Game) bool {
	if g.Phase != game.Playing {
		return true
	}
	if g.Save.Controls[0] == 1 {
		return false
	}
	return !(g.Event == game.Tube && g.Mode != game.Practice &&
		g.ActivePlayer+1 < len(g.Players) && g.Save.Controls[1] == 1)
}

func hasDirection(in game.Input) bool { return in.Up || in.Down || in.Left || in.Right }

func (a *Adapter) Apply(g *game.Game, in game.Input, keys Keyboard, mouse Mouse) game.Input {
	if !mouse.Focused {
		a.initialized = false
		a.known, a.steering = false, false
		a.wheel = 0
		return game.Input{}
	}
	if !a.initialized || a.phase != g.Phase || a.event != g.Event {
		a.initialized = true
		a.phase, a.event = g.Phase, g.Event
		a.known, a.steering = false, false
		a.wheel = 0
		a.blocked = (mouse.Left || mouse.Right) && !mouse.LeftPressed && !mouse.RightPressed
	}
	if !mouse.Left && !mouse.Right {
		a.blocked = false
	}
	if a.blocked {
		mouse.Left, mouse.Right, mouse.LeftPressed, mouse.RightPressed = false, false, false, false
	}
	editing := g.Phase == game.Signup && g.Front.Editing
	if !editing {
		// A legacy direction has priority over an additional keyboard layout.
		if !hasDirection(in) {
			in.Up, in.Down = keys.Up && !keys.Down, keys.Down && !keys.Up
			in.Left, in.Right = keys.Left && !keys.Right, keys.Right && !keys.Left
		}
		in.UpPressed = in.UpPressed || keys.UpPressed
		in.DownPressed = in.DownPressed || keys.DownPressed
		in.LeftPressed = in.LeftPressed || keys.LeftPressed
		in.RightPressed = in.RightPressed || keys.RightPressed
	}
	in.Action = in.Action || keys.Primary
	in.ActionPressed = in.ActionPressed || keys.PrimaryPressed
	in.Confirm = in.Confirm || keys.PrimaryPressed
	in.Trick = in.Trick || keys.Secondary
	in.TrickPressed = in.TrickPressed || keys.SecondaryPressed
	in.AnyKey = in.AnyKey || keys.PrimaryPressed || keys.SecondaryPressed
	keyboardDirection := hasDirection(in) || keys.DirectionUsed || keys.Up || keys.Down || keys.Left || keys.Right
	moved := mouse.Inside && a.known && (mouse.X != a.x || mouse.Y != a.y)
	a.x, a.y, a.known = mouse.X, mouse.Y, mouse.Inside
	if keyboardDirection || !mouse.Inside {
		a.steering = false
	} else if moved || mouse.LeftPressed || mouse.RightPressed {
		a.steering = true
	}
	if !mouse.Inside {
		return buttons(in)
	}
	in.AnyKey = in.AnyKey || mouse.LeftPressed || mouse.RightPressed || mouse.MiddlePressed
	if mouse.MiddlePressed && g.Phase == game.Playing {
		in.Pause = true
	}
	if g.Phase != game.Playing && g.Phase != game.Resort && mouse.RightPressed {
		in.Escape = true
	}
	keyboardAction, keyboardEdge, keyboardConfirm := in.Action, in.ActionPressed, in.Confirm
	in.Action = in.Action || mouse.Left
	in.ActionPressed = in.ActionPressed || mouse.LeftPressed
	in.Confirm = in.Confirm || mouse.LeftPressed
	if g.Phase == game.Playing || g.Phase == game.Resort {
		in.Trick = in.Trick || mouse.Right
		in.TrickPressed = in.TrickPressed || mouse.RightPressed
	}
	if moved || mouse.LeftPressed {
		if !keyboardDirection {
			in.Pointer, in.PointerX, in.PointerY = true, mouse.X, mouse.Y
		}
	}
	switch g.Phase {
	case game.Playing:
		if g.Event == game.Snowball {
			a.wheel += mouse.Wheel
			whole := int(a.wheel)
			a.wheel -= float64(whole)
			in.ViewDelta -= whole
			if mouse.RightPressed {
				in.ViewDelta++
			}
		} else if a.steering && !keyboardDirection {
			setStick(&in, mouse.X-160, mouse.Y-100)
		}
	case game.Resort:
		if mouse.Left && !keyboardDirection {
			// Use the original turning and movement inputs. The skier still
			// obeys the original collision geometry; there is no teleport.
			in.Action, in.ActionPressed = keyboardAction, keyboardEdge
			dx, dy := mouse.X-int(g.MenuX)-10, mouse.Y-int(g.MenuY)+5
			if dx*dx+dy*dy > 16 {
				heading := targetHeading(g, dx, dy)
				delta := (heading - g.Front.Heading + 16) % 16
				if delta == 0 {
					in.Action = true
				} else if delta <= 8 {
					in.Right = true
				} else {
					in.Left = true
				}
			}
		}
	case game.Signup:
		if !editing && g.Front.Choosing && mouse.Wheel != 0 {
			a.wheel += mouse.Wheel
			whole := int(a.wheel)
			a.wheel -= float64(whole)
			in.LeftPressed = in.LeftPressed || whole > 0
			in.RightPressed = in.RightPressed || whole < 0
		}
	case game.Replay:
		if mouse.LeftPressed {
			// The original YES / NO labels are at y=88; their markers at 100.
			if mouse.Y >= 85 && mouse.Y <= 112 && mouse.X >= 120 && mouse.X <= 215 {
				in.LeftPressed, in.RightPressed = mouse.X < 170, mouse.X >= 170
			} else {
				in.ActionPressed, in.Confirm = keyboardEdge, keyboardConfirm
			}
		}
	case game.QuitPrompt:
		if mouse.LeftPressed && mouse.Y >= 100 && mouse.Y <= 120 {
			if mouse.X >= 190 && mouse.X < 206 {
				in.Text = "y"
			} else if mouse.X >= 206 && mouse.X < 222 {
				in.Text = "n"
			}
		}
	}
	return buttons(in)
}

func buttons(in game.Input) game.Input {
	if in.Action {
		in.Trick, in.TrickPressed = false, false
	}
	return in
}

func setStick(in *game.Input, x, y int) {
	in.Up, in.Down, in.Left, in.Right = false, false, false, false
	if x*x+y*y <= 18*18 {
		return
	}
	sector := (int(math.Round(math.Atan2(float64(y), float64(x))*4/math.Pi)) + 8) % 8
	in.Right = sector == 0 || sector == 1 || sector == 7
	in.Down = sector == 1 || sector == 2 || sector == 3
	in.Left = sector == 3 || sector == 4 || sector == 5
	in.Up = sector == 5 || sector == 6 || sector == 7
}

func targetHeading(g *game.Game, dx, dy int) int {
	best, score := g.Front.Heading, math.Inf(-1)
	for h := 1; h <= 16; h++ {
		x, y := g.Config.Front.ResortDX[h], g.Config.Front.ResortDY[h]
		length := math.Hypot(float64(x), float64(y))
		if length == 0 {
			continue
		}
		dot := float64(dx*x+dy*y) / length
		if dot > score {
			best, score = h, dot
		}
	}
	return best
}
