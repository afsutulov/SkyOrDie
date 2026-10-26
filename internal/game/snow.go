package game

// Animation and spawn records were recovered from SKIEGA. Table provenance:
// tuunanet/rea-ski-or-die-reconstruction (MIT), see licenses/THIRD_PARTY.md.
type Behavior struct {
	Kind, LoopStart, LoopEnd int
	Delay                    int
	Frames                   [][4]int
}
type Spawn struct {
	X, Y, Behavior int
	Script         []int
}
type RoamRecord struct{ Kind, Threshold, Behavior, X, Y, Limit, DX int }
type SnowData struct {
	Roam            [6]RoamRecord
	PickupPositions [10][2]int
	PickupSprites   [3]int
	Behaviors       []Behavior
	Spawns          []Spawn
}
type SnowObject struct {
	Retired                                         bool
	Active, Dead, Mirror                            bool
	X, Y, BX, BY                                    int
	Behavior, Frame, Delay, Sprite, Kind, ScriptPos int
	Script                                          []int
}
type SnowCue struct {
	After int
	Name  string
}
type SnowState struct {
	PendingCues                                                                                  []SnowCue
	PendingFinish                                                                                bool
	PendingScore                                                                                 int
	PendingMessage                                                                               string
	ClearReady                                                                                   bool
	RoamDX, RoamLimit, PickupCursor                                                              int
	Shake, ShakeHold                                                                             int
	RefillTimer                                                                                  int
	X, Y, Accel, Ammo, Score, Level, Left, Credits, Time, Team, Tick, Cooldown, Unlimited, Meter int
	Groups                                                                                       [4][5]SnowObject
	Balls                                                                                        [4][3]SnowObject
	SpawnTimers                                                                                  [4]int
	Cursors                                                                                      [4]int
	Special                                                                                      SnowObject
	Pickup                                                                                       SnowObject
	SpecialTimer, PickupTimer, ClearDelay, Flash                                                 int
}

