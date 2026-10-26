package game

// TubeRider is the 8.8 movement and twelve-pose control state used by the
// original tube event. Heading rotates with button one; button two attacks.
type TubeRider struct {
	Catching                                 bool
	Direction                                int
	Heading, TurnDelay, VX, VY, Mode, Motion int
	OffsetX, OffsetY                         int
	Mirror                                   bool
	Anim                                     Animation
}

func (g *Game) tubeBegin(p *Racer, action int) {
	t := &p.Tube
	ids := g.Config.Tube.Actions[action]
	v := 0
	if action == 1 {
		v = t.Heading
	} else if action < 3 {
		v = g.Config.Tube.HeadingClass[t.Heading]
	}
	t.Motion = ids[v]
	m := g.Config.Tube.Motions[t.Motion]
	t.Anim = Animation{Remaining: m.HoldTicks, Direction: inputDirection(Input{})}
}
func (g *Game) stepTubeWait() {
	for _, p := range []*Racer{&g.Race.Player, &g.Race.Rival} {
		g.tubePose(p)
	}
}
func (g *Game) tubePose(p *Racer) {
	t := &p.Tube
	f := g.Config.Tube.Ground[t.Heading]
	if t.Mode != 0 && t.Mode != 8 {
		m := g.Config.Tube.Motions[t.Motion]
		advanceMotion(&t.Anim, m, 0)
		f = t.Anim.Visible
		if t.Anim.Done {
			switch t.Mode {
			case 1:
				t.VY = 512
			case 7:
				t.Mode = 8
				t.Heading = 6
			case 5:
				g.recoverTubePosition(p)
				g.Race.Frozen = false
			case 4:
				p.Air = 5 * 256
				p.Durability = 71
				q := &g.Race.Player
				if p == q {
					q = &g.Race.Rival
				}
				if p.Y < q.Y {
					p.Tube.VY = 512
				} else {
					q.Tube.VY = 512
				}
				g.Race.Frozen = false
			}
			if t.Mode != 8 {
				t.Mode = 0
			}
		}
	}
	p.Sprite = f[0]
	t.Mirror = f[1] != 0
	t.OffsetX = f[2]
	t.OffsetY = f[3]
	// The right rider uses a palette-remapped copy of the same artwork.
	if p.X < 23 {
		p.X = 23
		t.VX = -(t.VX >> 8) * 256
	}
	if p.X >= 303 {
		p.X = 303
		t.VX = -(t.VX >> 8) * 256
	}
	p.Speed = float64(t.VY >> 8)
	p.VX = float64(t.VX >> 8)
}
func (g *Game) tubeMove(p, q *Racer, in Input, human bool) {
	t := &p.Tube
	dir := inputDirection(in)
	buttons := 0
	if in.Action {
		buttons = 1
	}
	if in.Trick && !in.Action {
		buttons = 2
	}
	if !human {
		dir, buttons = g.tubeAI(p, q)
	}
	terrain := DecodeTubeTerrain(g.Config.Courses[1].Cell(p.X, p.Y))
	if !g.Race.Frozen && t.Mode != 7 && t.Mode != 8 && t.Mode != 1 {
		switch {
		case terrain.Obstacle == 1:
			t.VX = -1024
			p.BankTimer = 14
			p.BankDirection = -1
			g.Cue("fx13")
		case terrain.Obstacle == 2:
			t.VX = 1024
			p.BankTimer = 14
			p.BankDirection = 1
			g.Cue("fx13")
		case terrain.Obstacle == 3 || terrain.Obstacle == 4:
			t.Mode = 5
			action := 4
			if terrain.Obstacle == 4 {
				action = 5
			}
			g.tubeBegin(p, action)
			g.freezeTube()
			g.Cue("fx04")
		case t.Mode == 3:
		case p.Air == 0:
			t.Mode = 4
			g.tubeBegin(p, 3)
			g.freezeTube()
			g.Cue("fx15")
		case terrain.Surface == 1 && (t.VX>>8 != 0 || t.VY>>8 != 0):
			t.Mode = 3
			g.tubeBegin(p, 0)
			g.Cue("fx14")
		case terrain.Ramp:
			t.Mode = 1
			g.tubeBegin(p, 2)
			t.VX = 0
			t.VY = 768
			dir = 0
			buttons = 0
			g.Cue("fx16")
		case p.Y >= 3960:
			t.Mode = 7
			action := 9
			if !g.Race.Won {
				action = 8
				g.Race.Won = true
				p.Score += 1000
			}
			g.tubeBegin(p, action)
			t.VX = 0
			t.VY = 0
			dir = 0
			buttons = 0
			g.freezeTube()
		}
	}
	if g.Race.Frozen || t.Mode == 7 || t.Mode == 8 {
		dir = 0
		buttons = 0
	}
	if t.Mode == 3 {
		buttons = 0
	}
	// Rotation and translation are distinct in the original input routine.
	if buttons != 1 && int(p.Y)-g.Race.Camera+33 >= 60 {
		switch dir {
		case 7, 8, 1:
			t.VY -= 20
			if t.VY>>8 < 0 {
				t.VY &= 255
			}
		case 3, 4, 5:
			t.VY += 20
			if t.VY>>8 > 2 {
				t.VY = (t.VY & 255) | 512
			}
		}
		switch dir {
		case 5, 6, 7:
			t.VX -= 80
			if t.VX>>8 < -4 {
				t.VX = (t.VX & 255) - 1024
			}
		case 1, 2, 3:
			t.VX += 80
			if t.VX>>8 > 4 {
				t.VX = (t.VX & 255) | 1024
			}
		}
	}
	if buttons == 1 && (dir == 2 || dir == 6) || p.BankTimer > 0 || terrain.Surface == 2 {
		spin := dir
		if p.BankTimer > 0 {
			p.BankTimer--
			spin = 6
			if p.BankDirection < 0 {
				spin = 2
			}
		}
		if terrain.Surface == 2 {
			spin = 2
		}
		t.TurnDelay--
		if t.TurnDelay < 0 {
			t.TurnDelay = 2
			if spin == 2 {
				t.Heading = (t.Heading + 1) % 12
			} else {
				t.Heading = (t.Heading + 11) % 12
			}
		}
	}
	if buttons == 2 && t.Mode != 2 {
		t.Mode = 2
		g.tubeBegin(p, 1)
		g.Cue("fx20")
		offset := g.Config.Tube.AttackOffsets[t.Heading]
		distance := absInt(int(p.X)-13+offset[0]-int(q.X)) + absInt(int(p.Y)+offset[1]-int(q.Y)+10)
		if distance <= 19 {
			damage, points := 1, 50
			if p.WeaponKind == 2 {
				damage = 2
				points = 125
			}
			if p.WeaponKind == 6 {
				damage = 3
				points = 225
			}
			q.Air = max(0, q.Air-damage*256)
			q.Durability = q.Air * 100 / (7 * 256)
			p.Score += points
		}
	}
	screenY := int(p.Y) - g.Race.Camera + 33
	if screenY < 60 && !t.Catching {
		t.Catching = true
		t.VY = (t.VY & 255) | 1024
		dir = 0
	}
	if t.Catching && p.Y >= q.Y-31 {
		t.Catching = false
		t.VY = (t.VY & 255) | (q.Tube.VY & ^255)
		q.Score += 100
	}
	t.Direction = dir
	p.X += float64(t.VX >> 8)
	dy := max(0, t.VY>>8)
	extra := 0
	if t.Catching {
		extra = 4
	} else if screenY < 120 || dir >= 3 && dir <= 5 {
		extra = 1
	}
	if dy != 0 && screenY+extra <= 180 {
		dy += extra
		if t.Catching && int(p.Y)/8 >= 331 && int(p.Y)/8 <= 381 {
			p.X = float64(g.Config.Tube.RouteX[int(p.Y)/8] + 13)
		}
	}
	p.Y += float64(dy)
	p.Speed = float64(max(0, t.VY>>8))
	p.VX = float64(t.VX >> 8)
	if !g.Race.Frozen {
		for i := range g.Race.Items {
			it := &g.Race.Items[i]
			if !it.Taken && int(it.Y)/8 >= int(p.Y)/8 && int(it.Y)/8 <= int(p.Y)/8+2 && p.X >= it.X-22 && p.X <= it.X+12 {
				g.tubePickup(p, it)
				break
			}
		}
	}
}
func (g *Game) freezeTube() {
	g.Race.Frozen = true
	for _, p := range []*Racer{&g.Race.Player, &g.Race.Rival} {
		p.Tube.VX = 0
		p.Tube.VY = 0
		p.Tube.Catching = false
		p.Speed = 0
		p.VX = 0
	}
}
func (g *Game) stepTube(in Input) {
	r := &g.Race
	previous := tubePositions{r.Player.X, r.Player.Y, r.Rival.X, r.Rival.Y, r.Camera}
	r.Elapsed++
	second := Input{}
	human2 := g.Mode != Practice && g.ActivePlayer+1 < len(g.Players)
	if in.Second != nil {
		second = *in.Second
	}
	g.tubeMove(&r.Player, &r.Rival, in, true)
	g.tubeMove(&r.Rival, &r.Player, second, human2)
	// The faster rider determines scrolling. The slower rider falls behind.
	if !r.Frozen {
		r.Camera += int(max(r.Player.Speed, r.Rival.Speed))
	}
	g.tubeCollision(previous)
	g.tubePose(&r.Player)
	g.tubePose(&r.Rival)
	if r.Player.Tube.Mode == 8 || r.Rival.Tube.Mode == 8 {
		g.Finish(r.Player.Score, "")
	}
}

