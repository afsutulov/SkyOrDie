// Package assets embeds converted PNG/WAV resources. No DOS programs or legacy
// compressed resource files are part of the native game.
package assets

import (
	"embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io/fs"
	"path"
	"skyordie/internal/game"
)

//go:embed images/*.png images/*.json sprites/*.png sprites/atlas.json audio/*.wav audio/*.ogg icon/*.png snow.json downhill.json tube.json stunts.json frontend.json
var FS embed.FS

type Data struct {
	Images map[string]image.Image
	Sheets map[string]image.Image
	Rects  map[string][][4]int
	Config game.Config
}

func Load() (*Data, error) {
	d := &Data{Images: map[string]image.Image{}, Sheets: map[string]image.Image{}, Rects: map[string][][4]int{}}
	for _, dir := range []string{"images", "sprites"} {
		files, e := fs.ReadDir(FS, dir)
		if e != nil {
			return nil, e
		}
		for _, f := range files {
			if path.Ext(f.Name()) != ".png" {
				continue
			}
			p := path.Join(dir, f.Name())
			r, e := FS.Open(p)
			if e != nil {
				return nil, e
			}
			im, _, e := image.Decode(r)
			r.Close()
			if e != nil {
				return nil, fmt.Errorf("decode %s: %w", p, e)
			}
			key := f.Name()[:len(f.Name())-4]
			if dir == "images" {
				d.Images[key] = im
			} else {
				d.Sheets[key] = im
			}
		}
	}
	b, e := FS.ReadFile("sprites/atlas.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Rects); e != nil {
		return nil, e
	}
	for i, n := range []string{"dhcourse", "tbcourse", "ajcourse"} {
		b, e = FS.ReadFile("images/" + n + ".json")
		if e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &d.Config.Courses[i]); e != nil {
			return nil, e
		}
	}
	b, e = FS.ReadFile("snow.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Config.SnowTables); e != nil {
		return nil, e
	}
	b, e = FS.ReadFile("frontend.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Config.Front); e != nil {
		return nil, e
	}
	b, e = FS.ReadFile("stunts.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Config.Stunts); e != nil {
		return nil, e
	}
	if len(d.Config.Stunts.Air.Ramp) != 163 || len(d.Config.Stunts.Pipe.Actions) != 108 {
		return nil, fmt.Errorf("incomplete stunt data")
	}
	b, e = FS.ReadFile("downhill.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Config.Downhill); e != nil {
		return nil, e
	}
	b, e = FS.ReadFile("tube.json")
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(b, &d.Config.Tube); e != nil {
		return nil, e
	}
	if len(d.Config.Downhill.RespawnX) != 500 || len(d.Config.Downhill.Actions) != 15 || len(d.Config.Tube.RouteX) != 500 {
		return nil, fmt.Errorf("invalid recovered race tables")
	}
	for _, r := range d.Rects["sffbm"] {
		d.Config.SpriteSizes = append(d.Config.SpriteSizes, [2]int{r[2], r[3]})
	}
	if len(d.Config.SnowTables.Behaviors) != 41 || len(d.Config.SnowTables.Spawns) != 40 {
		return nil, fmt.Errorf("invalid snowball tables")
	}
	return d, nil
}
func (d *Data) Sprite(bank string, id int) image.Image {
	rs := d.Rects[bank]
	if id < 0 || id >= len(rs) {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	r := rs[id]
	return d.Sheets[bank].(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3]))
}