func (g *Game) initSnow() { g.Snow = SnowState{Ammo: 200, Level: 1}; g.snowLevel() }
func (g *Game) snowLevel() {
	s := &g.Snow
	s.X = 150
	s.Y = 150
	s.Accel = 0
	s.Left = 50 + 25*(s.Level-1)
	s.Credits = s.Left
	s.Time = 59 * 41
	s.RefillTimer = 50
	s.Unlimited = 0
	s.Team = 3
	s.Tick = 0
	s.Cooldown = 0
	s.Groups = [4][5]SnowObject{}
	s.Balls = [4][3]SnowObject{}
	s.SpawnTimers = [4]int{1, 1, 1, 1}
	s.SpecialTimer = 400
	s.PickupTimer = 600
	s.PendingCues = nil
	s.PendingFinish = false
}
func (g *Game) snowFrame(o *SnowObject, visible bool) bool {
	if !o.Active {
		return false
	}
	bs := g.Config.SnowTables.Behaviors
	if o.Behavior < 0 || o.Behavior >= len(bs) {
		o.Active = false
		return true
	}
	b := bs[o.Behavior]
	f := b.Frames[min(o.Frame, len(b.Frames)-1)]
	o.Sprite = f[0]
	o.Mirror = f[1] != 0
	o.X = max(0, min(319, o.BX+f[2]))
	if visible {
		o.X &= ^1
	}
	o.Y = max(0, min(199, o.BY+f[3]))
	o.Delay--
	if o.Delay < 0 {
		o.Delay = b.Delay
		o.Frame++
		if b.Kind == 4 && o.Frame > b.LoopEnd {
			o.Frame = b.LoopStart
		}
		return o.Frame >= len(b.Frames)
	}
	return false
}
func (g *Game) snowSet(o *SnowObject, b int) {
	o.Behavior = b
	o.Frame = 0
	o.Delay = 0
}
func (g *Game) snowHit(o *SnowObject, px, py int) bool {
	if !o.Active || o.Dead {
		return false
	}
	if o.Sprite < 0 || o.Sprite >= len(g.Config.SpriteSizes) {
		return false
	}
	sz := g.Config.SpriteSizes[o.Sprite]
	return px >= o.X && px <= o.X+sz[0] && py >= o.Y-sz[1] && py <= o.Y
}
func (g *Game) stepSnow(in Input) {
	s := &g.Snow
	s.Special.Retired = false
	if s.Shake > 0 {
		s.Shake--
		pending := s.PendingCues[:0]
		for _, cue := range s.PendingCues {
			cue.After--
			if cue.After == 0 {
				g.Cue(cue.Name)
			} else {
				pending = append(pending, cue)
			}
		}
		s.PendingCues = pending
		if s.Shake == 0 && s.PendingFinish {
			s.PendingFinish = false
			g.Finish(s.PendingScore, s.PendingMessage)
		}
		return
	}
	if s.ClearReady {
		if in.AnyKey || in.Action || in.Confirm {
			s.Level++
			s.ClearReady = false
			g.snowLevel()
		}
		return
	}
	if s.ClearDelay > 0 {
		s.ClearDelay--
		if s.ClearDelay == 0 {
			s.ClearReady = true
		}
		return
	}

	s.Tick++
	if g.Mode != Practice {
		s.Time--
	}
	if s.Flash > 0 {
		s.Flash--
	}
	if s.Unlimited > 0 {
		s.Unlimited--
	}
	dir := inputDirection(in)

	dx, dy := 0, 0
	if dir == 0 {
		s.Accel = 0
	}
	if dir != 0 {
		speed := min(s.Accel>>8, 2)
		dx = g.Config.Front.SnowDX[speed][dir]
		dy = g.Config.Front.SnowDY[speed][dir]
	}

	if dx != 0 || dy != 0 {
		s.X += dx
		s.Y += dy
	}
	s.Accel += 40
	if s.Accel>>8 > 2 {
		s.Accel = (s.Accel & 255) | 512
	}
	s.X = max(0, min(303, s.X))
	s.Y = max(53, min(199, s.Y))
	if in.Pointer {
		s.X = max(0, min(303, in.PointerX-9))
		s.Y = max(53, min(199, in.PointerY+8))
		s.Accel = 0
	}
	// Original 0x3b56–0x3b7b updates every kid pool before any ball pool.
	// Random draws therefore have the same ordering across all four sides.
	for _, group := range [4]int{3, 1, 0, 2} {
		g.snowKids(group)
	}
	for _, group := range [4]int{3, 1, 0, 2} {
		for i := range s.Balls[group] {
			b := &s.Balls[group][i]
			b.Retired = false
			if b.Active && g.snowFrame(b, group == s.Team) {
				b.Active = false
				b.Retired = true
				if group == s.Team {
					g.snowCue("fx02")
				}
				// Original 0x455a checks all completed trajectories, including unseen sides.
				if g.Random(256) < 8 {
					g.snowCue("fx11")
					s.Shake += 12
				}
			}
		}
	}

	if !s.Special.Active {
		s.SpecialTimer--
		if s.SpecialTimer == 0 {
			choice := 5
			value := g.Random(256)
			for i, r := range g.Config.SnowTables.Roam {
				if value >= r.Threshold {
					choice = i
					break
				}
			}
			r := g.Config.SnowTables.Roam[choice]
			s.Special = SnowObject{Active: true, BX: r.X, BY: r.Y, X: r.X, Y: r.Y, Kind: r.Kind}
			s.RoamDX = r.DX
			s.RoamLimit = r.Limit
			g.snowSet(&s.Special, r.Behavior)
			s.SpecialTimer = 400
		}
	}
	if s.Special.Active {
		o := &s.Special
		if !o.Dead {
			o.BX += s.RoamDX
			if o.BX < 4 || o.BX > s.RoamLimit {
				o.Active = false
			}
		}
		if o.Active {
			done := g.snowFrame(o, true)
			if !o.Dead {
				o.BX = o.X
			} else if done {
				o.Active = false
				o.Retired = true
			}
		}
	}
	s.PickupTimer--
	if s.PickupTimer <= 0 {
		if s.Pickup.Active {
			s.Pickup.Active = false
			s.PickupTimer = 600
		} else {
			value := g.Random(256)
			kind := 2
			if value >= 125 {
				kind = 0
			} else if value >= 50 {
				kind = 1
			}
			g.spawnSnowPickup(kind)
		}
	}

	if s.Cooldown > 0 {
		s.Cooldown--
	} else if in.Action || in.ActionPressed {
		s.Cooldown = 4
		if s.Unlimited > 0 {
			s.Cooldown = 0
		}
		if s.Unlimited == 0 {
			s.Ammo = max(0, s.Ammo-1)
		}
		g.snowCue("fx01")
		px, py := s.X+9, s.Y-8
		for i := range s.Groups[s.Team] {
			o := &s.Groups[s.Team][i]
			if !g.snowHit(o, px, py) {
				continue
			}
			s.Score += [4]int{100, 75, 50, 25}[o.Kind-5]
			b := 18 + o.Kind - 5
			o.Dead = true
			o.BX = o.X
			o.BY = o.Y
			g.snowSet(o, b)
			s.Flash = 3
			if o.Kind == 8 {
				g.snowCue("fx03")
			}
		}
		if g.snowHit(&s.Special, px, py) {
			o := &s.Special
			death := 40
			switch o.Kind {
			case 2:
				s.Score += 50
				death = 31
			case 3:
				s.Score += 30
				death = 30
			case 4:
				s.Ammo = max(0, s.Ammo-25)
				death = 32
			case 10:
				s.Score += 30
				death = 33
			case 11:
				s.Score += 30
				death = 34
			case 12:
				s.Score += 30
			}
			o.Dead = true
			o.BX = o.X
			o.BY = o.Y
			g.snowSet(o, death)
			g.snowCue("fx10")
		}

		if g.snowHit(&s.Pickup, px, py) {
			switch s.Pickup.Kind {
			case 0:
				s.Ammo += 100
			case 1:
				s.Meter = max(0, s.Meter-256)
			case 2:
				s.Unlimited = 400
			}
			s.Pickup.Active = false
			g.snowCue("fx10")
		}
	}
	if s.Ammo == 0 && s.Cooldown == 0 {
		s.RefillTimer--
		if s.RefillTimer < 0 {
			s.Ammo = 10
			g.spawnSnowPickup(0)
			s.RefillTimer = 50
		}
	}
	g.snowView(in)
	if s.Left <= 0 {
		s.Score += 1000 + s.Ammo
		s.Meter = max(0, s.Meter-768)
		s.ClearDelay = 150
		g.snowCue("fx10")
		return
	}
	if s.Time <= 0 || (g.Mode == Practice && s.Meter >= 15*256) {
		message := "YOU'RE OUTTA TIME, SLICK!"
		if g.Mode == Practice {
			message = "YOU'VE BEEN SNOWED, FROSTY!"
		}
		s.Score += s.Ammo
		if s.Shake > 0 {
			s.PendingFinish, s.PendingScore, s.PendingMessage = true, s.Score, message
		} else {
			g.Finish(s.Score, message)
		}
	}
}