type tubePositions struct {
	playerX, playerY, rivalX, rivalY float64
	camera                           int
}

// Original 0x2fad: exchange full 8.8 velocities and restore the complete
// previous positions/scroll, except for its two lower-edge stop branches.
func (g *Game) tubeCollision(previous tubePositions) bool {
	r := &g.Race
	p, q := &r.Player, &r.Rival
	if p.Tube.Catching || q.Tube.Catching || absInt(int(p.X-q.X))+absInt(int(p.Y-q.Y)) > 23 {
		return false
	}
	horizontal := func(direction int) bool { return direction == 2 || direction == 6 }
	exchange := true
	if p.Tube.VY>>8 != 0 || q.Tube.VY>>8 != 0 {
		if int(p.Y)-r.Camera+33 >= 180 && (!horizontal(p.Tube.Direction) || !horizontal(q.Tube.Direction)) {
			q.Tube.VY = 0
			exchange = false
		} else if int(p.Y)-r.Camera+33 < 180 && int(q.Y)-r.Camera+33 >= 180 && (!horizontal(p.Tube.Direction) || !horizontal(q.Tube.Direction)) {
			p.Tube.VY = 0
			exchange = false
		}
	}
	if exchange {
		p.Tube.VX, q.Tube.VX = q.Tube.VX, p.Tube.VX
		p.Tube.VY, q.Tube.VY = q.Tube.VY, p.Tube.VY
		p.X, p.Y, q.X, q.Y = previous.playerX, previous.playerY, previous.rivalX, previous.rivalY
		r.Camera = previous.camera
	}
	g.Cue("fx13")
	return true
}

