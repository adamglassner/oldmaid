// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// Drawing the parlour table: the felt, the doily, dust motes, the guests' seats, the cards,
// candlelight and the particle effects.
package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const barH = 50.0

// feltRect is where the table sits in the window.
func (g *Game) feltOrigin() (float64, float64) { return 16, barH + 16 }

func (g *Game) measure() {
	g.Wd, g.Ht = float64(g.W)-32, float64(g.H)-barH-32
	g.ch = math.Max(92, math.Min(math.Min(g.Ht*0.2, g.Wd*0.125), 210))
	g.cw = g.ch * artW / artH
}

/* ---------------- backgrounds (rendered once per window size) ---------------- */

func gradStops(t float64, stops []float64, cols []color.RGBA) [4]float64 {
	if t <= stops[0] {
		c := cols[0]
		return [4]float64{float64(c.R), float64(c.G), float64(c.B), float64(c.A)}
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i] {
			k := (t - stops[i-1]) / (stops[i] - stops[i-1])
			a, b := cols[i-1], cols[i]
			l := func(x, y uint8) float64 { return float64(x) + (float64(y)-float64(x))*k }
			return [4]float64{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
		}
	}
	c := cols[len(cols)-1]
	return [4]float64{float64(c.R), float64(c.G), float64(c.B), float64(c.A)}
}

// renderRoom draws the wood surround and brass rings around the table (the whole window).
func renderRoom(W, H int, Wd, Ht float64) *ebiten.Image {
	px := make([]byte, W*H*4)
	room, wood, brass2, brass := hex("#3a1d10"), hex("#5a2a16"), hex("#6b4a1e"), hex("#c9a24c")
	cx, cy := 16+Wd/2, barH+16+Ht/2
	hw, hh := Wd/2, Ht/2
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			fx, fy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := sdRR(fx, fy, hw, hh, 26)
			r, gg, b := float64(room.R), float64(room.G), float64(room.B)
			// drop shadow under the table
			sd := sdRR(fx, fy-10, hw, hh, 26)
			sh := 0.6 * 0.5 * math.Erfc(sd/(20*math.Sqrt2))
			r, gg, b = r*(1-sh), gg*(1-sh), b*(1-sh)
			layer := func(c color.RGBA, edge float64) {
				a := clamp(edge-d+0.5, 0, 1)
				r += (float64(c.R) - r) * a
				gg += (float64(c.G) - gg) * a
				b += (float64(c.B) - b) * a
			}
			layer(wood, 16)
			layer(brass2, 5)
			layer(brass, 3)
			i := (y*W + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = uint8(r), uint8(gg), uint8(b), 255
		}
	}
	im := ebiten.NewImage(W, H)
	im.WritePixels(px)
	return im
}

// renderFelt draws the green baize with its weave and the shadow around the rim.
func renderFelt(w, h int) *ebiten.Image {
	// the weave: the same speckle tile the original drew, laid at 55% opacity
	const tile = 160
	noise := make([][2]float64, tile*tile) // per pixel: white alpha, black alpha
	for i := 0; i < 2600; i++ {
		v := rand.Float64()
		white := v > .5
		a := rand.Float64() * .08
		if white {
			a = rand.Float64() * .05
		}
		x, y := int(rand.Float64()*tile), int(rand.Float64()*tile)
		hgt := 1 + rand.Float64()*2
		for k := 0; k < int(math.Ceil(hgt)); k++ {
			yy := (y + k) % tile
			cov := math.Min(1, hgt-float64(k))
			if white {
				noise[yy*tile+x][0] = 1 - (1-noise[yy*tile+x][0])*(1-a*cov)
			} else {
				noise[yy*tile+x][1] = 1 - (1-noise[yy*tile+x][1])*(1-a*cov)
			}
		}
	}
	stops := []float64{0, .4, .78, 1}
	cols := []color.RGBA{hex("#2f6b48"), hex("#245a3c"), hex("#173f29"), hex("#0f2c1c")}
	W, H := float64(w), float64(h)
	gx, gy := W*0.5, H*0.44
	rx, ry := math.Max(gx, W-gx)*math.Sqrt2, math.Max(gy, H-gy)*math.Sqrt2
	px := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			t := math.Hypot((fx-gx)/rx, (fy-gy)/ry)
			c := gradStops(t, stops, cols)
			r, gg, b := c[0], c[1], c[2]
			n := noise[(y%tile)*tile+x%tile]
			wa, ba := n[0]*.55, n[1]*.55
			r, gg, b = r+(255-r)*wa, gg+(255-gg)*wa, b+(255-b)*wa
			r, gg, b = r*(1-ba), gg*(1-ba), b*(1-ba)
			// inset 0 0 90px rgba(0,0,0,.55)
			d := -sdRR(fx-W/2, fy-H/2, W/2, H/2, 26)
			sh := .55 * math.Erfc(d/(45*math.Sqrt2)) * 0.5
			r, gg, b = r*(1-sh), gg*(1-sh), b*(1-sh)
			i := (y*w + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = uint8(r), uint8(gg), uint8(b), 255
		}
	}
	im := ebiten.NewImage(w, h)
	im.WritePixels(px)
	return im
}

