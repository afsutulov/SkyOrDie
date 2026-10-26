package game

type TubeData struct {
	Motions                                []Motion
	Actions                                [10][]int
	Ground                                 [12][4]int
	HeadingClass                           [12]int
	AttackOffsets                          [12][2]int
	Palette                                [16]int
	AIHeading                              [9]int
	AIGrid                                 [9]int
	ItemSprites                            [8]int
	RouteX, ItemRows, ItemCycle, RareCycle []int
	ItemX                                  [][4]int
}
type TubeTerrain struct {
	Surface, Obstacle int
	Ramp              bool
}

// The original 0x3842 routine decodes different bit fields from Downhill.
// In particular, snow banks are lateral impulses, not crash tiles.
func DecodeTubeTerrain(cell uint16) TubeTerrain {
	return TubeTerrain{Surface: int(cell>>10) & 3, Obstacle: int(cell>>12) & 7, Ramp: cell&0x8000 != 0}
}
func (g *Game) initTubeItems() {
	r := &g.Race
	data := g.Config.Tube
	// Original start coordinates: (50,56) and (240,56), plus the +13 hot point.
	r.Player.X = 63
	r.Player.Y = 56
	r.Player.Weapon = 1
	r.Player.WeaponKind = 3
	r.Rival.X = 253
	r.Rival.Y = 56
	r.Rival.Weapon = 1
	r.Rival.WeaponKind = 3
	r.Player.Air = 7 * 256
	r.Rival.Air = 7 * 256
	r.Player.Speed = 0
	r.Rival.Speed = 0
	r.Player.Tube = TubeRider{Heading: 6, Mode: 6}
	r.Rival.Tube = TubeRider{Heading: 6, Mode: 6}
	g.tubeBegin(&r.Player, 6)
	g.tubeBegin(&r.Rival, 6)
	g.tubePose(&r.Player)
	g.tubePose(&r.Rival)
	r.Items = nil
	normal, rare := 0, 0
	for i, row := range data.ItemRows {
		kind := 0
		if (i+1)%5 == 0 {
			kind = data.RareCycle[rare%len(data.RareCycle)]
			rare++
		} else {
			kind = data.ItemCycle[normal%len(data.ItemCycle)]
			normal++
		}
		r.Items = append(r.Items, TrackItem{X: float64(data.ItemX[i][g.Random(4)] + 13), Y: float64(row * 8), Kind: kind})
	}
}
func (g *Game) tubePickup(p *Racer, it *TrackItem) {
	it.Taken = true
	switch it.Kind {
	case 2, 3, 6:
		p.WeaponKind = it.Kind
		p.Weapon = 1
		p.Score += 25
	case 0:
		p.Air = min(7*256, p.Air+256)
		p.Score += 15
	case 1:
		p.Air = min(7*256, p.Air+512)
		p.Score += 15
	case 4, 5:
		p.Air = 0 // the two bad pickups empty the original air meter
	}
	p.Durability = p.Air * 100 / (7 * 256)
	g.Cue("fx09")
}