func (g *Game) tubeAttackDistance(p, q *Racer) int {
	offset := g.Config.Tube.AttackOffsets[p.Tube.Heading]
	return absInt(int(p.X)-13+offset[0]-int(q.X)) + absInt(int(p.Y)+offset[1]-int(q.Y)+10)
}
func (g *Game) tubeAI(p, q *Racer) (int, int) {
	r := &g.Race
	x := 0
	targetX, baseX := int(q.X), int(p.X)-13
	for x < 2 {
		baseX += 9
		if targetX < baseX {
			break
		}
		x++
	}
	y := 2
	targetY, baseY := int(q.Y)-10, int(p.Y)
	for y > 0 {
		baseY -= 7
		if targetY > baseY {
			break
		}
		y--
	}
	dir := g.Config.Tube.AIGrid[y*3+x]
	if !r.AIAggressive {
		dir = 4
	}
	target := g.Config.Tube.AIHeading[dir]
	delta := (target - p.Tube.Heading + 12) % 12
	if delta != 0 {
		if delta > 6 {
			return 6, 1
		}
		return 2, 1
	}
	r.AIAttackDelay--
	if r.AIAttackDelay < 0 {
		r.AIAttackDelay = 0
		if g.tubeAttackDistance(p, q) <= 19 {
			r.AIAttackDelay = 25
			return 0, 2
		}
	}
	if p.Score >= q.Score {
		r.AIDelay--
		if r.AIDelay < 0 {
			r.AIDelay = 40
			r.AIAggressive = true
		}
	}
	if q.Tube.VY>>8 == 0 || !r.AIAggressive {
		return 4, 0
	}
	return dir, 0
}
func (g *Game) tubeReady(in Input) {
	r := &g.Race
	if !r.FirstReady && (in.Action || in.ActionPressed || in.Confirm) {
		r.FirstReady = true
		g.tubeBegin(&r.Player, 7)
		r.Player.Tube.Mode = 6
	}
	if r.FirstReady && !r.SecondReady {
		human := g.Mode != Practice && g.ActivePlayer+1 < len(g.Players)
		if !human || in.Second != nil && (in.Second.Action || in.Second.ActionPressed) {
			r.SecondReady = true
			g.tubeBegin(&r.Rival, 7)
			r.Rival.Tube.Mode = 6
			g.Waiting = false
			g.FrameTick = 0
			r.AIDelay = 40
		}
	}
	g.stepTubeWait()
}

func (g *Game) recoverTubePosition(p *Racer) {
	q := &g.Race.Player
	if p == q {
		q = &g.Race.Rival
	}
	c := g.Config.Courses[1]
	x := max(13, q.X-33)
	for x >= 13 {
		if DecodeTubeTerrain(c.Cell(x, p.Y)).Obstacle == 0 {
			p.X = x
			return
		}
		x -= 2
	}
	x = min(303, q.X+33)
	for x <= 303 {
		if DecodeTubeTerrain(c.Cell(x, p.Y)).Obstacle == 0 {
			p.X = x
			return
		}
		x += 2
	}
	p.X = 303
}