// renderVeil is the dimmed backdrop behind panels.
func renderVeil(W, H int) *ebiten.Image {
	px := make([]byte, W*H*4)
	a0, a1 := rgba(20, 10, 6, .55), rgba(10, 5, 3, .86)
	rx, ry := float64(W)/2*math.Sqrt2, float64(H)/2*math.Sqrt2
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			t := math.Hypot((float64(x)-float64(W)/2)/rx, (float64(y)-float64(H)/2)/ry)
			c := lerpColor(a0, a1, clamp(t, 0, 1))
			af := float64(c.A) / 255
			i := (y*W + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = uint8(float64(c.R)*af), uint8(float64(c.G)*af), uint8(float64(c.B)*af), c.A
		}
	}
	im := ebiten.NewImage(W, H)
	im.WritePixels(px)
	return im
}

/* ---------------- candlelight (soft-light glow) and rounded clip ---------------- */

const glowShader = `//kage:unit pixels
package main

var Size vec2
var Opacity float
var Clip float

func softD(b float) float {
	if b <= 0.25 {
		return ((16*b-12)*b+4)*b
	}
	return sqrt(b)
}

func softLight(b, s float) float {
	if s <= 0.5 {
		return b - (1-2*s)*b*(1-b)
	}
	return b + (2*s-1)*(softD(b)-b)
}

func sdRR(p, hs vec2, r float) float {
	q := abs(p) - hs + r
	return length(max(q, 0)) + min(max(q.x, q.y), 0) - r
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	p := src - imageSrc0Origin()
	b := imageSrc0At(src)
	if Clip > 0 {
		d := sdRR(p - Size/2, Size/2, 26)
		return b * clamp(0.5-d, 0, 1)
	}
	// radial-gradient(ellipse at 50% 30%, ...) over a box 10% larger than the table on every side
	c := vec2(Size.x*0.5, -0.1*Size.y + 0.3*1.2*Size.y)
	rr := vec2(0.6*Size.x, max(c.y+0.1*Size.y, 1.1*Size.y-c.y)) * sqrt(2.0)
	t := length((p - c) / rr)
	var s vec4
	if t < 0.45 {
		s = mix(vec4(1, 200.0/255, 120.0/255, 0.55), vec4(1, 170.0/255, 90.0/255, 0.15), t/0.45)
	} else if t < 0.8 {
		s = mix(vec4(1, 170.0/255, 90.0/255, 0.15), vec4(0, 0, 0, 0.35), (t-0.45)/0.35)
	} else {
		s = vec4(0, 0, 0, 0.35)
	}
	a := s.a * Opacity
	out := vec3(softLight(b.r, s.r), softLight(b.g, s.g), softLight(b.b, s.b))
	return vec4(mix(b.rgb, out, a), 1)
}
`

var shGlow *ebiten.Shader

func initShaders() {
	s, err := ebiten.NewShader([]byte(glowShader))
	if err != nil {
		panic(err)
	}
	shGlow = s
}

// flicker follows the original 5-second candle keyframes.
func flicker(t float64) float64 {
	keys := []float64{0, .08, .12, .30, .47, .52, .70, .86, 1}
	vals := []float64{.9, .82, .95, .86, .97, .84, .93, .8, .9}
	f := math.Mod(t/5, 1)
	for i := 1; i < len(keys); i++ {
		if f <= keys[i] {
			k := (f - keys[i-1]) / (keys[i] - keys[i-1])
			return vals[i-1] + (vals[i]-vals[i-1])*easeCSS(k)
		}
	}
	return .9
}

/* ---------------- dust motes ---------------- */

type mote struct{ x, y, r, vx, vy, p float64 }

