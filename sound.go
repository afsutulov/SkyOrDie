package main

import (
	"bytes"
	"io"
	"log"
	"skyordie/assets"
	"skyordie/internal/game"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

type soundSystem struct {
	ctx             *audio.Context
	muted, disabled bool
	music           *audio.Player
	track           string
	card            string
	cues            map[string][]byte
	voices          []*audio.Player
	loops           map[string]*audio.Player
}

func newSound(muted bool) *soundSystem {
	return &soundSystem{muted: muted, card: "pc_", cues: map[string][]byte{}, loops: map[string]*audio.Player{}}
}
func (s *soundSystem) setMute(m bool) {
	s.muted = m
	if s.music != nil {
		if m {
			s.music.Pause()
		} else {
			s.music.Play()
		}
	}
	for _, p := range s.loops {
		if m {
			p.Pause()
		} else {
			p.Play()
		}
	}
	if m {
		for _, p := range s.voices {
			p.Pause()
		}
	}
}
func (s *soundSystem) fail(err error) {
	log.Printf("Audio disabled: %v", err)
	s.disabled = true
	s.close()
}
func (s *soundSystem) update(g *game.Game) {
	if s.muted != g.Save.Muted {
		s.setMute(g.Save.Muted)
	}
	if s.muted || s.disabled {
		return
	}
	if s.ctx == nil {
		s.ctx = audio.NewContext(44100)
	}
	if g.Phase != game.Playing {
		for name, p := range s.loops {
			p.Close()
			delete(s.loops, name)
		}
	}
	track := ""
	if g.Phase == game.Splash {
		track = "menu"
	}
	if g.Phase == game.Playing && !g.Waiting {
		switch g.Event {
		case game.Downhill:
			track = "downhill"
		case game.Tube:
			track = "tube"
		case game.Halfpipe:
			track = "halfpipe"
		}
	}

	if track != s.track {
		if s.music != nil {
			s.music.Close()
			s.music = nil
		}
		s.track = track
		if track != "" {
			b, err := assets.FS.ReadFile("audio/" + s.card + track + ".ogg")
			if err != nil {
				s.fail(err)
				return
			}
			stream, err := vorbis.DecodeWithSampleRate(44100, bytes.NewReader(b))
			if err != nil {
				s.fail(err)
				return
			}
			loop := audio.NewInfiniteLoop(stream, stream.Length())
			s.music, err = s.ctx.NewPlayer(loop)
			if err != nil {
				s.fail(err)
				return
			}
			s.music.SetVolume(1)
			s.music.Play()
		}
	}
	if s.music != nil {
		if g.Paused {
			s.music.Pause()
		} else if !s.music.IsPlaying() {
			s.music.Play()
		}
	}
	live := s.voices[:0]
	for _, v := range s.voices {
		if v.IsPlaying() {
			live = append(live, v)
		} else {
			v.Close()
		}
	}
	s.voices = live
	for _, name := range g.Sound {
		if name == "fx18" {
			if p := s.loops["fx17"]; p != nil {
				p.Close()
				delete(s.loops, "fx17")
			}
			continue
		}
		if name == "fx17" && s.card == "pc_" {
			continue
		}
		if name == "fx17" && s.loops[name] != nil {
			continue
		}
		if s.card == "pc_" && track != "" {
			continue
		}
		if s.card == "pc_" {
			for _, p := range s.voices {
				p.Close()
			}
			s.voices = nil
		}
		if len(s.voices) >= 12 {
			break
		}
		b, ok := s.cues[name]
		if !ok {
			src, err := assets.FS.ReadFile("audio/" + s.card + name + ".wav")
			if err != nil {
				continue
			}
			w, err := wav.DecodeWithSampleRate(44100, bytes.NewReader(src))
			if err != nil {
				continue
			}
			b, err = io.ReadAll(w)
			if err != nil {
				continue
			}
			s.cues[name] = b
		}
		var source io.Reader = bytes.NewReader(b)
		if name == "fx17" {
			source = audio.NewInfiniteLoop(bytes.NewReader(b), int64(len(b)))
		}
		p, err := s.ctx.NewPlayer(source)
		if err != nil {
			s.fail(err)
			return
		}
		p.SetVolume(1)
		p.Play()
		if name == "fx17" {
			s.loops[name] = p
		} else {
			s.voices = append(s.voices, p)
		}
	}
}
func (s *soundSystem) close() {
	for name, p := range s.loops {
		p.Close()
		delete(s.loops, name)
	}
	if s.music != nil {
		s.music.Close()
		s.music = nil
	}
	for _, p := range s.voices {
		p.Close()
	}
	s.voices = nil
}
