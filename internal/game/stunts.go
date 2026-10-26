package game

// StuntData contains converted original animation and movement records. No
// machine code or interpreted instructions are included in these records.
type StuntData struct {
	Motions []Motion
	Air     AirData
	Pipe    PipeData
}
type AirData struct {
	Ramp, Landing     [][3]int
	Actions           [16]int
	Idle              [3]int
	Scores            [16]int
	Success           [4]int
	Failure           [5]int
	SuccessThresholds [4]int
	FailureThresholds [5]int
}
type PipeData struct {
	Finish                   [3]int
	DecorPaths               [2][18][2]int
	DecorSprites             [4][7]int
	BashOffsets, BashSprites []int
	Wait, Start, StartFlag   int
	X, Y                     [109]int
	Ground                   [9][7][4]int
	Actions                  [108]int
	Scores, Unique           [108]int
	Crashes                  [8]int
	BankTicks                [3]int
	DecorFrame               [18]int
	RailSprites              [7]int
	RailPaths                [2][18][2]int
	ObstaclePaths            [3][18][2]int
	ObstacleSprites          [4][7]int
}

// Animation is the native Go version of the original 0x64EC frame scheduler.
// Holding, release, loop boundaries, and the landing-safe pose come from data.
type Animation struct {
	Motion, Frame, Delay, Remaining, Loops, Direction int
	Holding, Expired, Done                            bool
	Visible                                           [4]int
}

func (g *Game) animation(id, dir int) Animation {
	m := g.Config.Stunts.Motions[id]
	return Animation{Motion: id, Direction: dir, Remaining: m.HoldTicks}
}
func (g *Game) animate(a *Animation, dir int) {
	advanceMotion(a, g.Config.Stunts.Motions[a.Motion], dir)
}
func advanceMotion(a *Animation, m Motion, dir int) {
	if a.Done {
		return
	}
	a.Holding = false
	if m.Kind != 0 && m.Kind != 4 {
		if m.Kind >= 2 && a.Remaining == 255 {
			a.Holding = true
		} else if a.Direction == dir && a.Direction >= 0 {
			// The original counter is a byte: zero decrements to 255.
			a.Remaining = (a.Remaining - 1) & 255
			if a.Remaining > 0 {
				a.Holding = true
			} else {
				a.Expired = true
				a.Direction = -1
			}
		} else {
			a.Direction = -1
		}
	}
	a.Visible = m.Frames[min(a.Frame, len(m.Frames)-1)]
	a.Delay--
	if a.Delay >= 0 {
		return
	}
	a.Delay = m.Delay
	switch m.Kind {
	case 1:
		if a.Holding && a.Frame == m.HoldFrame {
			return
		}
	case 2, 3, 4:
		if a.Holding || m.Kind == 4 {
			a.Frame++
			if a.Frame > m.LoopEnd {
				a.Loops++
				a.Frame = m.LoopStart
			}
			return
		}
		if m.Kind == 2 && a.Direction < 0 {
			return
		}
	}
	a.Frame++
	if a.Frame >= len(m.Frames) {
		a.Done = true
		a.Frame = len(m.Frames) - 1
	}
}
func stuntDirection(in Input) int {
	return inputDirection(in)
}
func safePose(m Motion, frame, tolerance int) bool {
	n := m.LoopEnd - m.LoopStart + 1
	if n <= 0 {
		return true
	}
	d := absInt(frame - m.SafeFrame)
	if frame >= m.LoopStart && frame <= m.LoopEnd {
		d = min(d, n-d)
	}
	return d <= tolerance
}
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
