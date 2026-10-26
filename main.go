package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"skyofdie/assets"
	"skyofdie/internal/controls"
	"skyofdie/internal/game"
	"skyofdie/internal/render"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var version = "0.7.3"

type application struct {
	model           *game.Game
	renderer        *render.Renderer
	sound           *soundSystem
	canvas          *ebiten.Image
	savePath        string
	lastSave        []byte
	controls        controls.Adapter
	clock           int64
	pending         game.Input
	travelPresented bool
	screenshot      bool
	runTicks        int
	limitTicks      int
}

func readSave(path string) game.Save {
	s := game.Save{Controls: [2]int{0, 1}}
	b, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(b, &s); err != nil {
			log.Printf("Ignoring invalid settings: %v", err)
		}
	}
	if len(s.Names) > 6 {
		s.Names = s.Names[:6]
	}
	for i, n := range s.Names {
		if len(n) > 10 {
			s.Names[i] = n[:10]
		}
	}
	return s
}
func (a *application) save() {
	b, err := json.MarshalIndent(a.model.Save, "", "  ")
	if err != nil || bytes.Equal(b, a.lastSave) {
		return
	}
	if err = os.MkdirAll(filepath.Dir(a.savePath), 0755); err == nil {
		err = os.WriteFile(a.savePath+".tmp", b, 0600)
		if err == nil {
			err = os.Rename(a.savePath+".tmp", a.savePath)
		}
	}
	if err != nil {
		log.Printf("Cannot save settings: %v", err)

	} else {
		a.lastSave = append(a.lastSave[:0], b...)
	}
}
func down(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}
func pressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func (a *application) Update() error {
	a.runTicks++
	if a.limitTicks > 0 && a.runTicks >= a.limitTicks {
		return ebiten.Termination
	}
	if down(ebiten.KeyControlLeft, ebiten.KeyControlRight) && pressed(ebiten.KeyM) {
		a.model.Save.Muted = !a.model.Save.Muted
	}
	if pressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if pressed(ebiten.KeyF12) {
		a.screenshot = true
	}

	mode := ebiten.CursorModeVisible
	if a.model.Phase == game.Shop || a.model.Phase == game.Playing && a.model.Event == game.Snowball && !a.model.Paused {
		mode = ebiten.CursorModeHidden
	}
	ebiten.SetCursorMode(mode)
	in := a.input()
	if !ebiten.IsFocused() {
		a.pending = game.Input{}
		return nil
	}
	controls.Latch(&in, a.pending)
	a.pending = in
	// The DOS loading card covers disk I/O. Native resources are preloaded,
	// but the card must still reach the display before event initialization.
	if a.model.Phase == game.Travelling && !a.travelPresented {
		a.pending = game.Input{}
		return nil
	}
	if a.model.Phase != game.Travelling {
		a.travelPresented = false
	}
	a.clock += 1193182
	for a.clock >= int64(240*16384*a.model.TickDivisor()) {
		a.clock -= int64(240 * 16384 * a.model.TickDivisor())
		wasTravelling := a.model.Phase == game.Travelling
		a.model.Step(a.pending)
		if wasTravelling {
			a.travelPresented = false
		}
		a.pending = game.Input{}
		a.sound.update(a.model)
		if a.model.Phase == game.Travelling {
			break
		}
	}
	a.save()
	if a.model.Quit {
		return ebiten.Termination
	}
	return nil
}
func (a *application) Draw(screen *ebiten.Image) {
	frame := a.renderer.Draw(a.model)
	if a.model.Phase == game.Travelling {
		a.travelPresented = true
	}
	a.canvas.WritePixels(frame.Pix)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1, 1.2)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(a.canvas, op)
}
func (a *application) Layout(_, _ int) (int, int) { return 320, 240 }
func main() {
	adlib := flag.Bool("adlib", false, "use the original AdLib sound instead of PC Speaker")
	noSound := flag.Bool("nosound", false, "disable all audio")
	fullscreen := flag.Bool("fullscreen", false, "start in fullscreen")
	scale := flag.Int("scale", 3, "initial window scale (1..6)")
	event := flag.Int("event", 0, "start directly in event 1..5")
	showVersion := flag.Bool("version", false, "print version and exit")
	ticks := flag.Int("ticks", 0, "exit after this many ticks (smoke testing)")
	flag.Parse()
	if *showVersion {
		fmt.Println("SkyOfDie", version)
		return
	}
	data, err := assets.Load()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		cfg = "."
	}
	savePath := filepath.Join(cfg, "SkyOfDie", "settings.json")
	save := readSave(savePath)
	model := game.New(&data.Config, save)
	if *event >= 1 && *event <= 5 {
		model.Event = game.Event(*event - 1)
		model.StartEvent()
	}
	sound := newSound(*noSound || save.Muted)
	sound.disabled = *noSound
	if *adlib {
		sound.card = "adlib_"
	}
	defer sound.close()
	app := &application{model: model, renderer: render.New(data), sound: sound, canvas: ebiten.NewImage(320, 200), savePath: savePath, limitTicks: *ticks}
	ebiten.SetTPS(240)
	ebiten.SetWindowTitle("SkyOfDie")
	ebiten.SetWindowSize(320*max(1, min(6, *scale)), 240*max(1, min(6, *scale)))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(*fullscreen)
	if b, e := assets.FS.Open("icon/SkyOfDie.png"); e == nil {
		if icon, _, e := image.Decode(b); e == nil {
			ebiten.SetWindowIcon([]image.Image{icon})
		}
		b.Close()
	}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
	app.save()
}
