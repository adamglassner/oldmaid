// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// Turning the table into positions on this screen. You always sit at the bottom; the others
// sit round the table in turn order (the player on your left, the one opposite, then the one
// on your right), exactly as in the original.
package main

import (
	"math"
	"math/rand"
	"time"
)

var seatOrder = map[int][]byte{2: {'S', 'N'}, 3: {'S', 'W', 'E'}, 4: {'S', 'W', 'N', 'E'}}

func (g *Game) seatLetter(p *Player) byte {
	n := len(g.players)
	if n < 2 || n > 4 {
		return 'S'
	}
	return seatOrder[n][(p.seat-g.viewer+n)%n]
}

// canSee reports whether this screen may show the viewer's own hand face up.
func (g *Game) canSee() bool { return g.mode != modeHotseat || g.revealed }

func (g *Game) pileY() float64 { return g.Ht * 0.47 }

func (g *Game) seatAnchor(seat byte) (float64, float64) {
	switch seat {
	case 'S':
		return g.Wd / 2, g.Ht - g.ch*0.62 - 6
	case 'N':
		return g.Wd / 2, g.ch*0.62 + 8
	case 'W':
		return g.ch*0.62 + 10, g.Ht * 0.41
	}
	return g.Wd - g.ch*0.62 - 10, g.Ht * 0.41
}

func (g *Game) handPos(seat byte, i, n int) (x, y, r float64) {
	ax, ay := g.seatAnchor(seat)
	k := float64(i) - float64(n-1)/2
	fn := math.Max(float64(n), 1)
	switch seat {
	case 'S':
		sp := 0.0
		if n > 1 {
			sp = math.Min(g.cw*0.66, (g.Wd*0.62-g.cw)/float64(n-1))
		}
		return ax + k*sp, ay + math.Abs(k)*math.Abs(k)*0.9, k * math.Min(2.6, 34/fn)
	case 'N':
		sp := 0.0
		if n > 1 {
			sp = math.Min(g.cw*0.36, (g.Wd*0.30)/float64(n-1))
		}
		return ax - k*sp, ay - math.Abs(k)*math.Abs(k)*0.5, 180 - k*math.Min(2.2, 26/fn)
	}
	sp := 0.0
	if n > 1 {
		sp = math.Min(g.cw*0.34, (g.Ht*0.34)/float64(n-1))
	}
	dir := 1.0
	if seat == 'E' {
		dir = -1
	}
	return ax - dir*math.Abs(k)*math.Abs(k)*0.5, ay + dir*k*sp, dir*90 + dir*k*math.Min(2.2, 26/fn)
}

// cardTarget is where card c belongs on this screen.
func (g *Game) cardTarget(c *Card) (x, y, r, sc float64) {
	cx, cy := g.Wd/2, g.pileY()
	switch c.L {
	case lDeck:
		return cx + c.ux, cy + c.uy, c.rot, 1
	case lSplit:
		return cx + c.ux*g.cw*0.55, cy + c.uy, c.rot, 1
	case lHand:
		if c.owner >= 0 && c.owner < len(g.players) {
			x, y, r := g.handPos(g.seatLetter(g.players[c.owner]), c.idx, c.n)
			return x, y, r, 1
		}
	case lPile:
		return cx + c.ux*g.ch, cy + c.uy*g.ch, c.rot, c.sc
	case lReveal:
		return cx, cy, 0, c.sc
	}
	return cx, cy, 0, 1
}

func (g *Game) faceUp(c *Card) bool {
	return c.up || (c.L == lHand && c.owner == g.viewer && g.canSee() && g.players[c.owner].human())
}

// pickFrom is the seat whose cards this screen may draw from right now (-1 for none).
func (g *Game) pickFrom() int {
	if g.mode == modeClient {
		return g.clientPick
	}
	if g.busy == nil || g.busy.seat != g.viewer || g.players[g.viewer].kind != pkLocal || !g.canSee() || g.passTo >= 0 {
		return -1
	}
	return g.busy.from.seat
}

// updateView moves everything on this screen towards where the table says it should be.
func (g *Game) updateView() {
	now := time.Now()
	from := g.pickFrom()
	for _, c := range g.table {
		x, y, r, sc := g.cardTarget(c)
		dur := map[int]float64{modeNormal: 0.42, modeFast: 0.2, modeInstant: 0}[c.mode]
		// turning the long way round looks wrong, so keep rotations within half a turn
		if cur := c.ar.to; math.Abs(r-cur) > 180 {
			r += 360 * math.Round((cur-r)/360)
		}
		c.ax.set(x, now, dur, easeCard)
		c.ay.set(y, now, dur, easeCard)
		c.ar.set(r, now, dur, easeCard)
		c.as.set(sc, now, dur, easeCard)
		f := 1.0
		if g.faceUp(c) {
			f = 0
		}
		c.flip.set(f, now, 0.38, easeInOut)
		c.pick = from >= 0 && c.L == lHand && c.owner == from
	}
	g.layoutSeats()
	for _, p := range g.players {
		v := 0.0
		if p.seat == g.turnSeat {
			v = 1
		}
		p.turn.set(v, now, 0.3, easeCSS)
	}
	s := g.clientPrompt
	if g.mode != modeClient {
		s = ""
		if g.promptFn != nil && g.players != nil {
			s = g.promptFn(g.viewer)
		}
	}
	g.prompt(s)
}

