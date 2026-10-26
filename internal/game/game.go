// Package game implements the entire game simulation. It has no OS, audio,
// graphics, emulator, or wall-clock dependencies. Step follows the original PIT cadence.
package game

import (
	"math"
	"math/bits"
)

type Event int

const (
	Snowball Event = iota
	Downhill
	Aerials
	Tube
	Halfpipe
)

// Tournament order is the call sequence at original CS:0x445..0x451.
var TournamentOrder = [5]Event{Halfpipe, Snowball, Downhill, Aerials, Tube}

var EventNames = [5]string{"Snowball Blast", "Downhill Blitz", "Acro Aerials", "Innertube Thrash", "Snowboard Halfpipe"}

type Phase int

const (
	Resort Phase = iota
	Briefing
	Playing
	Results
	Standings
	Records
	Signup
	Splash
	Shop
	QuitPrompt
	Replay
	Travelling
	CompeteRequired
)

type Mode int

const (
	Practice Mode = iota
	Single
	Tournament
)

var ModeNames = [3]string{"PRACTICE", "SINGLE EVENT", "TOURNAMENT"}

type Input struct {
	// Optional desktop pointer events, expressed in the original framebuffer.
	// Zero-valued inputs retain the original keyboard/gamepad behavior.
	Pointer                                             bool
	PointerX, PointerY, ViewDelta                       int
	Second                                              *Input
	AnyKey                                              bool
	Left, Right, Up, Down, Action, Trick                bool
	ActionPressed, TrickPressed, Confirm, Escape, Pause bool
	LeftPressed, RightPressed, UpPressed, DownPressed   bool

	Tab, Backspace bool

	Text string
}
type Player struct {
	Points int
	Name   string
	Scores [5]int
	Done   [5]bool
}
type Save struct {
	Records   [5][3]Record `json:"records"`
	Controls  [2]int       `json:"controls"`
	Names     []string     `json:"names"`
	Best      [5]int       `json:"best"`
	BestNames [5]string    `json:"best_names"`
	Muted     bool         `json:"muted"`
}
type Course struct {
	Width, Height int
	Cells         []uint16
}

func (c Course) Cell(x, y float64) uint16 {
	ix, iy := int(x)/8, int(y)/8
	if x < 0 || y < 0 || ix >= c.Width || iy >= c.Height {
		return 0xffff
	}
	return c.Cells[iy*c.Width+ix]
}

type Config struct {
	Front       FrontData
	Stunts      StuntData
	Courses     [3]Course
	SnowTables  SnowData
	Downhill    DownhillData
	Tube        TubeData
	SpriteSizes [][2]int
}
type Game struct {
	NextEventPending bool
	ReplayYes        bool
	Front            FrontState
	Waiting          bool
	FrameTick        int
	Phase            Phase
	Event            Event
	Mode             Mode
	Players          []Player
	ActivePlayer     int

	Tick   int
	Paused bool
	Quit   bool
	Save   Save

	Sound        []string
	Snow         SnowState
	Race         RaceState
	Air          AirState
	Pipe         PipeState
	Result       string
	ResultScore  int
	MenuX, MenuY float64

	Config *Config
	rng    uint32
}

