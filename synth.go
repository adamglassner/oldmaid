// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// Music and sound effects, synthesized in Go (all original). This is a sample-exact
// port of the game's Web Audio instruments: a music box, a parlour piano and filtered
// noise, played into a small parlour-room reverb.
//
// Tunes: "Tea at Four" (music-box waltz), "The Parlour Rag" (piano),
//
//	"Lamplight Nocturne" (music box), "Sunday Promenade" (piano).
package main

import (
	"math"
	"math/cmplx"
	"math/rand"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const sampleRate = 44100

func hz(m float64) float64 { return 440 * math.Pow(2, (m-69)/12) }

/* ---------------- automation (Web Audio AudioParam semantics) ---------------- */

type paramEv struct {
	kind int // 0 set, 1 linear ramp, 2 exponential ramp
	t, v float64
}

type param struct {
	def float64
	ev  []paramEv
}

func (p *param) set(v, t float64) *param { p.ev = append(p.ev, paramEv{0, t, v}); return p }
func (p *param) lin(v, t float64) *param { p.ev = append(p.ev, paramEv{1, t, v}); return p }
func (p *param) exp(v, t float64) *param { p.ev = append(p.ev, paramEv{2, t, v}); return p }

func (p *param) at(t float64) float64 {
	v, t0 := p.def, math.Inf(-1)
	for _, e := range p.ev {
		if t < e.t {
			switch e.kind {
			case 1:
				if t0 == math.Inf(-1) {
					return v
				}
				return v + (e.v-v)*(t-t0)/(e.t-t0)
			case 2:
				if t0 == math.Inf(-1) || v <= 0 || e.v <= 0 {
					return v
				}
				return v * math.Pow(e.v/v, (t-t0)/(e.t-t0))
			}
			return v
		}
		v, t0 = e.v, e.t
	}
	return v
}

/* ---------------- building blocks ---------------- */

type wave int

const (
	sine wave = iota
	triangle
	sawtooth
)

// polyBLEP keeps the sawtooth band-limited, like Web Audio's periodic waves.
func polyBLEP(ph, dt float64) float64 {
	if ph < dt {
		ph /= dt
		return ph + ph - ph*ph - 1
	}
	if ph > 1-dt {
		ph = (ph - 1) / dt
		return ph*ph + ph + ph + 1
	}
	return 0
}

func osc(w wave, ph, dt float64) float64 {
	switch w {
	case triangle:
		if ph < 0.25 {
			return 4 * ph
		} else if ph < 0.75 {
			return 2 - 4*ph
		}
		return 4*ph - 4
	case sawtooth:
		return 2*ph - 1 - polyBLEP(ph, dt)
	}
	return math.Sin(2 * math.Pi * ph)
}

type filterKind int

const (
	lowpass filterKind = iota
	highpass
	bandpass
)

// biquad follows the Web Audio spec (lowpass/highpass Q is in dB).
type biquad struct {
	kind               filterKind
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
	lastF, lastQ       float64
}

func (b *biquad) setup(f, q float64) {
	if f == b.lastF && q == b.lastQ {
		return
	}
	b.lastF, b.lastQ = f, q
	f = math.Max(10, math.Min(f, sampleRate/2-10))
	w0 := 2 * math.Pi * f / sampleRate
	cw, sw := math.Cos(w0), math.Sin(w0)
	var a0, alpha float64
	switch b.kind {
	case lowpass:
		alpha = sw / (2 * math.Pow(10, q/20))
		b.b0, b.b1, b.b2 = (1-cw)/2, 1-cw, (1-cw)/2
	case highpass:
		alpha = sw / (2 * math.Pow(10, q/20))
		b.b0, b.b1, b.b2 = (1+cw)/2, -(1 + cw), (1+cw)/2
	case bandpass:
		alpha = sw / (2 * math.Max(q, 1e-4))
		b.b0, b.b1, b.b2 = alpha, 0, -alpha
	}
	a0 = 1 + alpha
	b.a1, b.a2 = -2*cw/a0, (1-alpha)/a0
	b.b0, b.b1, b.b2 = b.b0/a0, b.b1/a0, b.b2/a0
}

func (b *biquad) run(x float64) float64 {
	y := b.b0*x + b.b1*b.x1 + b.b2*b.x2 - b.a1*b.y1 - b.a2*b.y2
	b.x2, b.x1, b.y2, b.y1 = b.x1, x, b.y1, y
	return y
}

/* ---------------- a bus: dry mono signal, later sent through the room ---------------- */

type bus struct {
	d   []float64
	rng *rand.Rand
}

func newBus(seconds float64, seed int64) *bus {
	return &bus{d: make([]float64, int(seconds*sampleRate)+1), rng: rand.New(rand.NewSource(seed))}
}

func (b *bus) grow(n int) {
	if n > len(b.d) {
		b.d = append(b.d, make([]float64, n-len(b.d))...)
	}
}

func (b *bus) span(t0, t1 float64) (int, int) {
	i0, i1 := int(math.Max(0, t0)*sampleRate), int(t1*sampleRate)+1
	b.grow(i1)
	return i0, i1
}

// env reads an automation curve every 16 samples and interpolates in between.
type env struct {
	p    *param
	i0   int
	a, b float64
	seg  int
}

func newEnv(p *param, i0 int) *env { return &env{p: p, i0: i0, seg: -1} }

func (e *env) at(i int) float64 {
	k := i - e.i0
	if s := k >> 4; s != e.seg {
		e.seg = s
		e.a = e.p.at(float64(e.i0+s<<4) / sampleRate)
		e.b = e.p.at(float64(e.i0+(s+1)<<4) / sampleRate)
	}
	return e.a + (e.b-e.a)*float64(k&15)/16
}

// rotor is a sine oscillator by complex rotation: one multiply-add per sample.
type rotor struct{ c, s, cr, sr float64 }

func newRotor(f, phase float64) *rotor {
	w := 2 * math.Pi * f / sampleRate
	return &rotor{math.Cos(phase), math.Sin(phase), math.Cos(w), math.Sin(w)}
}

func (r *rotor) next() float64 {
	v := r.s
	r.c, r.s = r.c*r.cr-r.s*r.sr, r.s*r.cr+r.c*r.sr
	return v
}

func (b *bus) musicBox(t, m, vel float64) {
	f := hz(m)
	i0, i1 := b.span(t, t+2.3)
	t0 := float64(i0) / sampleRate
	g := newEnv((&param{def: 1}).set(0, t).lin(vel*0.32, t+0.004).exp(0.0008, t+2.2), i0)
	mults := []float64{1, 3.01, 5.43, 2}
	amps := []float64{1, 0.22, 0.08, 0.12}
	var rot [4]*rotor
	var envs [4]*env
	for k, mu := range mults {
		rot[k] = newRotor(f*mu, 2*math.Pi*f*mu*(t0-t))
		if mu > 2 {
			envs[k] = newEnv((&param{def: amps[k]}).set(amps[k], t).exp(0.001, t+0.4), i0)
		}
	}
	for i := i0; i < i1; i++ {
		s := 0.0
		for k := range mults {
			a := amps[k]
			if envs[k] != nil {
				if i-i0 < int(0.45*sampleRate) {
					a = envs[k].at(i)
				} else {
					rot[k].next()
					continue
				}
			}
			s += a * rot[k].next()
		}
		b.d[i] += s * g.at(i)
	}
}

func (b *bus) piano(t, m, dur, vel float64) {
	f := hz(m)
	lp := &biquad{kind: lowpass}
	lpf := (&param{def: 350}).set(math.Min(9000, f*9), t).exp(math.Max(300, f*2), t+1.2)
	end := t + math.Max(0.25, dur) + 0.35
	g0 := (&param{def: 1}).set(0, t).lin(vel*0.28, t+0.005).exp(vel*0.1, t+0.25).set(vel*0.1, end-0.35).exp(0.0008, end)
	type part struct {
		w             wave
		freq, amp, ph float64
	}
	parts := []*part{{triangle, f, 0.8, 0}, {sawtooth, f * math.Pow(2, 3.0/1200), 0.14, 0}, {sine, f * 2 * math.Pow(2, -2.0/1200), 0.18, 0}}
	i0, i1 := b.span(t, end+0.05)
	g := newEnv(g0, i0)
	for i := i0; i < i1; i++ {
		if (i-i0)%16 == 0 {
			lp.setup(lpf.at(float64(i)/sampleRate), 0.4)
		}
		s := 0.0
		for _, p := range parts {
			dt := p.freq / sampleRate
			s += p.amp * osc(p.w, p.ph, dt)
			if p.ph += dt; p.ph >= 1 {
				p.ph -= 1
			}
		}
		b.d[i] += lp.run(s) * g.at(i)
	}
}

func (b *bus) noise(t, dur, freq, q, vel float64, kind filterKind) {
	f := &biquad{kind: kind}
	f.setup(freq, q)
	g := (&param{def: 1}).set(0, t).lin(vel, t+0.006).exp(0.001, t+dur)
	i0, i1 := b.span(t, t+dur+0.02)
	for i := i0; i < i1; i++ {
		b.d[i] += f.run(b.rng.Float64()*2-1) * g.at(float64(i)/sampleRate)
	}
}

// sad trombone, Victorian edition
func (b *bus) lose(t float64) {
	for i, m := range []float64{58, 57, 56, 55} {
		st := t + float64(i)*0.42
		d := 0.36
		if i == 3 {
			d = 1.1
		}
		lp := &biquad{kind: lowpass}
		lpf := (&param{def: 350}).set(400, st).lin(1100, st+0.08).lin(600, st+d)
		g := (&param{def: 1}).set(0, st).lin(0.32, st+0.04).set(0.3, st+d-0.08).lin(0, st+d)
		ph := 0.0
		i0, i1 := b.span(st, st+d+0.05)
		for k := i0; k < i1; k++ {
			tt := float64(k) / sampleRate
			if (k-i0)%16 == 0 {
				lp.setup(lpf.at(tt), 1)
			}
			fr := hz(m - 12)
			if i == 3 && tt < st+d {
				fr += 6 * math.Sin(2*math.Pi*6*(tt-st))
			}
			dt := fr / sampleRate
			s := osc(sawtooth, ph, dt)
			if ph += dt; ph >= 1 {
				ph -= 1
			}
			b.d[k] += lp.run(s) * g.at(tt)
		}
	}
}

/* ---------------- the room: a convolution reverb ---------------- */

type room struct {
	irLen int
	n     int             // FFT size
	block int             // input block per FFT
	ir    [2][]complex128 // spectra of the two impulse responses
}

func newRoom() *room {
	r := &room{irLen: int(sampleRate * 1.6)}
	r.n = 1
	for r.n < r.irLen*2 {
		r.n <<= 1
	}
	r.block = r.n - r.irLen + 1
	rng := rand.New(rand.NewSource(1890))
	irs := [2][]float64{}
	power := 0.0
	for ch := 0; ch < 2; ch++ {
		irs[ch] = make([]float64, r.irLen)
		for i := range irs[ch] {
			v := (rng.Float64()*2 - 1) * math.Pow(1-float64(i)/float64(r.irLen), 3.2)
			irs[ch][i] = v
			power += v * v
		}
	}
	// Web Audio's ConvolverNode normalizes its impulse response like this.
	power = math.Sqrt(power / float64(2*r.irLen))
	scale := 1 / math.Max(power, 0.000125) * math.Pow(10, -58*0.05) * 44100 / sampleRate
	for ch := 0; ch < 2; ch++ {
		buf := make([]complex128, r.n)
		for i, v := range irs[ch] {
			buf[i] = complex(v*scale, 0)
		}
		fft(buf, false)
		r.ir[ch] = buf
	}
	return r
}

// apply returns the wet stereo signal (same length as dry plus the reverb tail).
func (r *room) apply(dry []float64) [2][]float64 {
	n := len(dry) + r.irLen
	var out [2][]float64
	out[0], out[1] = make([]float64, n), make([]float64, n)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for start := 0; start < len(dry); start += r.block {
		start := start
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			end := min(start+r.block, len(dry))
			x := make([]complex128, r.n)
			silent := true
			for i := start; i < end; i++ {
				x[i-start] = complex(dry[i], 0)
				if dry[i] != 0 {
					silent = false
				}
			}
			if silent {
				return
			}
			fft(x, false)
			var res [2][]complex128
			for ch := 0; ch < 2; ch++ {
				y := make([]complex128, r.n)
				for i := range y {
					y[i] = x[i] * r.ir[ch][i]
				}
				fft(y, true)
				res[ch] = y
			}
			mu.Lock()
			for ch := 0; ch < 2; ch++ {
				for i := 0; i < r.n && start+i < n; i++ {
					out[ch][start+i] += real(res[ch][i])
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func fft(a []complex128, inverse bool) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		ang := 2 * math.Pi / float64(size)
		if !inverse {
			ang = -ang
		}
		wl := cmplx.Rect(1, ang)
		half := size / 2
		tw := make([]complex128, half)
		w := complex(1, 0)
		for k := range tw {
			tw[k] = w
			w *= wl
		}
		for i := 0; i < n; i += size {
			for k := 0; k < half; k++ {
				u, v := a[i+k], a[i+k+half]*tw[k]
				a[i+k], a[i+k+half] = u+v, u-v
			}
		}
	}
	if inverse {
		inv := complex(1/float64(n), 0)
		for i := range a {
			a[i] *= inv
		}
	}
}

// mixdown: bus gain into the master, plus the same signal through the room at 0.22.
func (r *room) mixdown(dry []float64, gain float64) [2][]float64 {
	wet := r.apply(dry)
	for ch := 0; ch < 2; ch++ {
		for i := range wet[ch] {
			v := 0.22 * wet[ch][i]
			if i < len(dry) {
				v += dry[i]
			}
			wet[ch][i] = v * gain
		}
	}
	return wet
}

/* ---------------- tunes ---------------- */

type tune struct {
	name, inst, style string
	bpm, beats        float64
	chords, mel       string
}

var tunes = []tune{
	{"Tea at Four", "box", "waltz", 126, 3, "F F C7 C7 C7 C7 F F F F Bb Bb F C7 F F",
		`A4:2 C5:1 | F5:2 E5:1 | D5:1 C5:1 Bb4:1 | G4:3 | G4:2 Bb4:1 | E5:2 D5:1 | C5:1 A4:1 F4:1 | C5:3 |
		 A4:2 C5:1 | F5:2 A5:1 | G5:1 F5:1 D5:1 | Bb4:3 | A4:1 C5:1 F5:1 | E5:2 G5:1 | F5:1 C5:1 A4:1 | F4:3`},
	{"The Parlour Rag", "piano", "stride", 92, 2, "C C G7 G7 C C D7 G7 C C7 F Fm C G7 C C",
		`E5:.5 G5:.5 E5:.25 C5:.25 E5:.5 | G5:.75 A5:.25 G5:1 | F5:.5 D5:.5 B4:.5 D5:.5 | F5:.75 E5:.25 D5:1 |
		 E5:.5 G5:.5 C6:.5 B5:.5 | A5:.5 G5:.5 E5:1 | F#5:.5 A5:.5 F#5:.5 D5:.5 | G5:.75 F5:.25 D5:.5 B4:.5 |
		 C5:.5 E5:.5 G5:.5 C6:.5 | Bb5:.75 A5:.25 G5:1 | A5:.5 F5:.5 C5:.5 F5:.5 | Ab5:.75 G5:.25 F5:1 |
		 E5:.5 G5:.5 E5:.5 C5:.5 | D5:.5 F5:.5 B4:.5 D5:.5 | C5:.5 E5:.25 G5:.25 C6:.5 G5:.5 | C6:1.5 r:.5`},
	{"Lamplight Nocturne", "box", "arp", 84, 3, "Em Em Am Am D D G B7 Em C Am B7 Em Am B7 Em",
		`B4:2 E5:1 | G5:2 F#5:1 | E5:1 C5:1 A4:1 | B4:3 | A4:2 D5:1 | F#5:2 E5:1 | D5:1 B4:1 G4:1 | F#4:3 |
		 E4:1 G4:1 B4:1 | E5:2 D5:1 | C5:1 A4:1 E5:1 | D#5:3 | E5:2 G5:1 | F#5:1 E5:1 C5:1 | B4:2 D#5:1 | E5:3`},
	{"Sunday Promenade", "piano", "march", 108, 4, "G G C G D7 D7 G D7 G G C A7 G D7 G G",
		`D5:1 G5:1 F#5:.5 E5:.5 D5:1 | B4:1.5 C5:.5 D5:2 | E5:1 G5:1 E5:1 C5:1 | B4:2 G4:2 |
		 A4:1 D5:1 C5:.5 B4:.5 A4:1 | F#4:1.5 G4:.5 A4:2 | B4:1 D5:1 G5:1 F#5:1 | A5:3 r:1 |
		 D5:1 G5:1 F#5:.5 E5:.5 D5:1 | B5:1.5 A5:.5 G5:2 | E5:1 G5:1 C6:1 E5:1 | C#5:2 A4:2 |
		 B4:1 D5:1 G5:1 B4:1 | A4:1 C5:1 F#5:1 A4:1 | G4:1 B4:.5 D5:.5 G5:1 D5:1 | G5:3 r:1`},
}

var noteNum = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
var noteRe = regexp.MustCompile(`^([A-G])([#b]?)(-?\d)$`)
var chordRe = regexp.MustCompile(`^([A-G])([#b]?)(m?)(7?)$`)

func accidental(s string) int {
	switch s {
	case "#":
		return 1
	case "b":
		return -1
	}
	return 0
}

func midi(n string) float64 {
	m := noteRe.FindStringSubmatch(n)
	oct, _ := strconv.Atoi(m[3])
	return float64(12*(oct+1) + noteNum[m[1][0]] + accidental(m[2]))
}

type chordTones struct {
	bass  float64
	tones []float64
}

func mod12(x int) int { return ((x % 12) + 12) % 12 }

func chord(name string) chordTones {
	m := chordRe.FindStringSubmatch(name)
	pc := mod12(noteNum[m[1][0]] + accidental(m[2]))
	iv := []int{0, 4, 7}
	if m[3] != "" {
		iv = []int{0, 3, 7}
	}
	if m[4] != "" {
		iv = append(iv, 10)
	}
	c := chordTones{bass: float64(36 + mod12(pc-36))}
	for _, i := range iv {
		c.tones = append(c.tones, float64(53+mod12(pc+i-53)))
	}
	sort.Float64s(c.tones)
	return c
}

type noteEv struct{ t, m, d, v float64 }

func (tn tune) events() ([]noteEv, float64) {
	spb := 60 / tn.bpm
	var ev []noteEv
	t := 0.0
	for _, tok := range strings.Fields(tn.mel) {
		if tok == "|" {
			continue
		}
		parts := strings.SplitN(tok, ":", 2)
		d, _ := strconv.ParseFloat(parts[1], 64)
		if parts[0] != "r" {
			ev = append(ev, noteEv{t, midi(parts[0]), d * spb, 0.62})
		}
		t += d * spb
	}
	chords := strings.Fields(tn.chords)
	bar := tn.beats * spb
	for i, cn := range chords {
		c, b0 := chord(cn), float64(i)*bar
		switch tn.style {
		case "waltz":
			ev = append(ev, noteEv{b0, c.bass, spb, 0.42})
			for _, k := range []float64{1, 2} {
				for _, n := range c.tones {
					ev = append(ev, noteEv{b0 + k*spb, n, spb * 0.8, 0.16})
				}
			}
		case "arp":
			ev = append(ev, noteEv{b0, c.bass + 12, bar, 0.3})
			for j, n := range c.tones {
				ev = append(ev, noteEv{b0 + float64(j+1)*spb*0.66, n, spb, 0.14})
			}
		case "stride":
			for k := 0; k < int(tn.beats); k++ {
				bs := c.bass
				if k%2 == 1 {
					bs += 7
				}
				ev = append(ev, noteEv{b0 + float64(k)*spb, bs, spb * 0.45, 0.4})
				for _, n := range c.tones {
					ev = append(ev, noteEv{b0 + float64(k)*spb + spb/2, n, spb * 0.4, 0.15})
				}
			}
		default: // march
			for k := 0; k < int(tn.beats); k++ {
				bs := c.bass
				if k%2 == 1 {
					bs += 7
				}
				ev = append(ev, noteEv{b0 + float64(k)*spb, bs, spb * 0.5, 0.36})
				if k%2 == 1 {
					for _, n := range c.tones {
						ev = append(ev, noteEv{b0 + float64(k)*spb, n, spb * 0.5, 0.13})
					}
				}
			}
		}
	}
	sort.SliceStable(ev, func(a, b int) bool { return ev[a].t < ev[b].t })
	return ev, float64(len(chords)) * bar
}

// renderMusic plays the four tunes back to back (a breath of 1.6 s between them,
// as the game always has) and folds the tail onto the start so the loop is seamless.
// It returns the stereo loop and where each tune starts.
func renderMusic(r *room) ([2][]float64, []float64) {
	b := newBus(130, 7)
	var starts []float64
	at := 0.0
	for _, tn := range tunes {
		starts = append(starts, at)
		ev, length := tn.events()
		for _, e := range ev {
			if tn.inst == "box" {
				b.musicBox(at+e.t, e.m, e.v*1.25)
			} else {
				b.piano(at+e.t, e.m, e.d, e.v)
			}
		}
		at += length + 1.6
	}
	loop := int(at * sampleRate)
	out := r.mixdown(b.d, 0.55)
	for ch := 0; ch < 2; ch++ {
		for i := loop; i < len(out[ch]); i++ {
			out[ch][i-loop] += out[ch][i]
		}
		out[ch] = out[ch][:loop]
	}
	return out, starts
}

// sound effects; the noisy ones get a few takes so repeats don't sound identical
var sfxTakes = map[string]int{"slide": 3, "flip": 3, "shuffle": 2, "deal": 4, "tick": 2, "hover": 3}

func renderSfx(name string, seed int64, r *room) [2][]float64 {
	b := newBus(0.5, seed)
	box := func(t, m, v float64) { b.musicBox(t, m, v) }
	switch name {
	case "slide":
		b.noise(0, 0.16, 2600, 0.7, 0.32, bandpass)
		b.noise(0.02, 0.08, 5200, 1.2, 0.12, bandpass)
	case "flip":
		b.noise(0, 0.05, 3800, 1.5, 0.35, highpass)
		b.noise(0.03, 0.04, 1800, 2, 0.15, bandpass)
	case "shuffle":
		for i := 0; i < 18; i++ {
			b.noise(float64(i)*0.045+b.rng.Float64()*0.01, 0.06, 2200+b.rng.Float64()*1600, 1, 0.14, bandpass)
		}
	case "deal":
		b.noise(0, 0.1, 3000, 0.9, 0.2, bandpass)
	case "pair":
		box(0, 88, 0.6)
		box(0.09, 95, 0.5)
	case "out":
		for i, m := range []float64{79, 83, 86, 91} {
			box(float64(i)*0.08, m, 0.55)
		}
	case "tick":
		b.noise(0, 0.03, 4200, 2, 0.15, highpass)
	case "hover":
		b.noise(0, 0.04, 3200, 3, 0.05, bandpass)
	case "maid": // a dramatic little "gasp" chord for drawing the Old Maid
		for _, m := range []float64{43, 49, 55, 58} {
			b.piano(0, m, 0.9, 0.42)
		}
		for _, m := range []float64{61, 67} {
			b.piano(0.28, m, 0.8, 0.3)
		}
	case "phew":
		for i, m := range []float64{72, 76, 79} {
			b.piano(float64(i)*0.07, m, 0.3, 0.34)
		}
	case "win":
		for i, m := range []float64{60, 64, 67, 72, 76, 79, 84} {
			b.piano(float64(i)*0.09, m, 0.5, 0.45)
			box(float64(i)*0.09, m+12, 0.35)
		}
		for _, m := range []float64{60, 64, 67, 72} {
			b.piano(0.75, m, 1.4, 0.32)
		}
	case "lose":
		b.lose(0)
	}
	return r.mixdown(b.d, 0.9)
}

var sfxNames = []string{"slide", "flip", "shuffle", "deal", "pair", "out", "tick", "hover", "maid", "phew", "win", "lose"}

// pcm16 interleaves a stereo signal as 16-bit little-endian samples.
func pcm16(s [2][]float64) []byte {
	out := make([]byte, len(s[0])*4)
	for i := range s[0] {
		for ch := 0; ch < 2; ch++ {
			v := int16(math.Max(-1, math.Min(1, s[ch][i])) * 32767)
			out[i*4+ch*2] = byte(v)
			out[i*4+ch*2+1] = byte(v >> 8)
		}
	}
	return out
}