func newMotes() []*mote {
	var m []*mote
	for i := 0; i < 46; i++ {
		m = append(m, &mote{x: rand.Float64(), y: rand.Float64(), r: 0.6 + rand.Float64()*1.8, vx: (rand.Float64() - .5) * 0.00006, vy: -0.00002 - rand.Float64()*0.00005, p: rand.Float64() * 6})
	}
	return m
}

/* ---------------- per-frame animation ---------------- */

func (g *Game) animate(dt float64) {
	for _, m := range g.motes {
		m.x += m.vx * dt
		m.y += m.vy * dt
		m.p += dt * 0.0012
		if m.y < -0.02 {
			m.y, m.x = 1.02, rand.Float64()
		}
		if m.x < 0 {
			m.x = 1
		}
		if m.x > 1 {
			m.x = 0
		}
	}
	f := dt / 16.7
	t := float64(time.Now().UnixMilli())
	live := g.parts[:0]
	for i, p := range g.parts {
		p.x += p.vx * f
		p.y += p.vy * f
		p.life -= p.d * f
		switch p.kind {
		case 0:
			p.vy += 0.05 * f
			p.vx *= 0.98
			p.rot += 0.1 * f
		case 1:
			p.s += 0.6 * f
		case 2:
			p.vx += math.Sin(t/400+float64(i)) * 0.02
			p.rot += p.vr * f
			if p.y > g.Ht+20 {
				p.life = 0
			}
		}
		if p.life > 0 {
			live = append(live, p)
		}
	}
	g.parts = live
}

/* ---------------- drawing the table ---------------- */

func (g *Game) drawTable(screen *ebiten.Image) {
	now := time.Now()
	ox, oy := g.feltOrigin()
	w, h := int(math.Round(g.Wd)), int(math.Round(g.Ht))
	if g.layer == nil || g.layer.Bounds().Dx() != w || g.layer.Bounds().Dy() != h {
		g.layer = ebiten.NewImage(w, h)
		g.layer2 = ebiten.NewImage(w, h)
		g.feltBG = renderFelt(w, h)
	}
	L := g.layer
	L.DrawImage(g.feltBG, nil)

	// dust motes
	for _, m := range g.motes {
		a := 0.12 + math.Sin(m.p)*0.08
		vector.DrawFilledCircle(L, float32(m.x*g.Wd), float32(m.y*g.Ht), float32(m.r), rgbaPre(255, 230, 180, a), true)
	}
	// the doily under the discard pile
	ds := g.ch * 1.9
	{
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		s := ds / float64(imgDoily.Bounds().Dx())
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(g.Wd/2-ds/2, g.pileY()-ds/2+3)
		op.ColorScale.Scale(0, 0, 0, 0.35*0.92)
		L.DrawImage(imgDoily, op)
		op.ColorScale.Reset()
		op.GeoM.Translate(0, -3)
		op.ColorScale.ScaleAlpha(0.92)
		L.DrawImage(imgDoily, op)
	}
	for _, p := range g.players {
		g.drawSeat(L, p, now)
	}
	g.drawCards(L, now)

	// candlelight over everything on the table
	L2 := g.layer2
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = L
	op.Uniforms = map[string]any{"Size": []float32{float32(w), float32(h)}, "Opacity": float32(flicker(float64(now.UnixMilli()) / 1000)), "Clip": float32(0)}
	L2.DrawRectShader(w, h, shGlow, op)

	// prompt, effects, banner and version sit above the light
	if a := g.promptA.val(now); a > 0.01 {
		txt(L2, g.promptText, font(fItalic, 21), g.Wd/2, g.Ht-g.ch*1.32-34+14, .5, hex("#fbecc4"), &txtOpt{alpha: a, shadow: rgba(0, 0, 0, .8), sdy: 2})
	}
	g.drawParticles(L2)
	g.drawBanner(L2, now)
	txt(L2, "v"+version, font(fSC, 14), g.Wd-22, g.Ht-12-9, 1, rgba(240, 214, 142, .45), &txtOpt{alpha: 1, tracked: 1.4})

	// onto the window, clipped to the table's rounded corners
	op2 := &ebiten.DrawRectShaderOptions{}
	op2.Images[0] = L2
	op2.Uniforms = map[string]any{"Size": []float32{float32(w), float32(h)}, "Opacity": float32(0), "Clip": float32(1)}
	op2.GeoM.Translate(ox, oy)
	screen.DrawRectShader(w, h, shGlow, op2)
}