func New(cfg *Config, save Save) *Game {
	g := &Game{Config: cfg, Save: save, rng: 0x534b4931, MenuX: 32, MenuY: 156, Mode: Practice, Phase: Splash}
	names := save.Names
	if len(names) == 0 {
		names = []string{""}
	}
	for _, n := range names {
		if len(g.Players) < 6 {
			g.Players = append(g.Players, Player{Name: n})
		}
	}
	if save.Controls == [2]int{} && len(save.Names) == 0 {
		g.Save.Controls = [2]int{0, 1}
	}
	if g.Save.Records == [5][3]Record{} {
		g.Save.Records = cfg.Front.InitialRecords
	}
	g.Front = FrontState{ShopX: 90, ShopY: 100, Hover: -1, Speak: -1, Heading: 8, TurnDelay: 3}
	return g
}
func (g *Game) Random(n int) int {
	if n <= 0 {
		return 0
	}
	x := bits.RotateLeft16(uint16(g.rng)+0x9248, -3)
	g.rng = uint32(x)
	value := int(x & 255)
	if n >= 256 {
		return value
	}
	step := 255 / n
	index := 0
	limit := step
	for index < n-1 && value > limit {
		limit += step
		index++
	}
	return index
}
func (g *Game) Cue(s string) { g.Sound = append(g.Sound, s) }
func (g *Game) Step(in Input) {
	g.Sound = g.Sound[:0]
	g.Tick++
	g.FrameTick++
	if in.Escape && g.Phase != Travelling && g.Phase != Briefing && g.Phase != CompeteRequired && g.Phase != Standings {
		if g.Phase == Splash {
			g.enterShop()
			return
		}
		if g.Phase == Replay {
			g.enterResort()
			return
		}
		if (g.Phase == Playing || g.Phase == Results) && g.Mode != Practice {
			// Esc skips the remaining entrants of this event. The unfinished
			// attempt is not recorded (0x11c6, 0x3ccb, 0x4bee, 0x2b3d).
			g.completeEvent()
			return
		} else if g.Phase == Playing && g.Mode == Practice {
			g.Phase = Replay
			g.ReplayYes = true
			return
		}
		if g.Phase == Signup || g.Phase == Records {
			g.enterShop()
			g.Front.Choosing = false
			g.Front.Editing = false
			return
		}

		if g.Phase == Resort || g.Phase == Shop {
			g.Front.ReturnPhase = g.Phase
			g.Phase = QuitPrompt
			return
		}
		if g.Phase == QuitPrompt {
			g.Phase = g.Front.ReturnPhase
			return
		}
		g.enterResort()
		g.Paused = false
		return
	}

	switch g.Phase {
	case Splash:
		if in.AnyKey || in.ActionPressed || in.Confirm || in.Text != "" {
			g.enterShop()
		}
	case Shop:
		g.shop(in)
	case QuitPrompt:
		if in.Text == "y" || in.Text == "Y" {
			g.Quit = true
		}
		if in.Text == "n" || in.Text == "N" {
			g.Phase = g.Front.ReturnPhase
		}
	case Resort:
		g.resort(in)
	case Travelling:
		g.StartEvent()
	case Briefing:
		if g.FrameTick >= 50 {
			g.StartEvent()
		}
	case CompeteRequired:
		if g.FrameTick >= 50 {
			g.enterResort()
		}
	case Signup:
		g.originalSignup(in)
	case Records:
		if in.Confirm {
			g.enterShop()
		}
	case Results:
		if in.Confirm {
			if g.Mode == Practice {
				g.Phase = Replay
				g.ReplayYes = true
			} else {
				g.nextTurn()
			}
		}
	case Replay:
		if in.Left || in.LeftPressed {
			g.ReplayYes = true
		}
		if in.Right || in.RightPressed {
			g.ReplayYes = false
		}
		if in.Confirm || in.ActionPressed {
			if g.ReplayYes {
				g.StartEvent()
			} else {
				g.enterResort()
			}
		}

	case Standings:
		if in.Confirm {
			if g.NextEventPending {
				g.NextEventPending = false
				g.Event, _ = nextTournamentEvent(g.Event)
				g.BeginEvent()
			} else {
				g.enterResort()
			}
		}
	case Playing:
		if g.Waiting {
			if g.Event == Tube {
				g.tubeReady(in)
				return
			}
			if g.Event == Downhill {
				g.stepDownhillIntro(in)
				return
			}
			if in.ActionPressed || in.Confirm {
				g.Waiting = false
				if g.Event == Halfpipe {
					// The DOS clock's zero initial subcounter advances the
					// displayed second on the first active frame.
					g.Pipe.Time = 119*26 + 1
				}
				if g.Event == Aerials {
					g.Cue("fx17")
				}
				g.FrameTick = 0
			} else {
				if g.Event == Aerials {
					g.stepAirWait()
				}
				if g.Event == Tube {
					g.stepTubeWait()
				}
				if g.Event == Halfpipe {
					g.stepPipeWait()
				}
				return
			}
		}
		if in.Pause {
			g.Paused = !g.Paused
		}
		if g.Paused {
			return
		}
		switch g.Event {
		case Snowball:
			g.stepSnow(in)
		case Downhill:
			g.stepDownhill(in)
		case Tube:
			g.stepTube(in)
		case Aerials:
			g.stepAir(in)
		case Halfpipe:
			g.stepPipe(in)
		}
	}
}