func (g *Game) layoutAll() {}

func (g *Game) layoutSeats() {
	for _, p := range g.players {
		seat := g.seatLetter(p)
		ax, ay := g.seatAnchor(seat)
		var x, y float64
		switch seat {
		case 'N':
			x, y = g.Wd/2-g.Wd*0.15-g.cw*0.5-130, ay-4
		case 'W':
			x, y = ax+48, ay+g.Ht*0.17+g.cw*0.5+58
		case 'E':
			x, y = ax-48, ay+g.Ht*0.17+g.cw*0.5+58
		default:
			x, y = g.Wd/2-g.Wd*0.31-g.cw*0.5-70, g.Ht-g.ch*0.62
		}
		p.sx, p.sy = math.Max(110, math.Min(g.Wd-110, x)), y
	}
}

func ordinal(n int) string {
	suf := "th"
	if !(n%100 > 10 && n%100 < 14) {
		switch n % 10 {
		case 1:
			suf = "st"
		case 2:
			suf = "nd"
		case 3:
			suf = "rd"
		}
	}
	return itoa(n) + suf
}

func (g *Game) countText(p *Player) string {
	if p.label != "" {
		return p.label
	}
	if p.out {
		return "Safe — " + ordinal(p.place) + " out"
	}
	n := len(p.hand)
	if n == 1 {
		return "1 card"
	}
	return itoa(n) + " cards"
}

func (g *Game) prompt(s string) {
	if s == g.promptText && (s == "") == !g.promptOn {
		return
	}
	now := time.Now()
	if s != "" {
		g.promptText = s
		g.promptOn = true
		g.promptA.set(1, now, 0.25, easeCSS)
	} else {
		g.promptOn = false
		g.promptA.set(0, now, 0.25, easeCSS)
	}
}

func (g *Game) banner(title, sub string, ms float64) {
	g.bannerTitle, g.bannerSub = title, sub
	now := time.Now()
	g.bannerA.set(1, now, 0.3, easeCSS)
	g.bannerS.set(1, now, 0.35, easeBanner)
	g.bannerUntil = now.Add(time.Duration(ms) * time.Millisecond)
}

// applyEv shows one event on this screen.
func (g *Game) applyEv(ev *Ev) {
	switch ev.K {
	case "snd":
		g.snd.play(ev.S)
	case "banner":
		if ev.S != "" {
			g.snd.play(ev.S)
		}
		g.banner(ev.A, ev.B, ev.MS)
	case "say":
		if ev.Seat < len(g.players) {
			p := g.players[ev.Seat]
			p.say = ev.A
			now := time.Now()
			p.sayOn.set(1, now, 0.2, easeCSS)
			p.sayUntil = now.Add(time.Duration(ev.MS) * time.Millisecond)
		}
	case "burst":
		x, y := g.Wd/2+ev.X*g.ch, g.pileY()+ev.Y*g.ch
		if ev.Card != "" {
			for _, c := range g.table {
				if c.id == ev.Card {
					x, y, _, _ = g.cardTarget(c)
				}
			}
		}
		if ev.A == "win" {
			g.snd.play("win")
		} else if ev.A != "" {
			g.banner(ev.A, ev.B, ev.MS)
		}
		g.burst(x, y, ev.S, ev.N)
	case "result":
		// handled by the network code
	}
}

/* ---------------- effects ---------------- */

type particle struct {
	kind                     int // 0 spark, 1 puff, 2 petal
	x, y, vx, vy, life, d, s float64
	rot, vr                  float64
	r, gg, b                 uint8
}

func (g *Game) burst(x, y float64, kind string, n int) {
	for i := 0; i < n; i++ {
		switch kind {
		case "spark":
			a, v := rand.Float64()*math.Pi*2, 1.5+rand.Float64()*4
			g.parts = append(g.parts, &particle{kind: 0, x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 1, life: 1, d: 0.018 + rand.Float64()*0.02, s: 2 + rand.Float64()*3, rot: rand.Float64() * 6})
		case "puff":
			a, v := rand.Float64()*math.Pi*2, 0.8+rand.Float64()*1.8
			rr := g.cw * 0.5
			if n > 24 {
				rr = g.cw * 1.25
			}
			col := [3]uint8{120, 90, 140}
			if rand.Float64() >= .5 {
				col = [3]uint8{90, 80, 90}
			}
			g.parts = append(g.parts, &particle{kind: 1, x: x + math.Cos(a)*rr, y: y + math.Sin(a)*rr*1.3, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 0.4, life: 1, d: 0.011 + rand.Float64()*0.008, s: 12 + rand.Float64()*18, r: col[0], gg: col[1], b: col[2]})
		case "petal":
			c := hex(pick([]string{"#c8324a", "#e46b7a", "#f2a3b0", "#fbe3e8", "#d9b45a"}))
			g.parts = append(g.parts, &particle{kind: 2, x: rand.Float64() * g.Wd, y: -20 - rand.Float64()*g.Ht*0.6, vx: (rand.Float64() - .5) * 1.2, vy: 1 + rand.Float64()*1.6, life: 1, d: 0.0025, s: 6 + rand.Float64()*6, rot: rand.Float64() * 6, vr: (rand.Float64() - .5) * 0.1, r: c.R, gg: c.G, b: c.B})
		}
	}
}
