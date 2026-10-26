package game

// The board follows the recovered 217-position cross-section, not a parabola.
// Stunts use the original side/terrain/speed/direction dispatch and pose records.
type PipeState struct {
	BankReturning                                                           bool
	CommentVisible, CommentPending                                          bool
	Finishing                                                               bool
	FinishBonus                                                             int
	FinishAnim                                                              Animation
	OnBump                                                                  bool
	BumpTicks                                                               int
	Decorative                                                              []PipeObject
	DecorDelay                                                              int
	BashEnabled, BashPlaying                                                bool
	BashFrame, BashDelay                                                    int
	Starting                                                                bool
	WaitAnim                                                                Animation
	StartFlagFrame                                                          int
	Comment, CommentIndex, CommentTimer, FaceTimer, FaceFrame               int
	X, VX, Z, ScreenX, ScreenY                                              float64
	Time, Score, Pending, Combo, Tick, AirTick, Crash, Trick, TrickFrame    int
	Index, Velocity, Energy, Heading, TurnDelay, BankTimer, Zone, Direction int
	Sprite, OffsetX, OffsetY                                                int
	Mirror, Active, Recovering, Ending, NoCrash                             bool
	Anim                                                                    Animation
	Used                                                                    [256]bool
	Unique                                                                  int
	Rails                                                                   []int
	RailDelay, RailSpawn, ObstacleDelay                                     int
	Objects                                                                 []PipeObject
}
type PipeObject struct {
	Path, Kind, Frame int
	Done              bool
}

