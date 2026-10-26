package game

// Acro Aerials uses the original ramp path, 8.8 vertical velocity, fixed
// horizontal drift, animation dispatch, landing curve, and five-judge total.
type AirState struct {
	RampSound                                                                       bool
	WaitAnim                                                                        Animation
	Reaction                                                                        int
	Judge, JudgeFrame, JudgeTimer                                                   int
	Judging, Judged                                                                 bool
	JudgeFaces                                                                      [5]int
	X, Y, Speed, VY, Camera                                                         float64
	Stage, Attempts, Timer, Tick, Score, Pending, Trick, Combo, LastDirection       int
	Sprite                                                                          int
	OffsetX, OffsetY                                                                int
	Mirror, Landed, Active                                                          bool
	Anim                                                                            Animation
	RampIndex, RampVelocity, RampBoost, Pumps, Velocity, Height, ScrollStage, Drift int
	UnsafeTicks, LandingFrame                                                       int
	Judges                                                                          [5]float64
	LastScore                                                                       int
	History                                                                         [3]int
	Used                                                                            [16]bool
}

func (g *Game) initAir(score, attempts int, hist [3]int) {
	g.Air = AirState{Score: score, Attempts: attempts, History: hist, LastDirection: 0, Height: 120, RampVelocity: 1}
	g.Air.X = 11
	g.Air.Y = 67
	m := g.Config.Front.AirWait
	g.Air.X = float64(m.FixedX)
	g.Air.Y = float64(m.FixedY)
	g.Air.Sprite = m.Frames[0][0]
	g.Air.Judge = -1
}
func (g *Game) launchAir() {
	a := &g.Air
	a.Stage = 1
	a.X = 10
	a.Y = 80
	a.Camera = 152
	a.Height = 120
	a.Velocity = -min(8, 2+a.RampBoost) * 256
	a.VY = float64(a.Velocity) / 256
	a.Sprite = g.Config.Stunts.Air.Idle[0]
	a.Active = false
	a.Pending = 0
	a.ScrollStage = 0
	a.Drift = 2
	a.OffsetX, a.OffsetY = 0, 0
}
func (g *Game) moveAir() {
	a := &g.Air
	a.Velocity += 16
	delta := max(-4, min(6, a.Velocity>>8))
	a.VY = float64(a.Velocity) / 256
	switch a.ScrollStage {
	case 0:
		a.Y += float64(delta)
		a.Height += delta
		if a.Height <= 100 {
			a.ScrollStage = 1
		}
	case 1:
		if a.Camera >= 224 {
			a.ScrollStage = 2
			a.Camera++
			a.Y += float64(delta - 1)
		} else {
			a.Camera += float64(delta)
		}
	case 2:
		cameraStep := 0
		if a.Camera < 233 {
			cameraStep = 1
			a.Camera++
		}
		a.Y += float64(delta - cameraStep)
		a.Height += delta
		if a.Height >= 195 {
			a.Velocity = 0
		}
	}
	a.Drift++
	if a.Drift >= 3 {
		a.Drift = 0
		a.X = min(300, a.X+1)
	}
}
func (g *Game) airPoints(action, loops, penalty int) int {
	a := &g.Air
	base := g.Config.Stunts.Air.Scores[action]
	points := 0
	if !a.Used[action] {
		a.Used[action] = true
		points = base
		loops--
	}
	if loops > 0 {
		points += (base / 2) * loops
	}
	return max(0, points-penalty)
}
func (g *Game) airStart(in Input) {
	a := &g.Air
	dir := stuntDirection(in)
	if dir == 0 || in.Trick {
		return
	}
	action := dir - 1
	if in.Action || in.ActionPressed {
		action += 8
	} else if dir != 3 && dir != 4 && dir != 7 && dir != 8 {
		return
	}
	a.Trick = action
	a.Anim = g.animation(g.Config.Stunts.Air.Actions[action], dir)
	a.Active = true
	a.Combo++
}
func (g *Game) airAnimation(in Input) {
	a := &g.Air
	m := g.Config.Stunts.Motions[a.Anim.Motion]
	dir := stuntDirection(in)
	// The original direction, not arbitrary angle integration, controls loops.
	g.animate(&a.Anim, dir)
	a.Sprite = a.Anim.Visible[0]
	a.Mirror = a.Anim.Visible[1] != 0
	a.OffsetX, a.OffsetY = a.Anim.Visible[2], a.Anim.Visible[3]
	if a.UnsafeTicks > 0 {
		a.UnsafeTicks--
		if a.UnsafeTicks == 0 {
			a.Active = false
		}
		return
	}
	if a.Anim.Expired {
		a.Pending = max(0, a.Pending-3)
	}
	if m.Kind >= 2 && m.Kind != 4 && a.Anim.Direction < 0 {
		if safePose(m, a.Anim.Frame, 0) {
			a.Pending += g.airPoints(a.Trick, a.Anim.Loops+1, 0)
			a.Active = false

		} else if safePose(m, a.Anim.Frame, 1) {
			a.Pending += g.airPoints(a.Trick, a.Anim.Loops+1, 3)
			a.Active = false

		} else if a.UnsafeTicks == 0 {
			a.Pending += g.airPoints(a.Trick, a.Anim.Loops+1, 0) / 2
			a.UnsafeTicks = 40

		}
	}
	if a.Anim.Done {
		a.Pending += g.airPoints(a.Trick, 1, 0)
		a.Active = false

	}
}
func (g *Game) landAir(frame int) {
	a := &g.Air
	a.Stage = 2
	a.Attempts++
	a.LandingFrame = frame
	// 0x2393 selects the rollout record after drawing this flight frame.
	// The first rollout position is applied on the following game step.
	a.Landed = a.UnsafeTicks == 0
	if a.Active {
		if a.Anim.Holding {
			// 0x23d4 awards the unfinished move at half value before the
			// separate failed-landing penalty in 0x23ef.
			a.Pending += g.airPoints(a.Trick, a.Anim.Loops+1, 0) / 2
		}
		a.Landed = a.Landed && !a.Anim.Holding
	}
	if a.Landed {
		a.Pending += 20
		g.Cue("fx12")

	} else {
		a.Pending /= 2
		g.Cue("fx04")

	}
	base := a.Pending
	a.LastScore = 0
	a.Judges = [5]float64{}
	a.Timer = 0
	a.Active = false
	if !a.Landed {
		data := g.Config.Stunts.Air
		choice := 0
		for choice < 4 && base > data.FailureThresholds[choice] {
			choice++
		}
		a.Anim = g.animation(data.Failure[choice], 0)
	}
}
func (g *Game) stepAir(in Input) {
	a := &g.Air
	a.Tick++
	if a.Stage == 0 && a.RampIndex >= 163 {
		g.launchAir()
	}
	switch a.Stage {
	case 0:
		// Only alternating horizontal directions build the takeoff boost.
		side := -1
		dir := inputDirection(in)
		if dir >= 1 && dir <= 3 {
			side = 1
		}
		if dir >= 5 && dir <= 7 {
			side = 0
		}
		if side >= 0 && side != a.LastDirection {
			if a.LastDirection >= 0 {
				a.Pumps++
				if a.Pumps == 13 {
					a.Pumps = 0
					a.RampBoost++
				}
			}
			a.LastDirection = side
		}
		a.RampVelocity += 19
		a.RampIndex += min(3, min(1, a.RampVelocity>>8)+a.RampBoost)
		f := g.Config.Stunts.Air.Ramp[min(156, a.RampIndex)]
		a.X = float64(f[0])
		a.Y = float64(f[1])
		a.Sprite = f[2]
		a.Mirror = false
		a.OffsetX, a.OffsetY = 0, 0
		a.RampIndex++
		if a.RampIndex >= 135 && !a.RampSound {
			a.RampSound = true
			g.Cue("fx18")
			g.Cue("fx06")
		}
		a.Speed = float64(2 + a.RampBoost)
	case 1:
		if !a.Active {
			g.airStart(in)
		}
		g.moveAir()
		if a.Active {
			g.airAnimation(in)
		} else {
			v := 2
			if a.Velocity < 0 {
				v = 0
			} else if a.Velocity > 255 {
				v = 1
			}
			a.Sprite = g.Config.Stunts.Air.Idle[v]
			a.Mirror = false
			a.OffsetX, a.OffsetY = 0, 0
		}
		if a.ScrollStage == 2 {
			for i, f := range g.Config.Stunts.Air.Landing {
				if a.Height >= g.Config.Stunts.Air.Landing[0][1] && a.Height <= f[1] && a.X+4 < float64(f[0]) {
					g.landAir(i)
					break
				}
			}
		}
		if a.X >= 300 {
			g.landAir(51)
		}
	case 2:
		a.Timer--
		if a.Landed {
			f := g.Config.Stunts.Air.Landing[min(51, a.LandingFrame)]
			a.X = float64(f[0])
			a.Y += float64(f[1] - a.Height)
			a.Height = f[1]
			a.Sprite = f[2]
			a.Mirror = false
			a.OffsetX, a.OffsetY = 0, 0
			a.LandingFrame++
			if a.LandingFrame == len(g.Config.Stunts.Air.Landing) {
				g.airReaction()
			}
		} else if !a.Anim.Done {
			g.animate(&a.Anim, 0)
			a.Sprite = a.Anim.Visible[0]
			a.Mirror = a.Anim.Visible[1] != 0
			a.OffsetX, a.OffsetY = a.Anim.Visible[2], a.Anim.Visible[3]
		}
		if !a.Landed && a.Anim.Done {
			g.beginAirJudging()
		}
	case 3:
		if !a.Anim.Done {
			g.animate(&a.Anim, 0)
			f := a.Anim.Visible
			a.Sprite = f[0]
			a.Mirror = f[1] != 0
			a.OffsetX, a.OffsetY = f[2], f[3]
			m := g.Config.Stunts.Motions[a.Anim.Motion]
			if m.Mode == 2 {
				a.X = float64(m.FixedX)
				a.Y = float64((m.FixedY - int(a.Camera-152) + 200) % 200)
			}
			if !a.Anim.Done {
				return
			}
		}
		a.Stage = 4
		a.Judged = true
		a.Judging = false
	case 4:
		if !a.Judged {
			a.JudgeTimer++
			if a.JudgeTimer >= 5 {
				a.JudgeTimer = 0
				a.JudgeFrame++
				if a.JudgeFrame == 5 {
					g.airJudgeCard()
					a.Judge++
					a.JudgeFrame = 0
					g.Cue("fx08")
					if a.Judge == 5 {
						a.History[a.Attempts-1] = a.LastScore
						if a.Landed {
							// The original celebrates after all five cards.
							a.Stage = 3
							g.animate(&a.Anim, 0)
							a.Sprite = a.Anim.Visible[0]
							a.Mirror = a.Anim.Visible[1] != 0
							a.OffsetX, a.OffsetY = a.Anim.Visible[2], a.Anim.Visible[3]
						} else {
							a.Judged = true
							a.Judging = false
						}
					}
				} else {
					a.JudgeFaces[a.Judge] = a.JudgeFrame
				}
			}
			return
		}
		if in.Confirm || in.ActionPressed {
			if g.Mode != Practice {
				// Original 0x1e08 records each acknowledged jump. Esc in a
				// later attempt retains those earlier scores in the event table.
				g.Players[g.ActivePlayer].Scores[Aerials] = a.Score
			}
			if a.Attempts == 3 && g.Mode != Practice {
				g.Finish(a.Score, "")
				return
			}
			attempts := a.Attempts
			if g.Mode == Practice {
				attempts = 0
			}
			g.initAir(a.Score, attempts, a.History)
			g.Waiting = true
			g.FrameTick = 0
		}
	}
}
func (g *Game) stepAirWait() {
	a := &g.Air
	m := g.Config.Front.AirWait
	advanceMotion(&a.WaitAnim, m, 0)
	f := a.WaitAnim.Visible
	a.Sprite = f[0]
	a.Mirror = f[1] != 0
	a.X = float64(m.FixedX + f[2])
	a.Y = float64(m.FixedY + f[3])
	a.OffsetX, a.OffsetY = 0, 0
}
func (g *Game) airReaction() {
	a := &g.Air
	data := g.Config.Stunts.Air
	choice := 0
	for choice < 3 && a.Pending > data.SuccessThresholds[choice] {
		choice++
	}
	a.Anim = g.animation(data.Success[choice], 0)
	g.beginAirJudging()
}

func (g *Game) beginAirJudging() {
	a := &g.Air
	a.Stage = 4
	a.Judging = true
	a.Judge = 0
	a.JudgeFrame = 0
	// The first pose is already visible when the original starts its wait.
	a.JudgeTimer = -1
}

func (g *Game) airJudgeCard() {
	a := &g.Air
	jitter := g.Random(5)
	if g.Random(256) < 127 {
		jitter = -jitter
	}
	value := max(0, min(100, a.Pending+jitter))
	a.Judges[a.Judge] = float64(value) / 10
	a.LastScore += value
	a.Score += value
}
