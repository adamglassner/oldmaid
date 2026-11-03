// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// Plays the music and sound effects rendered by synth.go. Everything is rendered once in the
// background at start-up (a couple of seconds), then played from memory.
package main

import (
	"bytes"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Sound struct {
	ctx *audio.Context

	mu        sync.Mutex
	sfx       map[string][][]byte
	music     *audio.Player
	starts    []float64
	loopLen   float64
	vol       float64 // master volume 0..1
	musicOn   bool
	sfxOn     bool
	musicGain float64 // fades towards 1 or 0
	last      time.Time
}

func newSound(vol float64, musicOn, sfxOn bool) *Sound {
	s := &Sound{sfx: map[string][][]byte{}, vol: vol, musicOn: musicOn, sfxOn: sfxOn, last: time.Now()}
	if os.Getenv("OLDMAID_NOAUDIO") == "1" { // automated checks on machines without sound
		return s
	}
	s.ctx = audio.NewContext(sampleRate)
	go s.render()
	return s
}

func (s *Sound) render() {
	r := newRoom()
	for _, n := range sfxNames {
		takes := sfxTakes[n]
		if takes == 0 {
			takes = 1
		}
		var list [][]byte
		for k := 0; k < takes; k++ {
			list = append(list, pcm16(trimTail(renderSfx(n, int64(k+1), r))))
		}
		s.mu.Lock()
		s.sfx[n] = list
		s.mu.Unlock()
	}
	m, starts := renderMusic(r)
	data := pcm16(m)
	loop := audio.NewInfiniteLoop(bytes.NewReader(data), int64(len(data)))
	p, err := s.ctx.NewPlayer(loop)
	if err != nil {
		return
	}
	p.SetBufferSize(200 * time.Millisecond)
	s.mu.Lock()
	s.music, s.starts, s.loopLen = p, starts, float64(len(m[0]))/sampleRate
	if s.musicOn {
		s.musicGain = 0
		p.SetVolume(0)
		p.Play()
	}
	s.mu.Unlock()
}

// trimTail drops the silent end of a rendered effect so short sounds don't hold players open.
func trimTail(s [2][]float64) [2][]float64 {
	n := len(s[0])
	for n > 0 && math.Abs(s[0][n-1]) < 1e-4 && math.Abs(s[1][n-1]) < 1e-4 {
		n--
	}
	n = min(len(s[0]), n+sampleRate/50)
	return [2][]float64{s[0][:n], s[1][:n]}
}

func (s *Sound) play(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.sfxOn || s.vol <= 0 || s.ctx == nil {
		return
	}
	list := s.sfx[name]
	if len(list) == 0 {
		return
	}
	p := s.ctx.NewPlayerFromBytes(list[rand.Intn(len(list))])
	p.SetVolume(s.vol)
	p.Play()
}

func (s *Sound) setVolume(v float64) { s.mu.Lock(); s.vol = v; s.mu.Unlock() }
func (s *Sound) setSfx(on bool)      { s.mu.Lock(); s.sfxOn = on; s.mu.Unlock() }

func (s *Sound) setMusic(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.musicOn = on
	if on && s.music != nil && !s.music.IsPlaying() {
		s.music.Play()
	}
}

// update fades the music in and out; called every frame.
func (s *Sound) update() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	dt := now.Sub(s.last).Seconds()
	s.last = now
	if s.music == nil {
		return
	}
	target := 0.0
	if s.musicOn {
		target = 1
	}
	s.musicGain += (target - s.musicGain) * (1 - math.Exp(-dt/0.15))
	if math.Abs(s.musicGain-target) < 0.002 {
		s.musicGain = target
	}
	s.music.SetVolume(s.vol * s.musicGain)
	if s.musicGain == 0 && !s.musicOn && s.music.IsPlaying() {
		s.music.Pause()
	}
}

func (s *Sound) tuneName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.music == nil || s.loopLen == 0 {
		return tunes[0].name
	}
	pos := math.Mod(s.music.Position().Seconds(), s.loopLen)
	idx := 0
	for i, st := range s.starts {
		if pos >= st {
			idx = i
		}
	}
	return tunes[idx].name
}

func (s *Sound) musicReady() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.music != nil }
