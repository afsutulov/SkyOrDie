package game

// These records are extracted data, not interpreted executable instructions.
// Addresses refer to the EXEPACK-unpacked SKIEGA load image, DS base 0x8190.
type Motion struct {
	Mode, FixedX, FixedY, SafeFrame                       int
	Kind, HoldTicks, HoldFrame, Delay, LoopStart, LoopEnd int
	Frames                                                [][4]int // sprite, mirror, signed dx, signed dy
}
type DownhillData struct {
	RespawnX                  []int
	Lateral                   [7]int
	SteerClass                [7]int
	CrashActions              [8]int
	TrickActions, TrickScores [9]int
	Ground                    [3][7][][4]int
	Actions                   [][3]Motion
}
type TrailStamp struct {
	X, Y, Sprite int
	Mirror       bool
}
type DownhillState struct {
	Sliding                                                  bool
	RouteAllowed                                             [2]bool
	RouteAwarded                                             [2]bool
	MogulCount                                               int
	MogulBonus                                               bool
	Finishing                                                bool
	FinishAction, FinishBonus                                int
	FinishAnim                                               Animation
	LastGroundDirection                                      int
	IntroPhase                                               int
	IntroAnim                                                Animation
	Trail                                                    []TrailStamp
	IntroMark                                                bool
	Heading, TurnDelay, SpeedDelay, GroundDelay, GroundFrame int
	Action, Variant, Frame, Delay, Hold, Direction           int
	Animating, Crashing, Recovering, Mirror                  bool
	OffsetX, OffsetY                                         int
	Tricks                                                   [9]bool
	Combo, CrashCount                                        int
}

// DownhillTerrain decodes the bit fields used by the original 0x1C2E routine.
// Bits 0..9 select art. Bits 10..12 select a fall; 13..14 select snow,
// ice, moguls, or a hazard. Bit 15 enables player-initiated stunts.
type DownhillTerrain struct {
	Surface, Fall int
	Stunts        bool
}

