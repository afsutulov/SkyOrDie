// Package render is a platform independent 320x200 software renderer.
// The desktop window and the screenshot command consume exactly these pixels.
package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"skyofdie/assets"
	"skyofdie/internal/game"
	"sort"
	"strings"
)

var Palette = [16]color.RGBA{{0, 0, 0, 255}, {0, 0, 170, 255}, {0, 170, 0, 255}, {0, 170, 170, 255}, {170, 0, 0, 255}, {170, 0, 170, 255}, {170, 85, 0, 255}, {170, 170, 170, 255}, {85, 85, 85, 255}, {85, 85, 255, 255}, {85, 255, 85, 255}, {85, 255, 255, 255}, {255, 85, 85, 255}, {255, 85, 255, 255}, {255, 255, 85, 255}, {255, 255, 255, 255}}

type Renderer struct {
	Data      *assets.Data
	Frame     *image.RGBA
	font      [96][64]bool
	snowFrame []byte
	snowTick  int
}

func New(a *assets.Data) *Renderer {
	r := &Renderer{Data: a, Frame: image.NewRGBA(image.Rect(0, 0, 320, 200))}
	f := a.Images["font"]
	for c := 0; c < 96; c++ {
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				_, _, _, alpha := f.At((c%16)*8+x, (c/16)*8+y).RGBA()
				r.font[c][y*8+x] = alpha != 0
			}
		}
	}
	return r
}
func (r *Renderer) fill(x, y, w, h, c int) {
	draw.Draw(r.Frame, image.Rect(x, y, x+w, y+h), image.NewUniform(Palette[c]), image.Point{}, draw.Src)
}
func (r *Renderer) bg(name string, x, y int) {
	if im := r.Data.Images[name]; im != nil {
		draw.Draw(r.Frame, r.Frame.Bounds(), im, image.Pt(x, y), draw.Src)
	}
}
func (r *Renderer) blit(im image.Image, x, y int) {
	if im == nil {
		return
	}
	b := im.Bounds()
	draw.Draw(r.Frame, image.Rect(x, y, x+b.Dx(), y+b.Dy()), im, b.Min, draw.Over)
}
func (r *Renderer) sprite(bank string, id, x, y int, mirror bool) {
	im := r.Data.Sprite(bank, id)
	b := im.Bounds()
	// The EGA blitter clears the low X bit and uses an inclusive bottom row.
	x &= ^1
	y++
	if !mirror {
		r.blit(im, x, y-b.Dy())
		return
	}
	for iy := 0; iy < b.Dy(); iy++ {
		for ix := 0; ix < b.Dx(); ix++ {
			c := color.RGBAModel.Convert(im.At(b.Min.X+ix, b.Min.Y+iy)).(color.RGBA)
			if c.A > 0 {
				r.Frame.SetRGBA(x+b.Dx()-1-ix, y-b.Dy()+iy, c)
			}
		}
	}
}
func (r *Renderer) text(s string, x, y, c int) { r.textScale(s, x, y, c, 1) }
func (r *Renderer) textScale(s string, x, y, c, scale int) {
	ox := x
	for _, ch := range strings.ToUpper(s) {
		if ch == '\n' {
			x = ox
			y += 8 * scale
			continue
		}
		if ch < 32 || ch > 127 {
			ch = '?'
		}
		for iy := 0; iy < 8; iy++ {
			for ix := 0; ix < 8; ix++ {
				if !r.font[ch-32][iy*8+ix] {
					continue
				}
				if scale == 1 {
					r.Frame.SetRGBA(x+ix, y+iy, Palette[c])
				} else {
					r.fill(x+ix*scale, y+iy*scale, scale, scale, c)
				}
			}
		}
		x += 8 * scale
	}
}
func (r *Renderer) line(x0, y0, x1, y1, c int) {
	dx := abs(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -abs(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		r.Frame.SetRGBA(x0, y0, Palette[c])
		if x0 == x1 && y0 == y1 {
			break
		}
		e := 2 * err
		if e >= dy {
			err += dy
			x0 += sx
		}
		if e <= dx {
			err += dx
			y0 += sy
		}
	}
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func (r *Renderer) Draw(g *game.Game) *image.RGBA {
	r.fill(0, 0, 320, 200, 0)
	switch g.Phase {
	case game.Travelling:
		r.fill(0, 0, 320, 200, 9)
		r.text(g.Config.Front.TravelMessages[g.Event], g.Config.Front.TravelX[g.Event], 100, 15)
	case game.Briefing:
		r.fill(0, 0, 320, 200, 9)
		r.text(g.Config.Front.NextMessages[g.Event], g.Config.Front.NextX[g.Event], 100, 15)
	case game.CompeteRequired:
		r.fill(0, 0, 320, 200, 9)
		r.text(g.Config.Front.CompeteRequired, 10, 100, 15)
	case game.Replay:
		r.fill(0, 0, 320, 200, 9)
		r.text(g.Config.Front.ReplayQuestion, 100, 80, 15)
		x := 184
		if g.ReplayYes {
			x = 132
		}
		r.text(g.Config.Front.ReplayMark, x, 100, 15)
	case game.Splash:
		r.bg("splash", 0, 0)
	case game.Shop:
		r.shop(g)
	case game.Resort:
		r.resort(g)
	case game.Signup:
		r.signup(g)
	case game.Records:
		r.records(g)
	case game.Playing, game.Results:
		r.play(g)
	case game.Standings:
		r.standings(g)
	case game.QuitPrompt:
		if g.Front.ReturnPhase == game.Shop {
			r.shop(g)
		} else {
			r.resort(g)
		}
		r.fill(70, 100, 176, 21, 9)
		r.fill(71, 101, 174, 19, 0)
		r.text(" EXIT TO DOS (Y/N)?", 78, 103, 15)
	}
	return r.Frame
}
func (r *Renderer) resort(g *game.Game) {
	r.bg("tsquare", 0, 0)
	for _, a := range g.Config.Front.MenuAnimations {
		if len(a.Frames) > 0 {
			r.blit(r.Data.Sprite("mnshp", a.Frames[(g.FrameTick/(a.Delay+1))%len(a.Frames)]), a.X, a.Y)
		}
	}
	f := g.Config.Front.ResortSprites[g.Front.Heading]
	r.sprite("mnfbm", f[0], int(g.MenuX)+f[2], int(g.MenuY)+f[3], f[1] != 0)
}
func (r *Renderer) shop(g *game.Game) {
	r.bg("shop", 0, 0)
	text := ""
	if !g.Front.ShopGreeted {
		text = g.Config.Front.Welcome
	}
	if g.Front.Speak >= 0 {
		lines := g.Config.Front.ShopSpeech[g.Front.Speak]
		if len(lines) > 0 {
			text = lines[g.Front.SpeechCycle%len(lines)]
		}
	}
	r.fill(24, 23, 138, 28, 15)
	r.text(text, 24, 23, 5)
	frame := g.Config.Front.ShopCursor[(g.FrameTick/16)%3]
	r.sprite("mnfbm", frame, g.Front.ShopX, g.Front.ShopY, false)
}
func (r *Renderer) signup(g *game.Game) {
	r.bg("signup", 0, 0)
	f := &g.Front
	rows := g.Config.Front.SignupRows
	devices := [4]string{"KEYPAD ONE", "KEYPAD TWO", "JOYSTICK A", "JOYSTICK B"}
	for i := 0; i < 2; i++ {
		c := 1
		if f.Choosing && f.SignupRow == i {
			c = 9
		}
		r.text(devices[g.Save.Controls[i]%4], 194, rows[i], c)
	}
	music := "ON"
	if g.Save.Muted {
		music = "OFF"
	}
	c := 1
	if f.Choosing && f.SignupRow == 2 {
		c = 9
	}
	r.text(music, 194, rows[2], c)
	for i, p := range g.Players {
		if i >= 6 {
			break
		}
		r.text(p.Name, 194, rows[i+4], 1)
	}
	r.text("X", 184, rows[f.SignupRow], 12)
	if f.Editing && g.FrameTick%16 < 8 {
		n := len(g.Players[f.SignupRow-4].Name)
		r.text("_", 194+n*8, rows[f.SignupRow], 1)
	}
}
func (r *Renderer) status(g *game.Game) string {
	if g.Mode == game.Practice {
		return "PRACTICE"
	}
	if g.ActivePlayer < len(g.Players) {
		return g.Players[g.ActivePlayer].Name
	}
	return ""
}
func (r *Renderer) hud(g *game.Game, score int) {
	r.fill(0, 0, 320, 32, 0)
	r.line(0, 0, 319, 0, 15)
	r.line(0, 0, 0, 31, 15)
	r.line(319, 0, 319, 31, 15)
	r.line(0, 31, 319, 31, 15)
	r.text(r.status(g), 6, 5, 12)
	labelColor := 10
	if g.Event != game.Aerials {
		labelColor = 13
	}
	r.text("SCORE", 6, 15, labelColor)
	r.text(fmt.Sprintf("%d", score), 6, 25, 14)
}
func (r *Renderer) records(g *game.Game) {
	r.bg("hiscore", 0, 0)
	for row := 0; row < 5; row++ {
		for place, entry := range g.Save.Records[row] {
			y := g.Config.Front.HighScoreY[row*3+place]
			r.text(entry.Name, 78, y, 9)
			value := fmt.Sprint(entry.Score)
			if row == 0 {
				value = fmt.Sprintf("%d.%d", entry.Score/10, entry.Score%10)
			}
			r.text(value, 168, y, 9)
		}
	}
}
func (r *Renderer) standings(g *game.Game) {
	r.fill(0, 0, 320, 200, 9)
	r.text(g.Config.Front.SummaryTitles[g.Event], 6, 5, 15)
	if g.Mode == game.Tournament {
		r.text("OVERALL", 240, 5, 15)
	}
	for i, p := range g.Players {
		y := 30 + i*8
		r.text(p.Name, 6, y, 15)
		score := fmt.Sprint(p.Scores[g.Event])
		if g.Event == game.Aerials {
			score = fmt.Sprintf("%d.%d", p.Scores[g.Event]/10, p.Scores[g.Event]%10)
		}
		r.text(score, 131, y, 15)
		if g.Mode == game.Tournament {
			r.text(fmt.Sprint(p.Points), 256, y, 15)
		}
	}
}
func (r *Renderer) play(g *game.Game) {
	switch g.Event {
	case game.Snowball:
		r.snow(g)
	case game.Downhill, game.Tube:
		r.race(g)
	case game.Aerials:
		r.air(g)
	case game.Halfpipe:
		r.pipe(g)
	}
}
func (r *Renderer) snow(g *game.Game) {
	s := &g.Snow
	if s.Shake > 0 && len(r.snowFrame) == len(r.Frame.Pix) && s.Tick >= r.snowTick {
		// 0x41d7 changes the display address before the new draw list is
		// rendered. It shakes the previous completed image, not new actors.
		copy(r.Frame.Pix, r.snowFrame)
		r.snowJolt(g)
		return
	}
	r.bg("snowfort", 0, 0)
	r.blit(r.Data.Sprite("sfshp", [4]int{2, 1, 3, 0}[s.Team]), 0, 40)
	r.blit(r.Data.Sprite("sfshp", [4]int{6, 5, 7, 4}[s.Team]), 272, 3)
	objs := []game.SnowObject{}
	for _, o := range s.Groups[s.Team] {
		if o.Active || o.Retired {
			objs = append(objs, o)
		}
	}
	for _, o := range s.Balls[s.Team] {
		if o.Active || o.Retired {
			objs = append(objs, o)
		}
	}
	if s.Special.Active {
		objs = append(objs, s.Special)
	}
	if s.Pickup.Active {
		objs = append(objs, s.Pickup)
	}
	sort.SliceStable(objs, func(i, j int) bool { return objs[i].Y < objs[j].Y })
	for _, o := range objs {
		r.sprite("sffbm", o.Sprite, o.X, o.Y, o.Mirror)
	}
	cursor := 124
	if s.Cooldown > 0 && s.Unlimited == 0 {
		cursor = 143
	}
	r.sprite("sffbm", cursor, s.X, s.Y, false)
	r.text(r.status(g), 6, 5, 12)
	if g.Waiting {
		r.text("HIT BUTTON", 6, 13, g.FrameTick%16)
	}
	r.text("LEVEL", 176, 5, 13)
	r.text(fmt.Sprint(s.Level), 192, 12, 14)
	r.text("SCORE", 6, 23, 13)
	r.text(fmt.Sprint(s.Score), 6, 30, 14)
	r.text("SNOWBALLS   KIDS", 80, 23, 13)
	r.text(fmt.Sprint(s.Ammo), 104, 30, 14)
	r.text(fmt.Sprint(s.Left), 184, 30, 14)
	if g.Mode == game.Practice {
		// Original 0x4099 selects SFSHP[8..23] from the 8.8 snow depth.
		r.blit(r.Data.Sprite("sfshp", 8+max(0, min(15, s.Meter>>8))), 224, 2)
	} else {
		r.text("TIME", 224, 23, 13)
		r.text(fmt.Sprint((s.Time+40)/41), 232, 30, 14)
	}
	for team, pos := range [4][2]int{{302, 20}, {292, 28}, {280, 20}, {292, 13}} {
		count := 0
		for _, o := range s.Groups[team] {
			if o.Active || o.Retired {
				count++
			}
		}
		r.sprite("sffbm", 127+min(5, count), pos[0], pos[1], false)
	}
	if s.Unlimited > 0 {
		r.text("UNLIMITED", 80, 12, 10)
	}
	if s.ClearDelay > 0 || s.ClearReady || g.Phase == game.Results {
		index := 0
		if g.Phase == game.Results {
			index = 2
			if g.Mode == game.Practice {
				index = 1
			}
		}
		r.fill(32, 100, 259, 16, 10)
		r.fill(33, 101, 257, 14, 9)
		r.text(g.Config.Front.SnowMessages[index], 52, 105, 14)
	}

	if s.Shake > 0 {
		r.snowJolt(g)
	} else {
		r.snowFrame = append(r.snowFrame[:0], r.Frame.Pix...)
		r.snowTick = s.Tick
	}
}

func (r *Renderer) snowJolt(g *game.Game) {
	// 0x41f1 waits one display step (two PIT interrupts) before each offset.
	phase := (12 - g.Snow.Shake%12) % 12
	if phase == 0 {
		return
	}
	offset := g.Config.Front.SnowShake[phase-1] * 8 * 4
	// EGA display addresses are linear: the right edge continues into the
	// next scanline instead of exposing a black vertical border.
	copy(r.Frame.Pix, r.Frame.Pix[offset:])
	for i := len(r.Frame.Pix) - offset; i < len(r.Frame.Pix); i += 4 {
		copy(r.Frame.Pix[i:i+4], []byte{0, 0, 0, 255})
	}
}

func (r *Renderer) race(g *game.Game) {
	a := &g.Race
	p := &a.Player
	name, bank := "dhcourse", "dhfbm"
	if g.Event == game.Tube {
		name = "tbcourse"
		bank = "tbfbm"
	}
	anchor := 70
	if g.Event == game.Downhill {
		anchor = 103
	}
	cam := max(0, min(int(p.Y)-anchor, 3825))
	if g.Event == game.Tube {
		cam = max(0, min(a.Camera, 3800))
	}
	r.bg(name, 0, cam-33)
	if g.Event == game.Downhill {
		d := &p.Downhill
		for _, t := range d.Trail {
			if t.Y >= cam-30 && t.Y <= cam+230 {
				r.sprite("dhfbm", t.Sprite, t.X, t.Y-cam+33, t.Mirror)
			}
		}
		if d.IntroMark {
			r.sprite("dhfbm", 99, 148, 92-cam+33, false)
		}
		if d.Animating && !d.Crashing && !g.Waiting {
			r.sprite("dhfbm", 147, int(p.X)-10, int(p.Y)-cam+33, false)
		}
	}
	if g.Event == game.Tube {
		for _, it := range a.Items {
			if it.Taken || it.Y < float64(cam)-30 || it.Y > float64(cam)+230 {
				continue
			}
			r.sprite("tbfbm", g.Config.Tube.ItemSprites[it.Kind], int(it.X)-13, int(it.Y)-cam+33, false)
		}
		q := a.Rival
		r.tubeSprite(g, q, int(q.Y)-cam+33, true)
	}
	frame := 9
	if g.Event == game.Downhill {
		frame = p.Sprite
	}
	sy := int(p.Y) - cam + 33
	if p.Invuln == 0 || g.Tick%4 < 2 {
		if g.Event == game.Downhill {
			r.sprite(bank, frame, int(p.X)-10+p.Downhill.OffsetX, sy+p.Downhill.OffsetY, p.Downhill.Mirror)
		} else {
			r.tubeSprite(g, *p, sy, false)
		}
	}
	r.hud(g, p.Score)
	if g.Event == game.Downhill && p.Downhill.Finishing {
		if a.Elapsed/26 < 120 {
			r.text("TIME BONUS", 120, 10, 10)
		}
		r.text(fmt.Sprintf("*%d*", p.Downhill.FinishBonus), 136, 20, 14)
	}
	if g.Event == game.Tube {
		r.fill(1, 14, 206, 17, 0)
		r.text("SCORE", 8, 15, 14)
		if !g.Waiting {
			r.text(fmt.Sprint(p.Score), 8, 24, 9)
		}
		if g.Waiting {
			x := 8
			if g.Race.FirstReady {
				x = 208
			}
			r.text("HIT BUTTON", x, 25, g.FrameTick%16)
		}
	}
	if g.Event == game.Downhill {
		r.blit(r.Data.Sprite("dhshp", 10), 240, 2)
		seconds := a.Elapsed / 26
		r.dhDigit(seconds%10, 280, 18)
		r.dhDigit(seconds/10%6, 272, 18)
		r.dhDigit(seconds/60%10, 256, 18)
	} else {
		name := "LESTER"
		if g.Mode != game.Practice && g.ActivePlayer+1 < len(g.Players) {
			name = g.Players[g.ActivePlayer+1].Name
		}
		r.text(name, 208, 5, 12)
		r.text("SCORE", 208, 15, 14)
		if !g.Waiting {
			r.text(fmt.Sprint(a.Rival.Score), 208, 24, 13)
			r.blit(r.Data.Sprite("tbshp", min(7, p.Air>>8)), 96, 18)
			r.blit(r.Data.Sprite("tbshp", min(7, a.Rival.Air>>8)), 288, 18)
			r.sprite("tbfbm", g.Config.Tube.ItemSprites[p.WeaponKind], 96, 17, false)
			r.sprite("tbfbm", g.Config.Tube.ItemSprites[a.Rival.WeaponKind], 288, 17, false)
		}
	}
}
func (r *Renderer) dhDigit(n, x, y int) { r.blit(r.Data.Sprite("dhshp", n), x, y) }

func (r *Renderer) air(g *game.Game) {
	a := &g.Air
	if a.Stage == 0 {
		r.bg("acrojump", 0, 0)
	} else {
		r.bg("ajcourse", 0, int(a.Camera)-33)
	}
	x, y := int(a.X)+a.OffsetX, int(a.Y)+a.OffsetY
	if a.Stage != 0 {
		y += 33
	}
	r.sprite("ajfbm", a.Sprite, x, y, a.Mirror)
	r.hud(g, 0)
	r.fill(6, 24, 64, 7, 0)
	if a.Judging || a.Judged {
		score := 0
		for i := 0; i < min(a.Judge, 5); i++ {
			score += int(a.Judges[i]*10 + .1)
		}
		r.text(fmt.Sprintf("%d.%d", score/10, score%10), 6, 25, 14)
	}
	r.text("JUMP:", 75, 15, 10)
	jump := a.Attempts + 1
	if a.Stage >= 2 {
		jump = a.Attempts
	}
	r.text(fmt.Sprint(jump), 91, 25, 14)
	for i := 0; i < 5; i++ {
		pose := a.JudgeFaces[i]
		r.blit(r.Data.Sprite("ajshp", g.Config.Front.AirJudgeFaces[i][pose]), 120+i*40, 1)
		if a.Judging && i < a.Judge || a.Judged {
			v := int(a.Judges[i]*10 + .1)
			r.smallScore(v, g.Config.Front.AirJudgeScoreX[i], g.Config.Front.AirJudgeScoreY[i])
		}
	}
}
func (r *Renderer) smallScore(v, x, y int) {
	text := fmt.Sprintf("%d.%d", v/10, v%10)
	// Judge placards use the original compact glyph layout.
	for _, c := range text {
		advance := 7
		if c == '.' {
			advance = 4
		}
		r.text(string(c), x, y, 1)
		x += advance
	}
}

func (r *Renderer) pipe(g *game.Game) {
	p := &g.Pipe
	data := &g.Config.Stunts.Pipe
	r.bg("sbtube", 0, 0)
	if p.StartFlagFrame < 18 {
		pos := data.ObstaclePaths[0][p.StartFlagFrame]
		r.sprite("sbfbm", data.StartFlag, pos[0], pos[1], false)
	}
	// Only the original foreground sprites move. The old synthetic horizontal
	// lines and invented flag gates have been removed.
	for _, f := range p.Rails {
		sprite := data.RailSprites[data.DecorFrame[f]]
		for side := 0; side < 2; side++ {
			pos := data.RailPaths[side][f]
			r.sprite("sbfbm", sprite, pos[0], pos[1], side == 1)
		}
	}
	for _, o := range p.Objects {
		if o.Done {
			continue
		}
		pos := data.ObstaclePaths[o.Path][o.Frame]
		r.sprite("sbfbm", data.ObstacleSprites[o.Kind][data.DecorFrame[o.Frame]], pos[0], pos[1], false)
	}
	for _, o := range p.Decorative {
		pos := data.DecorPaths[o.Path][o.Frame]
		r.sprite("sbfbm", data.DecorSprites[o.Kind][data.DecorFrame[o.Frame]], pos[0], pos[1], o.Path == 1)
	}
	if p.BashPlaying {
		x := 24 - data.BashOffsets[p.BashFrame]
		mirror := true
		if p.Index >= 108 {
			x = 280 + data.BashOffsets[p.BashFrame]
			mirror = false
		}
		r.sprite("sbfbm", data.BashSprites[p.BashFrame], x, 113, mirror)
	}
	r.sprite("sbfbm", p.Sprite, int(p.ScreenX)+p.OffsetX, int(p.ScreenY)+p.OffsetY, p.Mirror)
	score := p.Score
	if g.Phase == game.Results {
		score += p.FinishBonus
	}
	r.hud(g, score)
	if g.Phase == game.Results {
		r.text("TRICK BONUS", 120, 40, 13)
		r.text(fmt.Sprintf("*%d*", p.FinishBonus), 144, 50, 14)
	}
	r.blit(r.Data.Sprite("sbshp", 8), 80, 2)
	r.blit(r.Data.Sprite("sbshp", g.Config.Front.HalfpipeLevels[min(2, p.Energy>>8)]), 88, 4)
	seconds := (120*26 - p.Time) / 26
	for i, x := range []int{120, 112, 96} {
		n := []int{seconds % 10, seconds / 10 % 6, seconds / 60 % 10}[i]
		r.blit(r.Data.Sprite("sbshp", g.Config.Front.ClockSprites[n]), x, 18)
	}
	r.blit(r.Data.Sprite("sbshp", g.Config.Front.HalfpipeFaces[p.FaceFrame]), 152, 1)
	comments := g.Config.Front.HalfpipeCommentary[p.Comment]
	if p.CommentVisible && len(comments) > 0 {
		r.text(comments[p.CommentIndex%len(comments)], 200, 5, 10)
	}
}

func (r *Renderer) tubeSprite(g *game.Game, p game.Racer, y int, rival bool) {
	if p.Tube.Catching && g.Tick%2 != 0 {
		return
	}
	im := r.Data.Sprite("tbfbm", p.Sprite)
	b := im.Bounds()
	x := int(p.X) - 13 + p.Tube.OffsetX
	y += p.Tube.OffsetY - b.Dy()
	if !rival {
		r.sprite("tbfbm", p.Sprite, x, y+b.Dy(), p.Tube.Mirror)
		return
	}
	x &= ^1
	y++
	for iy := 0; iy < b.Dy(); iy++ {
		for ix := 0; ix < b.Dx(); ix++ {
			c := color.RGBAModel.Convert(im.At(b.Min.X+ix, b.Min.Y+iy)).(color.RGBA)
			if c.A == 0 {
				continue
			}
			for n, pc := range Palette {
				if c == pc {
					c = Palette[g.Config.Tube.Palette[n]]
					break
				}
			}
			dx := ix
			if p.Tube.Mirror {
				dx = b.Dx() - 1 - ix
			}
			r.Frame.SetRGBA(x+dx, y+iy, c)
		}
	}
}