func (g *Game) initPipe() {
	g.Pipe = PipeState{Index: 100, Zone: 4, Heading: 0, Time: 120 * 26, NoCrash: true, Comment: 2, FaceFrame: 2, FaceTimer: 119, StartFlagFrame: 14, Starting: true, BumpTicks: 15}
	g.pipePosition()
	p := &g.Pipe
	p.Sprite = g.Config.Stunts.Motions[g.Config.Stunts.Pipe.Wait].Frames[0][0]
	p.WaitAnim = Animation{Remaining: 255}
	p.Anim = g.animation(g.Config.Stunts.Pipe.Start, 0)
	for i := 0; i < 200; i++ {
		g.pipeDecorative()
		g.pipeRails()
	}
}
func pipeZone(index int) int {
	for i, n := range []int{30, 38, 55, 70, 145, 160, 172, 182} {
		if index <= n {
			return i
		}
	}
	return 8
}
func (g *Game) pipePosition() {
	p := &g.Pipe
	data := g.Config.Stunts.Pipe
	p.Index = max(1, min(216, p.Index))
	i := p.Index
	x := 0
	if i > 108 {
		i = 216 - i
		x = 290 - data.X[i]
	} else {
		x = data.X[i]
	}
	p.ScreenX = float64(x)
	p.ScreenY = float64(data.Y[i])
	p.X = p.ScreenX - 145
	p.VX = float64(p.Velocity) / 256
	p.Z = 0
	f := data.Ground[p.Zone][p.Heading]
	p.Sprite = f[0]
	p.Mirror = f[1] != 0
	p.OffsetX = f[2]
	p.OffsetY = f[3]
}
func (g *Game) pipeMove(dir int) {
	p := &g.Pipe
	accel := func(sign int) {
		v := p.Velocity >> 8
		if sign < 0 {
			p.Direction = 6
			if v != -2 {
				p.Velocity -= 45
			}
		} else {
			p.Direction = 2
			if v < 2 {
				p.Velocity += 45
			}
		}
		p.Index = max(1, min(216, p.Index+v))
	}
	if p.Index <= 50 {
		p.BankTimer--
		if p.BankTimer < 0 {
			p.BankTimer = 0
			p.BankReturning = true
			accel(1)
		} else {
			accel(-1)
		}
	} else if p.Index >= 165 {
		p.BankTimer--
		if p.BankTimer < 0 {
			p.BankTimer = 0
			p.BankReturning = true
			accel(-1)
		} else {
			accel(1)
		}
	} else {
		if p.BankReturning && p.Zone != 4 {
			dir = 0
		}
		if dir == 2 || dir == 6 {
			p.BankReturning = false
			p.BankTimer = g.Config.Stunts.Pipe.BankTicks[p.Energy>>8]
		}
		switch dir {
		case 6:
			p.Energy = min(767, p.Energy+3)
			accel(-1)
		case 2:
			p.Energy = min(767, p.Energy+3)
			accel(1)
		default:
			p.Energy = max(0, p.Energy-2)
			if p.Index < 70 {
				accel(1)
			} else if p.Index > 145 {
				accel(-1)
			} else {
				v := p.Velocity >> 8
				if v < 0 {
					p.Velocity += 15
				} else if v > 0 {
					p.Velocity -= 15
				}
				p.Index += v
			}
		}
	}
	if p.Direction == 6 || p.Direction == 2 {
		p.TurnDelay--
		if p.TurnDelay < 0 {
			p.TurnDelay = 4
			if p.Direction == 6 {
				p.Heading = max(0, p.Heading-1)
			} else {
				p.Heading = min(6, p.Heading+1)
			}
			g.pipePosition()
			return
		}
	}
	// The right branch falls through to 0x511e when its turn counter has
	// not expired; the left branch jumps past it. Preserve that asymmetry.
	if p.Direction != 6 && p.Zone == 4 && p.Velocity>>8 == 0 {
		p.TurnDelay--
		if p.TurnDelay < 0 {
			p.TurnDelay = 4
			if p.Heading < 3 {
				p.Heading++
			} else if p.Heading > 3 {
				p.Heading--
			}
		}
	}
	g.pipePosition()
}
func (g *Game) pipeStart(in Input) {
	p := &g.Pipe
	if !(in.Action || in.ActionPressed) {
		return
	}
	dir := stuntDirection(in)
	if dir == 0 || dir%2 != 0 {
		return
	}
	side := 0
	if p.Index >= 108 {
		side = 1
	}
	env := 0
	if p.OnBump {
		env = 1
	}
	if p.Index >= 25 && p.Index <= 37 || p.Index >= 178 && p.Index <= 180 {
		env = 2
	} else if p.Zone != 4 {
		return
	}
	action := side*64 + env*16 + (p.Energy>>8)*4 + dir/2 - 1
	p.Trick = action
	p.Anim = g.animation(g.Config.Stunts.Pipe.Actions[action], dir)
	p.Active = true
	p.Combo++
	g.Cue("fx18")
}
func (g *Game) pipeAward() {
	p := &g.Pipe
	data := g.Config.Stunts.Pipe
	if p.Anim.Expired {
		g.pipeEndPose()
		return
	}
	points := data.Scores[p.Trick]
	id := data.Unique[p.Trick]
	if p.Used[id] {
		points /= 2
	} else {
		p.Used[id] = true
		p.Unique++
	}
	if p.Trick == 32 || p.Trick == 34 || p.Trick == 39 || p.Trick == 96 || p.Trick == 98 || p.Trick == 103 {
		points += max(0, 255-p.Anim.Remaining) >> 4
	}
	p.Score += points
	p.Pending = points
	if points > 0 {
		g.Cue("fx12")
	}
	// Original 0x5190: the high byte of energy selects groups 2..4.
	p.FaceFrame = 2 + (p.Energy >> 8)
	p.CommentPending = true
	g.pipeEndPose()
}