// rgbaPre converts straight alpha to the premultiplied colour vector helpers expect.
func rgbaPre(r, g, b uint8, a float64) color.RGBA {
	a = clamp(a, 0, 1)
	return color.RGBA{uint8(float64(r) * a), uint8(float64(g) * a), uint8(float64(b) * a), uint8(a * 255)}
}

/* ---------------- cards ---------------- */

type cardXf struct{ x, y, rot, sc, flip, lift float64 }

func (g *Game) cardXf(c *Card, now time.Time) cardXf {
	return cardXf{c.ax.val(now), c.ay.val(now), c.ar.val(now), c.as.val(now), c.flip.val(now), c.lift.val(now)}
}

// cardGeo places something drawn in card-local pixels (origin at the card's centre, card size
// cw x ch) onto the felt, following the card's rotation, scale, flip and lift.
func (g *Game) cardGeo(x cardXf, flipW float64) ebiten.GeoM {
	var m ebiten.GeoM
	m.Scale(flipW, 1)
	m.Translate(0, -0.16*g.ch*x.lift)
	m.Scale(x.sc, x.sc)
	m.Rotate(x.rot * math.Pi / 180)
	m.Translate(x.x, x.y)
	return m
}

func (g *Game) sortedTable() []*Card {
	cs := append([]*Card(nil), g.table...)
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].z < cs[j].z })
	return cs
}

func (g *Game) drawCards(dst *ebiten.Image, now time.Time) {
	cw, ch := math.Round(g.cw), math.Round(g.ch)
	rad := math.Round(g.cw * .065)
	for _, c := range g.sortedTable() {
		x := g.cardXf(c, now)
		fw := math.Abs(math.Cos(x.flip * math.Pi))
		if fw < 0.02 {
			continue
		}
		front := x.flip < 0.5
		drawSoft := func(spread, blur, ring, dy float64, col color.RGBA) {
			im := soft(cw, ch, rad, spread, blur, ring)
			m := float64(softPad(spread, blur, ring))
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Translate(-cw/2-m, -ch/2-m+dy)
			op.GeoM.Concat(g.cardGeo(x, fw))
			tint(&op.ColorScale, col)
			dst.DrawImage(im, op)
		}
		hovered := c.pick && c.hover
		switch {
		case c.glow:
			drawSoft(6, 22, 0, 0, rgba(255, 220, 120, .85))
			drawSoft(0, 0, 3, 0, hex("#fff3c4"))
		case c.isNew && c.L == lHand && c.owner == g.viewer && g.canSee():
			drawSoft(4, 18, 0, 0, rgba(140, 230, 160, .6))
			drawSoft(0, 0, 3, 0, hex("#a8e0b4"))
		case hovered:
			drawSoft(0, 24, 0, 10, rgba(0, 0, 0, .45))
			drawSoft(0, 0, 3, 0, cGold2)
		default:
			drawSoft(0, 14, 0, 6, rgba(0, 0, 0, .18))
			drawSoft(0, 4, 0, 2, rgba(0, 0, 0, .35))
		}
		im := imgBack
		if front {
			im = c.img
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-artW/2, -artH/2)
		op.GeoM.Scale(g.cw/artW, g.ch/artH)
		op.GeoM.Concat(g.cardGeo(x, fw))
		dst.DrawImage(im, op)
	}
}

// cardAt returns the topmost card under (mx, my) in felt coordinates.
func (g *Game) cardAt(mx, my float64) *Card {
	now := time.Now()
	cs := g.sortedTable()
	for i := len(cs) - 1; i >= 0; i-- {
		c := cs[i]
		x := g.cardXf(c, now)
		// into the card's own frame
		dx, dy := mx-x.x, my-x.y
		a := -x.rot * math.Pi / 180
		lx, ly := (dx*math.Cos(a)-dy*math.Sin(a))/x.sc, (dx*math.Sin(a)+dy*math.Cos(a))/x.sc
		hw, hh := g.cw/2, g.ch/2
		if math.Abs(lx) <= hw && ly >= -hh-0.16*g.ch*x.lift && ly <= hh {
			return c
		}
	}
	return nil
}

/* ---------------- seats ---------------- */

const camSize = 76.0

func (g *Game) seatBox(p *Player) (whoW, whoH, totalW float64) {
	nm := textW(g.nm(p, g.viewer), font(fSC, 18))
	ct := textW(g.countText(p), font(fItalic, 14))
	whoW, whoH = math.Max(nm, ct)+24+2, 50
	totalW = whoW
	if p.hasFace {
		totalW += camSize + 10
	}
	return
}

