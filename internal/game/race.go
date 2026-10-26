package game

type Racer struct {
	Air                                                                                    int
	Tube                                                                                   TubeRider
	Downhill                                                                               DownhillState
	BankTimer, BankDirection, WeaponKind, Deflated                                         int
	X, Y, VX, Speed, Z, VZ                                                                 float64
	Crash, Invuln, AirTicks, Combo, Score, LastTerrain, Durability, Weapon, Attack, Sprite int
}
type TrackItem struct {
	X, Y  float64
	Kind  int
	Taken bool
}
type RaceState struct {
	FirstReady, SecondReady bool
	AIAggressive            bool
	AIDelay, AIAttackDelay  int
	Camera                  int
	Frozen                  bool
	Player, Rival           Racer
	Items                   []TrackItem
	Elapsed                 int
	FinishY                 float64
	Won                     bool
}

func (g *Game) initRace() {
	g.Race = RaceState{Player: Racer{Durability: 100}, Rival: Racer{Durability: 100}}
	if g.Event == Downhill {
		g.Race.FinishY = 3928
		g.initDownhill()
	} else {
		g.initTubeItems()
	}
}