func (g *Game) pipeEndPose() {
	p := &g.Pipe
	p.Active = false
	p.Anim.Done = true
	if p.Zone == 4 {
		p.Velocity = 0
	} else {
		if p.Index > 108 {
			p.Velocity = (p.Velocity & 255) - 512
			p.Direction = 6
		} else {
			p.Velocity = (p.Velocity & 255) | 512
			p.Direction = 2
		}
	}
}
func (g *Game) pipeCrash() {
	p := &g.Pipe
	p.NoCrash = false
	elapsed := 120*26 - p.Time
	p.Time = max(0, 120*26-((elapsed/26+5)*26+1))
	p.Crash = 1
	p.Active = true
	p.Pending = 0
	p.Energy = 0
	variant := 0
	if p.Zone == 4 {
		variant = 4
	}
	if p.Index > 108 {
		variant += 2
	}
	// Deterministic visual variant uses the game's own seeded random source.
	if g.Random(256) >= 127 {
		variant++
	}
	p.Anim = g.animation(g.Config.Stunts.Pipe.Crashes[variant], 0)
	g.Cue("fx04")
	p.FaceFrame = 0
	g.showPipeComment(0)

}
func (g *Game) pipeAnimate(in Input) {
	p := &g.Pipe
	m := g.Config.Stunts.Motions[p.Anim.Motion]
	if m.Kind > 0 && m.Mode == 0 {
		g.pipeMove(p.Direction)
	} else {
		g.pipePosition()
	}
	g.animate(&p.Anim, stuntDirection(in))
	f := p.Anim.Visible
	p.Sprite = f[0]
	p.Mirror = f[1] != 0
	p.OffsetX = f[2]
	p.OffsetY = f[3]
	if m.Kind > 0 && m.Mode == 2 {
		p.ScreenX = float64(m.FixedX)
		p.ScreenY = float64(m.FixedY)
	}
	p.Z = float64(max(0, -f[3]))
	p.AirTick++
	if p.Crash > 0 {
		if p.Anim.Done {
			p.Crash = 0
			p.Recovering = true
			p.Active = false
		}
		return
	}
	if p.Anim.Expired {
		g.pipeEndPose()
		g.pipeCrash()
		return
	}
	// 0x5394 checks every active pose before its completion award.
	reentry := p.Index < 108 && p.Direction == 2 && p.Index >= 35 || p.Index >= 108 && p.Direction == 6 && p.Index <= 182
	if p.Zone != 4 && reentry {
		if p.Anim.Holding || pipeLoopPose(p.Trick) && !pipeSafePose(m, p.Anim.Frame, p.Trick%2 == 0) {
			g.pipeEndPose()
			g.pipeCrash()
			return
		}
		g.pipeEndPose()
	}
	if p.Anim.Done {
		g.pipeAward()
	}
}

func pipeLoopPose(action int) bool {
	return action >= 36 && action <= 43 || action >= 100 && action <= 107
}

