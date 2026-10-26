package game

type ScreenAnimation struct {
	X, Y, Delay int
	Frames      []int
}
type Record struct {
	Name  string
	Score int
}
type FrontData struct {
	TravelMessages, NextMessages   [5]string
	TravelX, NextX                 [5]int
	CompeteRequired                string
	SnowMessages                   [3]string
	ReplayQuestion, ReplayMark     string
	SummaryTitles                  [5]string
	HighScoreY                     [15]int
	InitialRecords                 [5][3]Record
	ShopRects                      [15][4]int
	ShopSpeech                     [16][]string
	Welcome                        string
	ShopCursor                     [3]int
	SignupRows                     [10]int
	MenuAnimations                 [2]ScreenAnimation
	ResortPortals                  [7][4]int
	ResortBlocks                   [9][4]int
	ResortDX, ResortDY             [17]int
	ResortSprites                  [17][4]int
	AirJudgeFaces                  [5][5]int
	AirJudgeScoreX, AirJudgeScoreY [5]int
	AirWait                        Motion
	SnowShake                      [12]int
	SnowDX, SnowDY                 [3][9]int
	Trail                          [3][7][4]int
	HalfpipeFaces                  [7]int
	HalfpipeLevels                 [3]int
	HalfpipeCommentary             [6][]string
	ClockSprites, DHClockSprites   [10]int
}
type FrontState struct {
	ShopX, ShopY, Hover, Speak, SpeechCycle int
	Heading, TurnDelay                      int
	SignupRow, Choice                       int
	Choosing, Editing                       bool
	ReturnPhase                             Phase
	Moving                                  bool
	ShopGreeted                             bool
}