func (g *Game) drawSeat(dst *ebiten.Image, p *Player, now time.Time) {
	whoW, whoH, totalW := g.seatBox(p)
	h := whoH
	if p.hasFace {
		h = camSize
	}
	left := p.sx - totalW/2
	x := left
	if p.hasFace {
		t := p.turn.val(now)
		cx, cy := x+camSize/2, p.sy
		s := 1 + 0.06*t
		cs := camSize * s
		shadowRR(dst, cx-cs/2, cy-cs/2, cs, cs, cs/2, 0, 4, 12, 0, rgba(0, 0, 0, .5))
		if t > 0.01 {
			shadowRR(dst, cx-cs/2, cy-cs/2, cs, cs, cs/2, 0, 0, 26, 6, rgba(255, 214, 120, .7*t))
			im := soft(math.Round(cs), math.Round(cs), math.Round(cs/2), 0, 0, 4)
			m := float64(softPad(0, 0, 4))
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Translate(cx-math.Round(cs)/2-m, cy-math.Round(cs)/2-m)
			tint(&op.ColorScale, withAlpha(cGold2, t))
			dst.DrawImage(im, op)
		}
		cam := cameoFor(p.face)
		b := cam.Bounds()
		var cm colorm.ColorM
		if p.out {
			cm.ChangeHSV(0, 0.4, 0.8)
		}
		op := &colorm.DrawImageOptions{Filter: ebiten.FilterLinear}
		sc := cs / float64(b.Dx())
		op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
		op.GeoM.Scale(sc, sc)
		op.GeoM.Translate(cx, cy)
		colorm.DrawImage(dst, cam, cm, op)
		x += camSize + 10
	}
	// name plate
	wy := p.sy - whoH/2
	pl := rrPath(x, wy, whoW, whoH, 6)
	fillGrad(dst, pl, rgba(40, 22, 12, .92), rgba(28, 14, 8, .92), wy, wy+whoH)
	stroke(dst, rrPath(x+0.5, wy+0.5, whoW-1, whoH-1, 6), 1, cBrass)
	txt(dst, g.nm(p, g.viewer), font(fSC, 18), x+13, wy+4+12, 0, cGold2, nil)
	txt(dst, g.countText(p), font(fItalic, 14), x+13, wy+4+24+9, 0, hex("#d8c29a"), nil)

	// speech bubble
	if !p.sayUntil.IsZero() && now.After(p.sayUntil) {
		p.sayUntil = time.Time{}
		p.sayOn.set(0, now, 0.2, easeCSS)
	}
	if a := p.sayOn.val(now); a > 0.01 && p.say != "" {
		f := font(fItalic, 17)
		bw, bh := textW(p.say, f)+28, 17*1.35+14
		s := 0.6 + 0.4*a
		var by float64 // bubble centre
		up := g.seatLetter(p) != 'S'
		if up {
			by = p.sy - h/2 - 10 - bh/2
		} else {
			by = p.sy + h/2 + 10 + bh/2
		}
		img := g.bubbleImg(p.say, bw, bh, up)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		bb := img.Bounds()
		op.GeoM.Translate(-float64(bb.Dx())/2, -float64(bb.Dy())/2)
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(math.Round(left+totalW/2), math.Round(by))
		op.ColorScale.ScaleAlpha(float32(a))
		dst.DrawImage(img, op)
	}
}

// bubbleImg renders a speech bubble once per line of dialogue (so it can scale and fade as one).
func (g *Game) bubbleImg(s string, bw, bh float64, up bool) *ebiten.Image {
	key := s
	if !up {
		key = "v" + s
	}
	if im, ok := g.bubbles[key]; ok {
		return im
	}
	pad := 28.0
	W, H := int(bw+2*pad), int(bh+2*pad)
	im := ebiten.NewImage(W, H)
	x, y := pad, pad
	shadowRR(im, x, y, bw, bh, 14, 0, 6, 18, 0, rgba(0, 0, 0, .4))
	fill(im, rrPath(x, y, bw, bh, 14), cParch)
	tri := &vector.Path{}
	cx := float32(x + bw/2)
	if up {
		ty := float32(y + bh)
		tri.MoveTo(cx-8, ty-1)
		tri.LineTo(cx+8, ty-1)
		tri.LineTo(cx, ty+8)
	} else {
		ty := float32(y)
		tri.MoveTo(cx-8, ty+1)
		tri.LineTo(cx+8, ty+1)
		tri.LineTo(cx, ty-8)
	}
	tri.Close()
	fill(im, tri, cParch)
	txt(im, s, font(fItalic, 17), x+bw/2, y+bh/2, .5, cInk, nil)
	if len(g.bubbles) > 80 {
		g.bubbles = map[string]*ebiten.Image{}
	}
	g.bubbles[key] = im
	return im
}