// The original lip tolerances are asymmetric: even entries accept +2,
// odd entries accept -2. Wrap boundaries follow 0x541c, including tails.
func pipeSafePose(m Motion, frame int, even bool) bool {
	if frame == m.SafeFrame {
		return true
	}
	previous := m.SafeFrame - 1
	if previous < 0 {
		previous = m.LoopEnd
	}
	if frame == previous {
		return true
	}
	if !even {
		previous = m.SafeFrame - 2
		if previous < 0 {
			previous += m.LoopEnd + 1
		}
		if frame == previous {
			return true
		}
	}
	next := m.SafeFrame + 1
	if next > m.LoopEnd {
		next = m.LoopStart
	}
	if frame == next {
		return true
	}
	if even {
		next = m.SafeFrame + 2
		if next > m.LoopEnd {
			next -= m.LoopEnd + 1
		}
		return frame == next
	}
	return false
}
func (g *Game) pipeScenery() {
	p := &g.Pipe
	delay := [3]int{4, 3, 2}[p.Energy>>8]
	g.pipeRails()
	g.pipeDecorative()

	if len(p.Objects) < 2 && g.Random(256) <= 45 {
		kind := 0
		r := g.Random(256)
		switch {
		case r <= 25:
			kind = 2
		case r <= 50:
			kind = 3
		case r <= 100:
			kind = 1
		}
		p.Objects = append(p.Objects, PipeObject{Path: g.Random(3), Kind: kind})
	}
	p.ObstacleDelay--
	if p.ObstacleDelay < 0 {
		p.ObstacleDelay = delay
		live := p.Objects[:0]
		for _, o := range p.Objects {
			o.Frame++
			if o.Frame < 18 && !o.Done {
				live = append(live, o)
			}
		}
		p.Objects = live
	}
	if p.Active || p.Recovering {
		return
	}
	data := g.Config.Stunts.Pipe
	p.OnBump = false
	contact := false
	for i := range p.Objects {
		o := &p.Objects[i]
		if o.Done {
			continue
		}
		pos := data.ObstaclePaths[o.Path][o.Frame]
		if p.ScreenY > float64(pos[1]-8) && p.ScreenY < float64(pos[1]+15) && p.ScreenX >= float64(pos[0]-10) && p.ScreenX <= float64(pos[0]+18) {
			contact = true
			switch o.Kind {
			case 3:
				o.Done = true
				p.Score += 15
				g.Cue("fx10")
			case 1, 2:
				g.pipeCrash()
			default:
				p.OnBump = true
				p.BumpTicks--
				if p.BumpTicks == 0 {
					g.pipeCrash()
				}
			}
			break
		}
	}
	if !contact {
		p.BumpTicks = 15
	}
}
func (g *Game) stepPipe(in Input) {
	p := &g.Pipe
	p.Tick++
	// The original HUD update follows movement, awards and crash decisions.
	defer g.pipeComment()
	// 0x55e0 selects terrain before movement. The same cached zone drives
	// this frame's pose and lip test, even if the index crosses a boundary.
	p.Zone = pipeZone(p.Index)
	if p.Finishing {
		g.animate(&p.FinishAnim, 0)
		f := p.FinishAnim.Visible
		p.Sprite = f[0]
		p.Mirror = f[1] != 0
		p.OffsetX = f[2]
		p.OffsetY = f[3]
		if p.FinishAnim.Done {
			g.Finish(p.Score+p.FinishBonus, "")
		}
		return
	}
	if !p.Recovering {
		if p.Time > 0 {
			p.Time--
		}
		if p.Time == 0 {
			p.Ending = true
		}
	}
	g.pipeBash()
	if p.StartFlagFrame < 18 && p.Tick%5 == 0 {
		p.StartFlagFrame++
	}
	if p.Starting {
		g.animate(&p.Anim, 0)
		f := p.Anim.Visible
		p.Sprite = f[0]
		p.OffsetX = f[2]
		p.OffsetY = f[3]
		p.Mirror = f[1] != 0
		if p.Anim.Done {
			p.Starting = false
			// The original start motion is the first scored entry (five
			// points), and shares its uniqueness record with floor entry 0.
			g.pipeAward()
		}
		return
	}

	if p.Recovering {
		if !(in.Confirm || in.ActionPressed) {
			return
		}
		score, time, used, unique := p.Score, p.Time, p.Used, p.Unique
		heading, turnDelay := p.Heading, p.TurnDelay
		g.initPipe()
		p = &g.Pipe
		p.Score = score
		p.Time = time
		p.Used = used
		p.Unique = unique
		p.Heading, p.TurnDelay = heading, turnDelay
		p.NoCrash = false
		p.Starting = false
		p.StartFlagFrame = 18

		return
	}
	if p.Ending && !p.Active {
		if p.Zone == 4 {
			bonus := 0
			if p.Unique > 0 {
				bonus = 100 + p.Unique*(p.Unique-1)/2*100
			}
			if p.NoCrash {
				bonus += 1000
			}
			p.FinishBonus = bonus
			p.Finishing = true
			ending := 0
			if p.Score+bonus > 15000 {
				ending = 1
			}
			if p.Score+bonus > 30000 {
				ending = 2
			}
			p.FaceFrame = ending * 2
			p.CommentPending = true
			g.showPipeComment(p.FaceFrame)
			p.FinishAnim = g.animation(g.Config.Stunts.Pipe.Finish[ending], 0)
			return
		}
		if p.Index < 108 {
			in = Input{Right: true}
		} else {
			in = Input{Left: true}
		}
	}
	if p.Active {
		g.pipeAnimate(in)
	} else {
		dir := stuntDirection(in)
		if !p.Ending {
			g.pipeStart(in)
		}
		if p.Active {
			g.pipeAnimate(in)
		} else {
			p.Direction = dir
			g.pipeMove(dir)
		}
	}
	g.pipeScenery()
}