// The original 0x4501 blocks inside 0x41d7. Effects that follow a jolt
// must wait for that jolt; emitting them together cuts off the PC Speaker.
func (g *Game) snowCue(name string) {
	if g.Snow.Shake == 0 {
		g.Cue(name)
		return
	}
	g.Snow.PendingCues = append(g.Snow.PendingCues, SnowCue{After: g.Snow.Shake, Name: name})
}

func (g *Game) spawnSnowPickup(kind int) {
	s := &g.Snow
	pos := g.Config.SnowTables.PickupPositions[s.PickupCursor]
	s.PickupCursor = (s.PickupCursor + 1) % 10
	s.Pickup = SnowObject{Active: true, X: pos[0], Y: pos[1], Kind: kind, Sprite: g.Config.SnowTables.PickupSprites[kind]}
	// 0x463f falls through into 0x4699: the spawn frame consumes a tick.
	s.PickupTimer = 599
}

// snowKids mirrors 0x427d/0x4338: advance existing actors, then create one
// new actor. A new arrival is drawn once before its first bounds check.
func (g *Game) snowKids(group int) {
	s := &g.Snow
	// 0x42b4 tests the stored count from before 0x4338, even if one of
	// those five actors finishes during this update.
	previousCount := 0
	for _, o := range s.Groups[group] {
		if o.Active {
			previousCount++
		}
	}
	for slot := range s.Groups[group] {
		o := &s.Groups[group][slot]
		o.Retired = false
		if !o.Active {
			continue
		}
		done := g.snowFrame(o, group == s.Team)
		limit := 291
		if o.Kind == 5 {
			limit = 305
		} else if o.Kind == 6 {
			limit = 299
		}
		if o.X < 1 || o.X >= limit {
			o.Active = false
			o.Retired = true // 0x64ec already queued this last frame for drawing
			s.Credits++
			continue
		}
		if !o.Dead && o.Behavior >= 6 && o.Behavior <= 8 && o.Frame == 5 && o.Delay == g.Config.SnowTables.Behaviors[o.Behavior].Delay {
			// 0x4495 increments the practice meter even if all three ball slots are busy.
			s.Meter += 6
			for i := range s.Balls[group] {
				ball := &s.Balls[group][i]
				if ball.Active {
					continue
				}
				*ball = SnowObject{Active: true, BX: o.X, BY: o.Y, X: o.X, Y: o.Y}
				behavior := 22
				if o.Kind == 6 {
					behavior = 23
				} else if o.Kind == 7 {
					behavior = 24
				}
				g.snowSet(ball, behavior)
				break
			}
		}
		if !done {
			continue
		}
		if o.Dead {
			o.Active = false
			o.Retired = true
			s.Left--
			continue
		}
		if o.ScriptPos >= len(o.Script) || o.Script[o.ScriptPos] == 255 {
			o.Active = false
			o.Retired = true
			s.Credits++
			continue
		}
		old := o.Behavior
		// 0x43e8 always carries the last displayed X into the next behavior.
		o.BX = o.X
		if old < 6 {
			o.BX += [6]int{2, -4, 4, -6, 6, -8}[old]
		}
		g.snowSet(o, o.Script[o.ScriptPos])
		o.ScriptPos++
	}
	if previousCount == len(s.Groups[group]) {
		return
	} // full pools do not decrement the timer
	s.SpawnTimers[group] = (s.SpawnTimers[group] - 1) & 255
	if s.SpawnTimers[group] != 0 || s.Credits == 0 {
		return
	}
	s.Credits--
	s.SpawnTimers[group] = 90
	for slot := range s.Groups[group] {
		if s.Groups[group][slot].Active {
			continue
		}
		value := g.Random(256)
		kind := 8
		if value >= 190 {
			kind = 5
		} else if value >= 127 {
			kind = 6
		} else if value >= 50 {
			kind = 7
		}
		// CL at 0x431b is the type-table index, not the free actor slot.
		index := kind - 5
		spawn := g.Config.SnowTables.Spawns[index*10+s.Cursors[index]]
		s.Cursors[index] = (s.Cursors[index] + 1) % 10
		o := SnowObject{Active: true, X: spawn.X, Y: spawn.Y, BX: spawn.X, BY: spawn.Y, Kind: kind, Script: spawn.Script, ScriptPos: 1}
		g.snowSet(&o, spawn.Behavior)
		g.snowFrame(&o, group == s.Team)
		s.Groups[group][slot] = o
		return
	}
}

func (g *Game) snowView(in Input) {
	s := &g.Snow
	team := ((s.Team+in.ViewDelta)%4 + 4) % 4
	if in.Trick {
		switch inputDirection(in) {
		case 2:
			team = 0
		case 4:
			team = 1
		case 6:
			team = 2
		case 8:
			team = 3
		}
	}
	if team != s.Team {
		s.Team = team
		// Original 0x41c9 clears view-local roaming/pickup objects on a switch.
		s.Special.Active = false
		s.Pickup.Active = false
	}
}