/* ---------------- particles and banner ---------------- */

func (g *Game) drawParticles(dst *ebiten.Image) {
	for _, p := range g.parts {
		switch p.kind {
		case 0: // four-pointed sparkle
			a := math.Max(0, p.life)
			path := &vector.Path{}
			for j := 0; j < 4; j++ {
				r := p.rot + float64(j+1)*math.Pi/2
				pt := func(x, y float64) (float32, float32) {
					return float32(p.x + x*math.Cos(r) - y*math.Sin(r)), float32(p.y + x*math.Sin(r) + y*math.Cos(r))
				}
				x0, y0 := pt(0, 0)
				x1, y1 := pt(p.s*0.5, p.s*0.5)
				x2, y2 := pt(0, p.s*2.2)
				x3, y3 := pt(-p.s*0.5, p.s*0.5)
				path.MoveTo(x0, y0)
				path.LineTo(x1, y1)
				path.LineTo(x2, y2)
				path.LineTo(x3, y3)
				path.Close()
			}
			fill(dst, path, rgba(0xff, 0xe9, 0xa8, a))
		case 1:
			vector.DrawFilledCircle(dst, float32(p.x), float32(p.y), float32(p.s), rgbaPre(p.r, p.gg, p.b, math.Max(0, p.life)*0.34), true)
		case 2:
			a := math.Min(1, p.life*2)
			path := &vector.Path{}
			for k := 0; k <= 18; k++ {
				t := float64(k) / 18 * 2 * math.Pi
				ex, ey := p.s*math.Cos(t), p.s*0.55*math.Sin(t)
				x, y := float32(p.x+ex*math.Cos(p.rot)-ey*math.Sin(p.rot)), float32(p.y+ex*math.Sin(p.rot)+ey*math.Cos(p.rot))
				if k == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			path.Close()
			fill(dst, path, rgba(p.r, p.gg, p.b, clamp(a, 0, 1)))
		}
	}
}

func (g *Game) drawBanner(dst *ebiten.Image, now time.Time) {
	if !g.bannerUntil.IsZero() && now.After(g.bannerUntil) {
		g.bannerUntil = time.Time{}
		g.bannerA.set(0, now, 0.3, easeCSS)
		g.bannerS.set(0, now, 0.35, easeBanner)
	}
	a := g.bannerA.val(now)
	if a < 0.01 || g.bannerTitle == "" {
		return
	}
	key := g.bannerTitle + "|" + g.bannerSub
	if g.bannerKey != key {
		g.bannerKey = key
		ft, fs := font(fSC, 38), font(fItalic, 20)
		bw := math.Max(textW(g.bannerTitle, ft), textW(g.bannerSub, fs)) + 68
		bh := 12 + 38*1.2 + 20*1.35 + 14.0
		pad := 50.0
		im := ebiten.NewImage(int(bw+2*pad), int(bh+2*pad))
		x, y := pad, pad
		shadowRR(im, x, y, bw, bh, 10, 0, 14, 40, 0, rgba(0, 0, 0, .6))
		shadowRR(im, x, y, bw, bh, 10, 0, 0, 0, 5, rgba(48, 24, 14, .6))
		fillGrad(im, rrPath(x, y, bw, bh, 10), rgba(48, 24, 14, .94), rgba(30, 14, 8, .94), y, y+bh)
		stroke(im, rrPath(x+1, y+1, bw-2, bh-2, 9), 2, hex("#c9a24c"))
		txt(im, g.bannerTitle, ft, x+bw/2, y+12+38*0.6, .5, hex("#fff3d0"), &txtOpt{shadow: rgba(0, 0, 0, .9), sdy: 2})
		txt(im, g.bannerSub, fs, x+bw/2, y+12+38*1.2+20*0.68, .5, hex("#e8d3a6"), nil)
		g.bannerImg = im
	}
	s := 0.7 + 0.3*g.bannerS.val(now)
	b := g.bannerImg.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(g.Wd/2, g.Ht*0.40)
	op.ColorScale.ScaleAlpha(float32(a))
	dst.DrawImage(g.bannerImg, op)
}