func (g *Game) stepPipeWait() {
	p := &g.Pipe
	if p.StartFlagFrame == 14 {
		p.StartFlagFrame = 15
	}
	advanceMotion(&p.WaitAnim, g.Config.Stunts.Motions[g.Config.Stunts.Pipe.Wait], 0)
	f := p.WaitAnim.Visible
	p.Sprite = f[0]
	p.OffsetX = f[2]
	p.OffsetY = f[3]
	p.Mirror = f[1] != 0
	g.pipeComment()
}

// 0x5703 uses two independent timers: 61 calls until the displayed text
// is erased, and 120 neutral/pending calls before the next selection. The
// expression can change while a previous line is still on screen.
func (g *Game) showPipeComment(group int) {
	p := &g.Pipe
	p.Comment = group
	p.CommentIndex = g.Random(4)
	p.CommentVisible = true
	p.CommentTimer = 60
}
func (g *Game) pipeComment() {
	p := &g.Pipe
	if p.CommentVisible {
		p.CommentTimer--
		if p.CommentTimer < 0 {
			p.CommentVisible = false
			p.FaceFrame = 2
			p.FaceTimer = 120
		}
	}
	if !p.CommentPending && p.FaceFrame != 2 {
		return
	}
	p.FaceTimer--
	if p.FaceTimer != 0 {
		return
	}
	p.FaceTimer = 120
	group := p.FaceFrame
	if !p.CommentPending {
		if g.Waiting {
			group = 5
		} else {
			group = 1 + g.Random(2)
		}
	}
	p.CommentPending = false
	p.FaceFrame = group
	// Preserve the original recursive update at 0x576e, before drawing text.
	g.pipeComment()
	g.showPipeComment(group)
}

func (g *Game) pipeRails() {
	p := &g.Pipe
	delay := [3]int{4, 3, 2}[p.Energy>>8]
	p.RailDelay--
	if p.RailDelay < 0 {
		p.RailDelay = delay
		for i := range p.Rails {
			p.Rails[i]++
		}
		p.RailSpawn++
		if p.RailSpawn >= 6 {
			p.RailSpawn = 0
			p.Rails = append(p.Rails, 0)
		}
		live := p.Rails[:0]
		for _, f := range p.Rails {
			if f < 18 {
				live = append(live, f)
			}
		}
		p.Rails = live
	}
}
func (g *Game) pipeDecorative() {
	p := &g.Pipe
	if len(p.Decorative) < 2 && g.Random(256) <= 40 {
		value := g.Random(256)
		kind := 0
		switch {
		case value <= 40:
			kind = 3
		case value <= 80:
			kind = 2
		case value <= 190:
			kind = 1
		}
		p.Decorative = append(p.Decorative, PipeObject{Path: g.Random(2), Kind: kind})
	}
	if p.Active && (p.Trick == 32 || p.Trick == 34 || p.Trick == 96 || p.Trick == 98) {
		return
	}
	p.DecorDelay--
	if p.DecorDelay < 0 {
		p.DecorDelay = [3]int{4, 3, 2}[p.Energy>>8]
		live := p.Decorative[:0]
		for _, o := range p.Decorative {
			o.Frame++
			if o.Frame < 18 {
				live = append(live, o)
			}
		}
		p.Decorative = live
	}
}
func (g *Game) pipeBash() {
	p := &g.Pipe
	data := g.Config.Stunts.Pipe
	if !p.BashEnabled && (p.Zone == 1 || p.Zone == 7) {
		p.BashEnabled = true
		p.BashPlaying = true
		p.BashFrame = 0
		p.BashDelay = 5
		g.Cue("fx07")
	}
	if p.Zone == 4 || p.Trick == 39 || p.Trick == 103 {
		p.BashEnabled = false
	}
	if p.BashPlaying {
		p.BashDelay--
		if p.BashDelay < 0 {
			p.BashDelay = 5
			p.BashFrame++
			if p.BashFrame >= len(data.BashSprites) {
				p.BashPlaying = false
				p.BashFrame = 0
			}
		}
	}
}