func contains(r [4]int, x, y int) bool { return x >= r[0] && x <= r[2] && y >= r[1] && y <= r[3] }
func (g *Game) shop(in Input) {
	dx, dy := direction(in)
	f := &g.Front
	f.ShopX = max(0, min(290, f.ShopX+int(dx)*2))
	f.ShopY = max(25, min(199, f.ShopY+int(dy)*2))
	if in.Pointer {
		f.ShopX = max(0, min(290, in.PointerX-10))
		f.ShopY = max(25, min(199, in.PointerY+15))
	}
	hover := -1
	for i, r := range g.Config.Front.ShopRects {
		if contains(r, f.ShopX+10, f.ShopY-15) {
			hover = i
			break
		}
	}
	if hover != f.Hover {
		f.Hover = hover
		f.Speak = hover
		if hover == 0 && !g.registered() {
			f.Speak = 15
		}
		f.ShopGreeted = true
		f.SpeechCycle++
	}
	if !(in.ActionPressed || in.Confirm) {
		return
	}
	switch hover {
	case 0:
		if g.registered() {
			g.Mode = Single
			g.enterResort()
			g.Cue("fx09")
		}
	case 1:
		g.Mode = Practice
		g.enterResort()
		g.Cue("fx09")
	case 2:
		g.Phase = Signup
		f.SignupRow = 0
		f.Choosing = false
		f.Editing = false
		g.Cue("fx09")
	case 14:
		g.Phase = Records
		g.Cue("fx09")
	}
}
func (g *Game) registered() bool {
	for _, p := range g.Players {
		if p.Name != "" {
			return true
		}
	}
	return false
}
func (g *Game) enterResort() {
	g.Phase = Resort
	if g.Mode == Tournament {
		g.Mode = Single
	}
	g.MenuX = 150
	g.MenuY = 140
	g.Front.Heading = 8
	g.Front.TurnDelay = 3
	g.Front.Moving = false
	g.FrameTick = 0
}
func (g *Game) resort(in Input) {
	f := &g.Front
	dir := inputDirection(in)
	if dir == 2 || dir == 6 {
		f.TurnDelay--
		if f.TurnDelay < 0 {
			f.TurnDelay = 3
			if dir == 2 {
				f.Heading++
			} else {
				f.Heading--
			}
			if f.Heading < 1 {
				f.Heading = 16
			}
			if f.Heading > 16 {
				f.Heading = 1
			}
		}
	}
	if in.Action {
		f.Moving = true
	} else if !in.Trick {
		f.Moving = false
	}
	if !f.Moving {
		return
	}
	oldX, oldY := g.MenuX, g.MenuY
	x := oldX + float64(g.Config.Front.ResortDX[f.Heading])
	y := oldY + float64(g.Config.Front.ResortDY[f.Heading])
	for i := len(g.Config.Front.ResortPortals) - 1; i >= 0; i-- {
		r := g.Config.Front.ResortPortals[i]
		if contains(r, int(x)+10, int(y)-5) {
			if i == 0 {
				g.enterShop()
				return
			}
			if i == 1 {
				if g.Mode == Practice || !g.registered() {
					g.Phase = CompeteRequired
					g.FrameTick = 0
					return
				}
				g.Mode = Tournament
				g.Event = TournamentOrder[0]
				for i := range g.Players {
					g.Players[i].Points = 0
					g.Players[i].Scores = [5]int{}
					g.Players[i].Done = [5]bool{}
				}
			} else {
				g.Event = [5]Event{Snowball, Aerials, Tube, Downhill, Halfpipe}[i-2]
			}
			g.ActivePlayer = 0
			g.BeginEvent()
			return
		}
	}
	for i := len(g.Config.Front.ResortBlocks) - 1; i >= 0; i-- {
		r := g.Config.Front.ResortBlocks[i]
		if contains(r, int(x)+10, int(y)-5) {
			// 0x332 checks the old point against every block before sliding.
			oldBlocked := false
			for _, oldBlock := range g.Config.Front.ResortBlocks {
				oldBlocked = oldBlocked || contains(oldBlock, int(oldX)+10, int(oldY)-5)
			}
			if oldBlocked {
				x, y = oldX, oldY
				break
			}
			if !contains(r, int(x)+10, int(oldY)-5) {
				y = oldY
			} else if !contains(r, int(oldX)+10, int(y)-5) {
				x = oldX
			} else {
				x = oldX
				y = oldY
			}
			break
		}
	}
	// Keep the complete skier on the native framebuffer. The DOS renderer
	// could wrap an out-of-range coordinate into unrelated video memory.
	g.MenuX = clamp(x, 0, 288)
	g.MenuY = clamp(y, 23, 193)
}
func (g *Game) originalSignup(in Input) {
	f := &g.Front
	if in.Pointer && in.ActionPressed && !f.Editing {
		row := -1
		for i, y := range g.Config.Front.SignupRows {
			if in.PointerX >= 60 && in.PointerX < 315 && in.PointerY >= y-2 && in.PointerY <= y+9 {
				row = i
				break
			}
		}
		last := min(9, 4+len(g.Players))
		if len(g.Players) == 1 && g.Players[0].Name == "" {
			last = 4
		}
		if row < 0 || row > last {
			return
		}
		if f.Choosing && row == f.SignupRow {
			in.RightPressed = true
			in.ActionPressed, in.Confirm = false, false
		} else {
			f.Choosing = false
			f.SignupRow = row
		}
	}
	if f.Editing {
		p := &g.Players[f.SignupRow-4]
		if in.Backspace && len(p.Name) > 0 {
			p.Name = p.Name[:len(p.Name)-1]
		}
		for _, c := range in.Text {
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			if c >= 32 && c < 127 && len(p.Name) < 10 {
				p.Name += string(c)
			}
		}
		if in.Confirm {
			f.Editing = false
			g.Save.Names = g.names()
			if p.Name == "" {
				index := f.SignupRow - 4
				g.Players = append(g.Players[:index], g.Players[index+1:]...)
			} else if f.SignupRow < 9 {
				f.SignupRow++
			}
			g.Save.Names = g.names()
		}
		return
	}
	if f.Choosing {
		if in.LeftPressed || in.RightPressed {
			delta := 1
			if in.LeftPressed {
				delta = -1
			}
			if f.SignupRow < 2 {
				g.Save.Controls[f.SignupRow] = (g.Save.Controls[f.SignupRow] + delta + 4) % 4
			} else {
				g.Save.Muted = !g.Save.Muted
			}
		}
		if in.Confirm {
			f.Choosing = false
		}
		return
	}
	if in.UpPressed {
		f.SignupRow = max(0, f.SignupRow-1)
	}
	if in.DownPressed || in.Tab {
		n := len(g.Players)
		if n == 1 && g.Players[0].Name == "" {
			n = 0
		}
		f.SignupRow = min(min(9, 4+n), f.SignupRow+1)
	}
	if !(in.Confirm || in.ActionPressed) {
		return
	}
	switch {
	case f.SignupRow < 3:
		f.Choosing = true
	case f.SignupRow == 3:
		g.enterShop()
	default:
		for len(g.Players) <= f.SignupRow-4 {
			g.Players = append(g.Players, Player{})
		}
		f.Editing = true
		g.Players[f.SignupRow-4].Name = ""
	}
}

func (g *Game) enterShop() {
	g.Phase = Shop
	g.FrameTick = 0
	g.Front.ShopX, g.Front.ShopY = 90, 100
	g.Front.Hover, g.Front.Speak = -1, -1
	g.Front.ShopGreeted = false
}