// BeginEvent restores the original loading card. Resources are already decoded
// in the native build, so the desktop presents the card before initializing the
// event; unlike NEXT cards, the DOS loader has no fixed timer here.
func (g *Game) BeginEvent() {
	g.Phase = Travelling
	g.FrameTick = 0
	g.NextEventPending = false
	g.ActivePlayer = 0
	for i := range g.Players {
		g.Players[i].Scores[g.Event] = 0
		g.Players[i].Done[g.Event] = false
	}
}
func nextTournamentEvent(event Event) (Event, bool) {
	for i, e := range TournamentOrder {
		if e == event && i+1 < len(TournamentOrder) {
			return TournamentOrder[i+1], true
		}
	}
	return event, false
}

func (g *Game) StartEvent() {
	g.Phase = Playing
	g.Paused = false

	g.Result = ""
	g.ResultScore = 0
	switch g.Event {
	case Snowball:
		g.initSnow()
	case Downhill, Tube:
		g.initRace()
	case Aerials:
		g.initAir(0, 0, [3]int{})
	case Halfpipe:
		g.initPipe()
	}
	g.Waiting = true
	g.FrameTick = 0
}
func (g *Game) Finish(score int, message string) {
	g.ResultScore = max(0, score)
	g.Result = message
	g.Phase = Results
	if g.Mode != Practice {
		count := 1
		if g.Event == Tube && g.ActivePlayer+1 < len(g.Players) {
			count = 2
		}
		for i := 0; i < count; i++ {
			value := g.ResultScore
			if i == 1 {
				value = g.Race.Rival.Score
			}
			p := &g.Players[g.ActivePlayer+i]
			p.Scores[g.Event] = value
			p.Done[g.Event] = true
			slot := [5]int{3, 1, 0, 4, 2}[g.Event]
			record := Record{p.Name, value}
			for place := 0; place < 3; place++ {
				if record.Score > g.Save.Records[slot][place].Score {
					record, g.Save.Records[slot][place] = g.Save.Records[slot][place], record
				}
			}
			if value > g.Save.Best[g.Event] {
				g.Save.Best[g.Event] = value
				g.Save.BestNames[g.Event] = p.Name
				g.Cue("fx10")
			}
		}
	}
}

func (g *Game) nextTurn() {
	if g.Mode == Practice {
		g.enterResort()
		return
	}
	if g.Event == Tube {
		g.ActivePlayer += 2
	} else {
		g.ActivePlayer++
	}
	if g.ActivePlayer < len(g.Players) {
		g.Phase = Briefing
		g.FrameTick = 0
		return
	}
	g.completeEvent()
}

func (g *Game) completeEvent() {
	g.ActivePlayer = 0
	g.Paused = false
	g.NextEventPending = false
	if g.Mode == Tournament {
		scores := make([]int, len(g.Players))
		for i, p := range g.Players {
			scores[i] = p.Scores[g.Event]
		}
		for award := 5; award >= 1; award -= 2 {
			best := -1
			for i, score := range scores {
				if score > 0 && (best < 0 || score > scores[best]) {
					best = i
				}
			}
			if best < 0 {
				break
			}
			g.Players[best].Points += award
			scores[best] = 0
		}
		_, g.NextEventPending = nextTournamentEvent(g.Event)
	}
	g.Phase = Standings
}
func (g *Game) names() []string {
	n := []string{}
	for _, p := range g.Players {
		n = append(n, p.Name)
	}
	return n
}
func direction(in Input) (float64, float64) {
	x, y := 0., 0.
	if in.Left {
		x--
	}
	if in.Right {
		x++
	}
	if in.Up {
		y--
	}
	if in.Down {
		y++
	}
	return x, y
}
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// The original PIT runs at 1193182/16384 Hz. Disciplines wait for 2 or 3
// timer interrupts; the shop uses one. Snowball's CRTC jolt keeps the
// same two-interrupt wait as its gameplay loop (0x41f1 -> 0x61a4).
func (g *Game) TickDivisor() int {
	if g.Phase == Shop || g.Phase == Signup || g.Phase == Splash {
		return 1
	}
	if g.Phase == Playing || g.Phase == Briefing {
		if g.Event == Downhill || g.Event == Tube {
			return 3
		}
	}
	return 2
}