func DecodeDownhillTerrain(cell uint16) DownhillTerrain {
	return DownhillTerrain{Surface: int(cell>>13) & 3, Fall: int(cell>>10) & 7, Stunts: cell&0x8000 != 0}
}
func (g *Game) initDownhill() {
	p := &g.Race.Player
	p.X = 158
	p.Y = 103
	p.Speed = 0
	p.Downhill = DownhillState{Heading: 4, TurnDelay: 2, SpeedDelay: 3, GroundDelay: 3, RouteAllowed: [2]bool{true, true}, MogulCount: 1}
	d := &p.Downhill
	d.IntroAnim = Animation{Remaining: g.Config.Downhill.Actions[0][0].HoldTicks}
	f := g.Config.Downhill.Actions[0][0].Frames[0]
	p.Sprite = f[0]
	d.OffsetX = f[2]
	d.OffsetY = f[3]
}
func inputDirection(in Input) int {
	// Original direction numbering runs clockwise from north-east.
	if in.Up {
		if in.Left {
			return 7
		}
		if in.Right {
			return 1
		}
		return 8
	}
	if in.Down {
		if in.Left {
			return 5
		}
		if in.Right {
			return 3
		}
		return 4
	}
	if in.Left {
		return 6
	}
	if in.Right {
		return 2
	}
	return 0
}
func (g *Game) downhillGroundFrame() {
	p := &g.Race.Player
	d := &p.Downhill
	terrain := DecodeDownhillTerrain(g.Config.Courses[0].Cell(p.X, p.Y))
	frames := g.Config.Downhill.Ground[min(terrain.Surface, 2)][d.Heading]
	f := frames[d.GroundFrame%len(frames)]
	p.Sprite = f[0]
	d.Mirror = f[1] != 0
	d.OffsetX = f[2]
	d.OffsetY = f[3]
	p.Z = 0
}
func (g *Game) downhillBegin(action, dir int, crash bool) {
	p := &g.Race.Player
	d := &p.Downhill
	d.Action = action
	d.Variant = g.Config.Downhill.SteerClass[d.Heading]
	d.Direction = dir
	d.Frame = 0
	d.Delay = 0
	d.Hold = g.Config.Downhill.Actions[action][d.Variant].HoldTicks
	d.Animating = true
	d.Crashing = crash
	d.Recovering = false
	if crash {
		d.CrashCount++
		p.Speed = 0
		p.Combo = 0
		d.Combo = 0
		g.Cue("fx04")
	} else {
		p.Speed = 2
		g.Cue("fx05")
	}
}
func (g *Game) downhillFall(terrain DownhillTerrain) {
	action := g.Config.Downhill.CrashActions[terrain.Fall]
	g.downhillBegin(action, 0, true)
	// Five calls to 0x1ae2 add five displayed seconds and restart the
	// fractional counter; 0x1ad3 then consumes this frame's first tick.
	g.Race.Elapsed = (g.Race.Elapsed/26+5)*26 + 1
}
func (g *Game) downhillAnimation(in Input) {
	p := &g.Race.Player
	d := &p.Downhill
	m := g.Config.Downhill.Actions[d.Action][d.Variant]
	f := m.Frames[d.Frame]
	p.Sprite = f[0]
	d.Mirror = f[1] != 0
	d.OffsetX = f[2]
	d.OffsetY = f[3]
	p.Z = float64(max(0, -f[3]))
	if !d.Crashing {
		p.X += float64((d.Variant - 1) * int(p.Speed))
		p.X = clamp(p.X, 15, 290)
	}
	held := m.Kind == 1 && !d.Crashing && inputDirection(in) == d.Direction && d.Hold > 1
	if held {
		d.Hold--
	}
	d.Delay--
	if d.Delay >= 0 {
		return
	}
	d.Delay = m.Delay
	if held && d.Frame == m.HoldFrame {
		return
	}
	d.Frame++
	if d.Frame < len(m.Frames) {
		return
	}
	d.Frame = len(m.Frames) - 1
	d.Animating = false
	p.Z = 0
	d.OffsetX = 0
	d.OffsetY = 0
	if d.Crashing {
		d.Recovering = true

		return
	}
	terrain := DecodeDownhillTerrain(g.Config.Courses[0].Cell(p.X, p.Y))
	if terrain.Surface == 3 {
		g.downhillFall(terrain)
		return
	}
	points := g.Config.Downhill.TrickScores[d.Direction]
	if d.MogulBonus {
		points += d.MogulCount * 100
		d.MogulBonus = false
	}
	p.Score += points
	if d.Direction >= 0 && d.Direction < len(d.Tricks) {
		d.Tricks[d.Direction] = true
	}
	p.Speed = 1
	d.Combo = 0
	g.Cue("fx12")
	if points > 0 {

	}
}
func (g *Game) stepDownhill(in Input) {
	r := &g.Race
	p := &r.Player
	d := &p.Downhill
	if d.Finishing {
		m := g.Config.Downhill.Actions[d.FinishAction][d.Variant]
		advanceMotion(&d.FinishAnim, m, 0)
		f := d.FinishAnim.Visible
		p.Sprite = f[0]
		d.Mirror = f[1] != 0
		d.OffsetX = f[2]
		d.OffsetY = f[3]
		if d.FinishAnim.Done {
			g.Finish(p.Score, "")
		}
		return
	}
	if d.Recovering {
		if !(in.ActionPressed || in.Confirm) {
			return
		}
		row := max(0, min(499, int(p.Y)/8))
		p.X = float64(g.Config.Downhill.RespawnX[row] + 10)
		d.Recovering = false
		d.Crashing = false
		d.OffsetX = 0
		d.OffsetY = 0
		d.Heading = 4
		d.TurnDelay = 2
		d.SpeedDelay = 3
		p.Speed = 0
		p.Z = 0
		p.Invuln = 0

		g.downhillGroundFrame()
		return
	}
	r.Elapsed++
	if d.Animating {
		g.downhillAnimation(in)
	} else {
		terrain := DecodeDownhillTerrain(g.Config.Courses[0].Cell(p.X, p.Y))
		if terrain.Surface == 3 {
			g.downhillFall(terrain)
			g.downhillAnimation(in)
		} else {
			dir := inputDirection(in)
			if terrain.Stunts && (in.Action || in.ActionPressed) && !in.Trick {
				action := g.Config.Downhill.TrickActions[dir]
				if action != 255 {
					if terrain.Surface == 2 {
						d.MogulCount++
						d.MogulBonus = true
					}
					g.downhillBegin(action, dir, false)
					g.downhillAnimation(in)
				}
			}
			if !d.Animating {
				// Original ground input preserves the last direction while button one is held.
				if in.Action && !in.Trick {
					dir = d.LastGroundDirection
				} else {
					d.LastGroundDirection = dir
				}
				// Original EGA counters at 0x16D6..0x174E.
				if dir == 7 || dir == 8 || dir == 1 {
					d.SpeedDelay--
					if d.SpeedDelay < 0 {
						d.SpeedDelay = 7
						p.Speed = max(0, p.Speed-1)
					}
				} else if dir == 4 {
					d.SpeedDelay--
					if d.SpeedDelay < 0 {
						d.SpeedDelay = 10
						p.Speed = min(3, p.Speed+1)
					}
				}
				if dir == 6 || dir == 2 {
					d.TurnDelay--
					if d.TurnDelay < 0 {
						d.TurnDelay = 2
						if dir == 6 {
							d.Heading = max(0, d.Heading-1)
						} else {
							d.Heading = min(6, d.Heading+1)
						}
					}
				}
				p.VX = 0
				if p.Speed > 0 {
					p.VX = float64(g.Config.Downhill.Lateral[d.Heading])
					p.X = clamp(p.X+p.VX, 15, 290)
				}
				d.GroundDelay--
				if d.GroundDelay < 0 {
					d.GroundDelay = 3
					d.GroundFrame++
				}
				g.downhillGroundFrame()
				if p.Speed > 0 && r.Elapsed%2 == 0 {
					f := g.Config.Front.Trail[min(terrain.Surface, 2)][d.Heading]
					d.Trail = append(d.Trail, TrailStamp{int(p.X) - 10 + d.OffsetX + f[2], int(p.Y) + d.OffsetY + f[3], f[0], f[1] != 0})
				}
			}
		}
	}
	if !d.Crashing && !d.Recovering {
		p.Y += p.Speed
	}
	if p.Speed > 0 && !d.Animating && !d.Sliding {
		d.Sliding = true
		g.Cue("fx17")
	}
	if (p.Speed == 0 || d.Animating) && d.Sliding {
		d.Sliding = false
		g.Cue("fx18")
	}
	p.LastTerrain = int(g.Config.Courses[0].Cell(p.X, p.Y) >> 10)
	row := int(p.Y) / 8
	if row >= 150 && row <= 208 && p.X < 85 {
		d.RouteAllowed[0] = false
	}
	if row == 340 && p.X >= 240 {
		d.RouteAllowed[1] = false
	}
	for i, r := range [2]int{208, 345} {
		if row == r && !d.RouteAwarded[i] && d.RouteAllowed[i] {
			p.Score += [2]int{1000, 1500}[i]
			d.RouteAwarded[i] = true
			g.Cue("fx10")
		}
	}

	if p.Y >= r.FinishY {
		// Bonus uses the original displayed clock and recovered ending poses.
		bonus := 0
		for _, used := range d.Tricks {
			if used {
				bonus += 500
			}
		}
		if d.CrashCount == 0 {
			bonus += 1000
		}
		r.Won = true
		seconds := r.Elapsed / 26
		if seconds < 120 {
			bonus += (1000 - (seconds/60*100 + seconds%60)) * 10
		}
		p.Score += bonus
		d.FinishBonus = bonus
		d.Finishing = true
		d.FinishAction = 9
		if p.Score >= 10000 {
			d.FinishAction = 7
		}
		if p.Score >= 13500 {
			d.FinishAction = 8
		}
		d.FinishAnim = Animation{}
		p.Speed = 0
		return
	}
}

// The opening fall and standing loop are actions 0 and 1, before the race clock.
func (g *Game) stepDownhillIntro(in Input) {
	p := &g.Race.Player
	d := &p.Downhill
	m := g.Config.Downhill.Actions[d.IntroPhase][0]
	advanceMotion(&d.IntroAnim, m, 0)
	f := d.IntroAnim.Visible
	p.Sprite = f[0]
	d.Mirror = f[1] != 0
	d.OffsetX = f[2]
	d.OffsetY = f[3]
	if d.IntroPhase == 0 {
		if d.IntroAnim.Frame == 8 && d.IntroAnim.Delay == m.Delay {
			g.Cue("fx04")
		}
		if d.IntroAnim.Done {
			d.IntroPhase = 1
			d.IntroMark = true
			p.X = 174
			d.IntroAnim = Animation{Remaining: 255}
		}
	} else if in.ActionPressed || in.Confirm {
		g.Waiting = false
		g.FrameTick = 0
		g.Race.Elapsed = 25
		g.downhillGroundFrame()
	}
}
