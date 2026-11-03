// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// The guests' cameo portraits, drawn in Go in the same hand as the cards (a port of the
// original portrait code), and the random faces, names and manners for single player.
package main

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

/* ---------------- a little canvas, so the portrait code reads like the original ---------------- */

type cstate struct {
	m          ebiten.GeoM
	fill, strk color.RGBA
	lw         float64
}

type canvas2d struct {
	dst              *ebiten.Image
	st               cstate
	stack            []cstate
	path             *vector.Path
	cx, cy, sx, sy   float64 // current point and subpath start (device space)
	hasCur, needMove bool
}

func newCanvas(dst *ebiten.Image) *canvas2d {
	return &canvas2d{dst: dst, st: cstate{fill: color.RGBA{0, 0, 0, 255}, strk: color.RGBA{0, 0, 0, 255}, lw: 1}, path: &vector.Path{}}
}

func (c *canvas2d) save()    { c.stack = append(c.stack, c.st) }
func (c *canvas2d) restore() { c.st = c.stack[len(c.stack)-1]; c.stack = c.stack[:len(c.stack)-1] }
func (c *canvas2d) translate(x, y float64) {
	var m ebiten.GeoM
	m.Translate(x, y)
	m.Concat(c.st.m)
	c.st.m = m
}
func (c *canvas2d) rotate(a float64) {
	var m ebiten.GeoM
	m.Rotate(a)
	m.Concat(c.st.m)
	c.st.m = m
}
func (c *canvas2d) scale(x, y float64) {
	var m ebiten.GeoM
	m.Scale(x, y)
	m.Concat(c.st.m)
	c.st.m = m
}
func (c *canvas2d) unit() float64 {
	return math.Hypot(c.st.m.Element(0, 0), c.st.m.Element(1, 0))
}
func (c *canvas2d) pt(x, y float64) (float64, float64) { return c.st.m.Apply(x, y) }

func (c *canvas2d) beginPath() { c.path = &vector.Path{}; c.hasCur, c.needMove = false, false }

func (c *canvas2d) moveTo(x, y float64) {
	px, py := c.pt(x, y)
	c.path.MoveTo(float32(px), float32(py))
	c.cx, c.cy, c.sx, c.sy = px, py, px, py
	c.hasCur, c.needMove = true, false
}

func (c *canvas2d) ensure(px, py float64) {
	if !c.hasCur {
		c.path.MoveTo(float32(px), float32(py))
		c.cx, c.cy, c.sx, c.sy = px, py, px, py
		c.hasCur = true
		return
	}
	if c.needMove {
		c.path.MoveTo(float32(c.sx), float32(c.sy))
		c.cx, c.cy = c.sx, c.sy
		c.needMove = false
	}
}

func (c *canvas2d) lineTo(x, y float64) {
	px, py := c.pt(x, y)
	c.ensure(px, py)
	c.path.LineTo(float32(px), float32(py))
	c.cx, c.cy = px, py
}

func (c *canvas2d) quadTo(x1, y1, x, y float64) {
	ax, ay := c.pt(x1, y1)
	px, py := c.pt(x, y)
	c.ensure(ax, ay)
	c.path.QuadTo(float32(ax), float32(ay), float32(px), float32(py))
	c.cx, c.cy = px, py
}

func (c *canvas2d) bezTo(x1, y1, x2, y2, x, y float64) {
	ax, ay := c.pt(x1, y1)
	bx, by := c.pt(x2, y2)
	px, py := c.pt(x, y)
	c.ensure(ax, ay)
	c.path.CubicTo(float32(ax), float32(ay), float32(bx), float32(by), float32(px), float32(py))
	c.cx, c.cy = px, py
}

func (c *canvas2d) closePath() {
	if c.hasCur {
		c.path.Close()
		c.cx, c.cy = c.sx, c.sy
		c.needMove = true
	}
}

// ellipse follows the canvas rules: a line joins the current point to the arc's start.
func (c *canvas2d) ellipse(x, y, rx, ry, rot, a0, a1 float64, ccw bool) {
	const tau = 2 * math.Pi
	var sweep float64
	if !ccw {
		if a1-a0 >= tau {
			sweep = tau
		} else {
			sweep = math.Mod(a1-a0, tau)
			if sweep < 0 {
				sweep += tau
			}
		}
	} else {
		if a0-a1 >= tau {
			sweep = -tau
		} else {
			s := math.Mod(a0-a1, tau)
			if s < 0 {
				s += tau
			}
			sweep = -s
		}
	}
	at := func(a float64) (float64, float64) {
		ex, ey := rx*math.Cos(a), ry*math.Sin(a)
		return c.pt(x+ex*math.Cos(rot)-ey*math.Sin(rot), y+ex*math.Sin(rot)+ey*math.Cos(rot))
	}
	n := int(math.Max(8, math.Ceil(math.Abs(sweep)*math.Max(rx, ry)*c.unit()/2.5)))
	px, py := at(a0)
	if c.hasCur && !c.needMove {
		c.path.LineTo(float32(px), float32(py))
	} else {
		c.path.MoveTo(float32(px), float32(py))
		c.sx, c.sy = px, py
		c.hasCur, c.needMove = true, false
	}
	for i := 1; i <= n; i++ {
		qx, qy := at(a0 + sweep*float64(i)/float64(n))
		c.path.LineTo(float32(qx), float32(qy))
		c.cx, c.cy = qx, qy
	}
}

func (c *canvas2d) arc(x, y, r, a0, a1 float64) { c.ellipse(x, y, r, r, 0, a0, a1, false) }
func (c *canvas2d) circle(x, y, r float64)      { c.arc(x, y, r, 0, 2*math.Pi) }

func (c *canvas2d) rrect(x, y, w, h, r float64) {
	c.moveTo(x+r, y)
	c.lineTo(x+w-r, y)
	c.ellipse(x+w-r, y+r, r, r, 0, -math.Pi/2, 0, false)
	c.lineTo(x+w, y+h-r)
	c.ellipse(x+w-r, y+h-r, r, r, 0, 0, math.Pi/2, false)
	c.lineTo(x+r, y+h)
	c.ellipse(x+r, y+h-r, r, r, 0, math.Pi/2, math.Pi, false)
	c.lineTo(x, y+r)
	c.ellipse(x+r, y+r, r, r, 0, math.Pi, 3*math.Pi/2, false)
	c.closePath()
}

func (c *canvas2d) rect(x, y, w, h float64) {
	c.moveTo(x, y)
	c.lineTo(x+w, y)
	c.lineTo(x+w, y+h)
	c.lineTo(x, y+h)
	c.closePath()
}

func (c *canvas2d) fill()   { fill(c.dst, c.path, c.st.fill) }
func (c *canvas2d) stroke() { stroke(c.dst, c.path, c.st.lw*c.unit(), c.st.strk) }
func (c *canvas2d) fillRect(x, y, w, h float64) {
	p := c.path
	c.beginPath()
	c.rect(x, y, w, h)
	c.fill()
	c.path = p
}
func (c *canvas2d) strokeRect(x, y, w, h float64) {
	p := c.path
	c.beginPath()
	c.rect(x, y, w, h)
	c.stroke()
	c.path = p
}

/* ---------------- the look of a guest ---------------- */

type Look struct {
	Lady, Stern                                   bool
	Skin, Hair, Brow, MustacheColor               color.RGBA
	HairStyle                                     string // short, bun, updo, ringlets
	Coat, Dark, Accent, Accent2, HatColor, Shawl  color.RGBA
	Collar                                        string // cravat, lace, buttons, military, shawl
	Pearls, Sleeves, Mustache, Muttonchops, Beard bool
	Monocle, Spectacles                           bool
	Hat                                           string // top, feather, bonnet, pillbox, military, lacecap, ""
}

var (
	pInk   = hex("#2b1d14")
	pGold  = hex("#b8913f")
	pWhite = hex("#f7f1e3")
)

func orC(c, d color.RGBA) color.RGBA {
	if c.A == 0 {
		return d
	}
	return c
}

// bust draws the head and shoulders, origin at the bottom centre (ported from the original game's portrait code).
func bust(c *canvas2d, o Look) {
	const tau = 2 * math.Pi
	skin, hair := orC(o.Skin, hex("#f2d4b6")), orC(o.Hair, hex("#5a3a22"))
	c.save()
	c.st.strk, c.st.lw = pInk, 2.6
	setInk := func() { c.st.strk, c.st.lw = pInk, 2.6 }
	// back hair
	c.st.fill = hair
	if o.HairStyle == "updo" || o.HairStyle == "ringlets" || o.HairStyle == "bun" {
		c.beginPath()
		c.circle(0, -150, 17)
		c.fill()
		c.stroke()
	}
	if o.HairStyle == "ringlets" {
		for _, sx := range []float64{-1, 1} {
			for i := 0; i < 4; i++ {
				c.beginPath()
				c.circle(sx*(27+float64(i%2)*3), -118+float64(i)*11, 6.5)
				c.fill()
				c.stroke()
			}
		}
	}
	// shoulders
	c.beginPath()
	c.moveTo(-80, 0)
	c.bezTo(-78, -42, -54, -60, -22, -64)
	c.lineTo(22, -64)
	c.bezTo(54, -60, 78, -42, 80, 0)
	c.closePath()
	c.st.fill = o.Coat
	c.fill()
	c.stroke()
	if o.Sleeves {
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.ellipse(sx*62, -36, 22, 28, sx*0.3, 0, tau, false)
			c.st.fill = o.Coat
			c.fill()
			c.stroke()
			c.beginPath()
			c.moveTo(sx*50, -56)
			c.quadTo(sx*64, -40, sx*52, -14)
			c.st.strk = rgba(0, 0, 0, .25)
			c.stroke()
			c.st.strk = pInk
		}
	}
	// neck
	c.st.fill = skin
	c.beginPath()
	c.moveTo(-11, -88)
	c.lineTo(-12, -60)
	c.lineTo(12, -60)
	c.lineTo(11, -88)
	c.closePath()
	c.fill()
	switch o.Collar {
	case "cravat":
		c.st.fill = pWhite
		c.beginPath()
		c.moveTo(-20, -64)
		c.lineTo(0, -16)
		c.lineTo(20, -64)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = o.Dark
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.moveTo(sx*22, -64)
			c.lineTo(sx*4, -16)
			c.lineTo(sx*36, -30)
			c.closePath()
			c.fill()
			c.stroke()
		}
		c.st.fill = o.Accent
		c.beginPath()
		c.moveTo(0, -56)
		c.lineTo(-14, -64)
		c.lineTo(-14, -48)
		c.closePath()
		c.moveTo(0, -56)
		c.lineTo(14, -64)
		c.lineTo(14, -48)
		c.closePath()
		c.fill()
		c.stroke()
		c.beginPath()
		c.circle(0, -56, 4)
		c.fill()
		c.stroke()
		for _, y := range []float64{-34, -22} {
			c.beginPath()
			c.circle(0, y, 2.6)
			c.fill()
		}
	case "lace":
		c.st.fill = hex("#fbf7ee")
		c.beginPath()
		c.moveTo(-24, -60)
		c.lineTo(-17, -92)
		c.lineTo(17, -92)
		c.lineTo(24, -60)
		c.closePath()
		c.fill()
		c.stroke()
		for i := -3; i <= 3; i++ {
			c.beginPath()
			c.arc(float64(i)*7, -92, 3.6, math.Pi, 0)
			c.fill()
			c.stroke()
		}
		c.st.strk, c.st.lw = rgba(80, 60, 40, .35), 1
		for y := -86.0; y < -62; y += 6 {
			c.beginPath()
			c.moveTo(-18, y)
			c.lineTo(18, y)
			c.stroke()
		}
		setInk()
		c.st.fill = o.Accent
		c.beginPath()
		c.ellipse(0, -56, 9, 11, 0, 0, tau, false)
		c.fill()
		c.stroke()
		c.st.fill = hex("#f3e6d6")
		c.beginPath()
		c.ellipse(0, -56, 5, 7, 0, 0, tau, false)
		c.fill()
		if o.Pearls {
			c.st.fill = hex("#fffaf0")
			for i := 0; i < 9; i++ {
				a := math.Pi * (0.15 + float64(i)*0.0875)
				c.beginPath()
				c.circle(math.Cos(a)*34, -64+math.Sin(a)*26, 3.2)
				c.fill()
				c.stroke()
			}
		}
	case "buttons":
		c.st.fill = o.Dark
		c.beginPath()
		c.moveTo(-20, -64)
		c.lineTo(20, -64)
		c.lineTo(18, -54)
		c.lineTo(-18, -54)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = o.Accent, 2
		c.beginPath()
		c.moveTo(-18, -58)
		c.lineTo(18, -58)
		c.stroke()
		setInk()
		c.st.fill = o.Accent
		for _, y := range []float64{-42, -28, -14} {
			for _, sx := range []float64{-1, 1} {
				c.beginPath()
				c.circle(sx*11, y, 3.6)
				c.fill()
				c.stroke()
			}
		}
		c.st.strk, c.st.lw = o.Accent, 2
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.moveTo(sx*40, -54)
			c.lineTo(sx*58, -50)
			c.stroke()
		}
		setInk()
	case "military":
		c.st.fill = o.Dark
		c.beginPath()
		c.moveTo(-18, -64)
		c.lineTo(18, -64)
		c.lineTo(16, -52)
		c.lineTo(-16, -52)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = o.Accent
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.rrect(sx*58-16, -60, 32, 10, 4)
			c.fill()
			c.stroke()
			for i := 0; i < 4; i++ {
				c.beginPath()
				c.moveTo(sx*58-14+float64(i)*9, -50)
				c.lineTo(sx*58-14+float64(i)*9, -42)
				c.st.strk = o.Accent
				c.stroke()
			}
			c.st.strk = pInk
		}
		for _, y := range []float64{-40, -26, -12} {
			c.beginPath()
			c.circle(0, y, 3.4)
			c.fill()
			c.stroke()
		}
		c.st.fill = hex("#d9b44a")
		c.beginPath()
		c.circle(-34, -30, 7)
		c.fill()
		c.stroke()
		c.st.fill = hex("#b02a2a")
		c.fillRect(-38, -44, 8, 8)
	case "shawl":
		c.st.fill = o.Shawl
		c.beginPath()
		c.moveTo(-82, -6)
		c.bezTo(-74, -50, -40, -66, -18, -66)
		c.lineTo(0, -20)
		c.lineTo(18, -66)
		c.bezTo(40, -66, 74, -50, 82, -6)
		c.lineTo(40, 0)
		c.lineTo(0, -10)
		c.lineTo(-40, 0)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = rgba(60, 40, 60, .4), 1.2
		for i := 0; i < 14; i++ {
			x := -78 + float64(i)*12
			c.beginPath()
			c.moveTo(x, -4+math.Abs(x)*0.02)
			c.lineTo(x-2, 6)
			c.stroke()
		}
		setInk()
		c.st.fill = hex("#fbf7ee")
		c.beginPath()
		c.moveTo(-14, -62)
		c.lineTo(-12, -86)
		c.lineTo(12, -86)
		c.lineTo(14, -62)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = o.Accent
		c.beginPath()
		c.ellipse(0, -60, 7, 8, 0, 0, tau, false)
		c.fill()
		c.stroke()
	}
	// head
	hy := -118.0
	c.st.fill = skin
	c.beginPath()
	c.ellipse(-27, hy+2, 5, 8, 0, 0, tau, false)
	c.ellipse(27, hy+2, 5, 8, 0, 0, tau, false)
	c.fill()
	c.stroke()
	c.beginPath()
	c.ellipse(0, hy, 27, 33, 0, 0, tau, false)
	c.fill()
	c.stroke()
	// front hair
	c.st.fill = hair
	if o.HairStyle == "short" {
		c.beginPath()
		c.ellipse(0, hy-14, 28, 22, 0, math.Pi, 0, false)
		c.quadTo(14, hy-22, -4, hy-18)
		c.quadTo(-18, hy-14, -28, hy-6)
		c.closePath()
		c.fill()
		c.stroke()
	} else if o.HairStyle == "updo" || o.HairStyle == "ringlets" || o.HairStyle == "bun" {
		c.beginPath()
		c.moveTo(-29, hy+6)
		c.bezTo(-34, hy-30, -14, hy-40, 0, hy-36)
		c.bezTo(14, hy-40, 34, hy-30, 29, hy+6)
		c.bezTo(22, hy-10, 14, hy-24, 0, hy-22)
		c.bezTo(-14, hy-24, -22, hy-10, -29, hy+6)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = rgba(255, 255, 255, .18), 1.2
		c.beginPath()
		c.moveTo(-18, hy-26)
		c.quadTo(-6, hy-34, 6, hy-30)
		c.stroke()
		setInk()
	}
	if o.Muttonchops {
		for _, sx := range []float64{-1, 1} {
			c.beginPath()
			c.moveTo(sx*26, hy-10)
			c.quadTo(sx*30, hy+18, sx*14, hy+24)
			c.quadTo(sx*18, hy+6, sx*22, hy-8)
			c.closePath()
			c.fill()
			c.stroke()
		}
	}
	if o.Beard {
		c.beginPath()
		c.moveTo(-24, hy+8)
		c.quadTo(-22, hy+46, 0, hy+50)
		c.quadTo(22, hy+46, 24, hy+8)
		c.quadTo(12, hy+24, 0, hy+22)
		c.quadTo(-12, hy+24, -24, hy+8)
		c.fill()
		c.stroke()
	}
	// face
	c.st.fill = rgba(214, 110, 100, .28)
	c.beginPath()
	c.circle(-15, hy+9, 6.5)
	c.circle(15, hy+9, 6.5)
	c.fill()
	c.st.fill = pInk
	ey := 3.0
	if o.Lady {
		ey = 3.4
	}
	c.beginPath()
	c.ellipse(-10, hy-3, 2.6, ey, 0, 0, tau, false)
	c.ellipse(10, hy-3, 2.6, ey, 0, 0, tau, false)
	c.fill()
	c.st.lw, c.st.strk = 2, orC(o.Brow, hair)
	st, st1 := 0.0, 1.0
	if o.Stern {
		st, st1 = 2, 0
	}
	c.beginPath()
	c.moveTo(-15, hy-11+st)
	c.lineTo(-5, hy-12-st1)
	c.moveTo(15, hy-11+st)
	c.lineTo(5, hy-12-st1)
	c.stroke()
	c.st.strk, c.st.lw = pInk, 1.8
	c.beginPath()
	c.moveTo(1, hy)
	c.quadTo(-3, hy+9, 1, hy+11)
	c.stroke()
	if o.Lady {
		c.st.fill = hex("#b4423f")
		rx, ry := 6.0, 2.8
		if o.Stern {
			rx, ry = 5, 1.8
		}
		c.beginPath()
		c.ellipse(0, hy+20, rx, ry, 0, 0, tau, false)
		c.fill()
	} else if !o.Mustache {
		c.beginPath()
		c.moveTo(-6, hy+19)
		c.quadTo(0, hy+22, 6, hy+19)
		c.stroke()
	}
	if o.Mustache {
		c.st.fill, c.st.strk, c.st.lw = orC(o.MustacheColor, hair), pInk, 1.6
		c.beginPath()
		c.moveTo(0, hy+13)
		c.bezTo(-8, hy+10, -18, hy+14, -26, hy+8)
		c.bezTo(-22, hy+20, -8, hy+20, 0, hy+17)
		c.bezTo(8, hy+20, 22, hy+20, 26, hy+8)
		c.bezTo(18, hy+14, 8, hy+10, 0, hy+13)
		c.fill()
		c.stroke()
	}
	setInk()
	if o.Monocle {
		c.st.strk, c.st.lw = pGold, 2.2
		c.beginPath()
		c.circle(10, hy-3, 8)
		c.stroke()
		c.beginPath()
		c.moveTo(17, hy+1)
		c.quadTo(30, hy+30, 22, hy+52)
		c.stroke()
		setInk()
	}
	if o.Spectacles {
		c.st.strk, c.st.lw = hex("#6d5a48"), 2
		c.beginPath()
		c.circle(-10, hy-3, 7.5)
		c.moveTo(-2.5, hy-3)
		c.arc(10, hy-3, 7.5, math.Pi, math.Pi*3)
		c.moveTo(-2.5, hy-4)
		c.lineTo(2.5, hy-4)
		c.moveTo(-17.5, hy-4)
		c.lineTo(-26, hy-7)
		c.moveTo(17.5, hy-4)
		c.lineTo(26, hy-7)
		c.stroke()
		setInk()
	}
	// hats
	switch o.Hat {
	case "top":
		c.st.fill = hex("#1d1712")
		c.beginPath()
		c.moveTo(-23, hy-30)
		c.lineTo(-25, hy-86)
		c.quadTo(0, hy-92, 25, hy-86)
		c.lineTo(23, hy-30)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = orC(o.Accent2, o.Coat)
		c.fillRect(-23, hy-42, 46, 9)
		c.strokeRect(-23, hy-42, 46, 9)
		c.st.fill = hex("#1d1712")
		c.beginPath()
		c.ellipse(0, hy-30, 40, 7, 0, 0, tau, false)
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = rgba(255, 255, 255, .15), 3
		c.beginPath()
		c.moveTo(-14, hy-82)
		c.lineTo(-13, hy-46)
		c.stroke()
		setInk()
	case "feather":
		c.save()
		c.translate(10, hy-36)
		c.rotate(0.22)
		c.st.fill = orC(o.HatColor, o.Dark)
		c.beginPath()
		c.ellipse(0, 4, 30, 7, 0, 0, tau, false)
		c.fill()
		c.stroke()
		c.beginPath()
		c.ellipse(0, -4, 17, 12, 0, math.Pi, 0, false)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = o.Accent
		c.fillRect(-17, -4, 34, 5)
		c.st.fill = hex("#f6efe2")
		c.beginPath()
		c.moveTo(10, -6)
		c.bezTo(30, -36, 54, -34, 52, -16)
		c.bezTo(40, -24, 28, -14, 10, -6)
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = rgba(120, 100, 80, .6), 1
		for i := 0; i < 6; i++ {
			c.beginPath()
			c.moveTo(16+float64(i)*6, -10-float64(i)*2.5)
			c.lineTo(20+float64(i)*6, -20-float64(i)*2)
			c.stroke()
		}
		c.restore()
	case "bonnet":
		c.st.fill = orC(o.HatColor, hex("#e9d9b8"))
		c.beginPath()
		c.moveTo(-34, hy+8)
		c.bezTo(-42, hy-46, 42, hy-46, 34, hy+8)
		c.bezTo(26, hy-26, -26, hy-26, -34, hy+8)
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = o.Accent, 3
		c.beginPath()
		c.moveTo(-30, hy+4)
		c.quadTo(-22, hy+30, -6, hy+36)
		c.moveTo(30, hy+4)
		c.quadTo(22, hy+30, 6, hy+36)
		c.stroke()
		c.st.fill = o.Accent
		c.beginPath()
		c.circle(0, hy+36, 5)
		c.fill()
		setInk()
		c.st.fill = hex("#e46b7a")
		for _, p := range [][2]float64{{-26, hy - 22}, {-20, hy - 30}} {
			c.beginPath()
			c.circle(p[0], p[1], 5)
			c.fill()
			c.stroke()
		}
	case "pillbox":
		c.st.fill = o.Coat
		c.beginPath()
		c.rrect(-18, hy-52, 36, 20, 4)
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = o.Accent, 2.5
		c.beginPath()
		c.moveTo(-18, hy-36)
		c.lineTo(18, hy-36)
		c.moveTo(-18, hy-48)
		c.lineTo(18, hy-48)
		c.stroke()
		c.beginPath()
		c.moveTo(-26, hy-4)
		c.quadTo(-20, hy+30, 0, hy+32)
		c.stroke()
		setInk()
		c.st.fill = o.Accent
		c.beginPath()
		c.circle(0, hy-42, 3)
		c.fill()
	case "military":
		c.st.fill = o.Coat
		c.beginPath()
		c.moveTo(-26, hy-28)
		c.lineTo(-28, hy-52)
		c.quadTo(0, hy-62, 28, hy-52)
		c.lineTo(26, hy-28)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.fill = hex("#1d1712")
		c.beginPath()
		c.moveTo(-28, hy-28)
		c.quadTo(0, hy-14, 30, hy-26)
		c.lineTo(26, hy-32)
		c.lineTo(-26, hy-32)
		c.closePath()
		c.fill()
		c.stroke()
		c.st.strk, c.st.lw = o.Accent, 3
		c.beginPath()
		c.moveTo(-26, hy-36)
		c.lineTo(26, hy-36)
		c.stroke()
		setInk()
		c.st.fill = o.Accent
		c.beginPath()
		c.circle(0, hy-46, 6)
		c.fill()
		c.stroke()
	case "lacecap":
		c.st.fill = hex("#fbf8f0")
		c.beginPath()
		c.ellipse(0, hy-26, 30, 18, 0, math.Pi, 0, false)
		c.closePath()
		c.fill()
		c.stroke()
		for i := 0; i < 11; i++ {
			a := math.Pi + float64(i)/10*math.Pi
			c.beginPath()
			c.circle(math.Cos(a)*31, hy-26+math.Sin(a)*19, 5)
			c.fill()
			c.stroke()
		}
		c.st.fill = hex("#c9a7c9")
		c.beginPath()
		c.moveTo(-24, hy-30)
		c.lineTo(-34, hy-18)
		c.lineTo(-28, hy-14)
		c.closePath()
		c.fill()
		c.stroke()
		c.beginPath()
		c.moveTo(24, hy-30)
		c.lineTo(34, hy-18)
		c.lineTo(28, hy-14)
		c.closePath()
		c.fill()
		c.stroke()
	}
	c.restore()
}

// renderCameo draws a round gilt-framed portrait (ported from the original game's cameo code).
func renderCameo(o Look, size int) *ebiten.Image {
	S := float64(size)
	r := S / 2
	out := ebiten.NewImage(size, size)
	// parchment background with a soft radial light
	px := make([]byte, size*size*4)
	in, edge := hex("#f3e6c6"), hex("#c9b083")
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := float64(x)+.5, float64(y)+.5
			d := math.Hypot(fx-r, fy-r)
			a := clamp(r-2-d+.5, 0, 1)
			// createRadialGradient(r, .8r, .1r, r, r, r)
			t := clamp((math.Hypot(fx-r, fy-r*0.8)-0.1*r)/(0.9*r), 0, 1)
			c := lerpColor(in, edge, t)
			i := (y*size + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = uint8(float64(c.R)*a), uint8(float64(c.G)*a), uint8(float64(c.B)*a), uint8(255*a)
		}
	}
	out.WritePixels(px)
	// the figure, clipped to the inner circle
	fig := ebiten.NewImage(size, size)
	cv := newCanvas(fig)
	cv.translate(r, S*1.02)
	k := S / 190
	cv.scale(k, k)
	bust(cv, o)
	mask := ebiten.NewImage(size, size)
	mpx := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			a := uint8(255 * clamp(r-7-math.Hypot(float64(x)+.5-r, float64(y)+.5-r)+.5, 0, 1))
			i := (y*size + x) * 4
			mpx[i], mpx[i+1], mpx[i+2], mpx[i+3] = a, a, a, a
		}
	}
	mask.WritePixels(mpx)
	fig.DrawImage(mask, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	out.DrawImage(fig, nil)
	// gilt rings
	ring := func(rad, w float64, col color.RGBA) {
		p := &vector.Path{}
		p.Arc(float32(r), float32(r), float32(rad), 0, 2*math.Pi, vector.Clockwise)
		p.Close()
		stroke(out, p, w, col)
	}
	ring(r-S*0.03, S*0.05, pGold)
	ring(r-S*0.06, 1.5, hex("#7a5a26"))
	fig.Deallocate()
	mask.Deallocate()
	return out
}

/* ---------------- random guests ---------------- */

// Face is all a seat needs to share over the network: the rest is derived from it.
type Face struct {
	Female bool   `json:"f"`
	Seed   uint64 `json:"s"`
}

func darken(c color.RGBA, k float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * k), uint8(float64(c.G) * k), uint8(float64(c.B) * k), 255}
}

func choose[T any](r *rand.Rand, a ...T) T { return a[r.IntN(len(a))] }

var skins = []string{"#f6dbc4", "#f2d4b6", "#f1d0b5", "#eec3a2", "#e3b48e", "#c98f66", "#a8704a", "#7d5236"}

// look builds a guest's appearance from their face seed. Ladies and gentlemen draw from
// different wardrobes, so a face always matches its name.
func (f Face) look() Look {
	r := rand.New(rand.NewPCG(f.Seed, 0x01d3a1d))
	l := Look{Skin: hex(choose(r, skins...)), Sleeves: r.IntN(10) < 6, Stern: r.IntN(10) < 4}
	if f.Female {
		l.Lady = true
		grey := r.IntN(10) < 3
		if grey {
			l.Hair = hex(choose(r, "#9a9aa4", "#c9c4bc", "#b0aaa0"))
			l.Brow = darken(l.Hair, .7)
		} else {
			l.Hair = hex(choose(r, "#b5651d", "#5a3a22", "#7a4a2a", "#d8b46a", "#2a1e18", "#a8461e", "#8a5a30"))
		}
		l.HairStyle = choose(r, "bun", "updo", "ringlets")
		l.Coat = hex(choose(r, "#4b2a5c", "#2f7a7a", "#7a2a3a", "#3a4f7a", "#5a6a3a", "#8a4a6a", "#3a2a3e", "#6b3a28", "#2a5a5a"))
		l.Dark = darken(l.Coat, .62)
		l.Accent = hex(choose(r, "#d9b44a", "#e8b4b8", "#b7a3c9", "#e9e2cf", "#d9b44a"))
		if grey && r.IntN(2) == 0 || r.IntN(5) == 0 {
			l.Collar = "shawl"
			l.Shawl = hex(choose(r, "#8a7a9a", "#7a8a9a", "#9a7a6a", "#6a8a7a", "#a08a7a"))
		} else {
			l.Collar = "lace"
			l.Pearls = r.IntN(10) < 4
		}
		switch n := r.IntN(100); {
		case grey && n < 40:
			l.Hat = "lacecap"
		case n < 35:
			l.Hat = "feather"
			l.HatColor = darken(hex(choose(r, "#4b2a5c", "#2e1838", "#3a4f7a", "#7a2a3a", "#2a2a2a")), .8)
		case n < 60:
			l.Hat = "bonnet"
			l.HatColor = hex(choose(r, "#f1e4c4", "#e9d9b8", "#f4dfe0", "#e2e8d8"))
		case n < 78:
			l.Hat = "pillbox"
		}
		l.Spectacles = r.IntN(100) < 14
		return l
	}
	white := r.IntN(10) < 3
	if white {
		l.Hair = hex(choose(r, "#e8e4dc", "#c9c4bc"))
		l.MustacheColor = hex("#f2eee6")
		l.Brow = hex("#d8d2c6")
	} else {
		l.Hair = hex(choose(r, "#5a3a22", "#2a1e18", "#7a4a2a", "#b5651d", "#c8a060", "#9a9aa4", "#4a3020"))
	}
	l.HairStyle = "short"
	l.Collar = choose(r, "cravat", "cravat", "buttons", "buttons", "military")
	l.Accent = hex(choose(r, "#d9b44a", "#c9a24c", "#c0c4cc"))
	if l.Collar == "military" {
		l.Coat = hex(choose(r, "#a3242b", "#273a5c", "#2f5a3c", "#a3242b"))
		if r.IntN(10) < 6 {
			l.Hat = "military"
		}
	} else {
		l.Coat = hex(choose(r, "#273a5c", "#3b2a1e", "#2f4a3a", "#4a2a2a", "#2a2a3a", "#5a4a3a", "#6b3a28", "#3a3a4a"))
		if r.IntN(10) < 4 {
			l.Hat = "top"
			l.Accent2 = darken(hex(choose(r, "#6b1218", "#273a5c", "#2f5a3c", "#4a3a2a")), .9)
		}
	}
	l.Dark = darken(l.Coat, .62)
	l.Mustache = r.IntN(100) < 55
	l.Muttonchops = r.IntN(100) < 30
	l.Beard = !l.Muttonchops && r.IntN(100) < 25
	switch n := r.IntN(100); {
	case n < 15:
		l.Monocle = true
	case n < 33:
		l.Spectacles = true
	}
	return l
}

// manner is the kind of guest a face suggests; it picks the title and the turns of phrase.
func (f Face) manner() string {
	l := f.look()
	if f.Female {
		switch {
		case l.Hat == "lacecap" || l.Collar == "shawl":
			return "matron"
		case l.Hat == "feather" || l.Pearls:
			return "grand"
		}
		return "young"
	}
	switch {
	case l.Collar == "military":
		return "officer"
	case l.Spectacles:
		return "scholar"
	case l.Hat == "top" || l.Monocle:
		return "gent"
	}
	return "mister"
}

func (f Face) name() string {
	r := rand.New(rand.NewPCG(f.Seed, 0x9a3e5))
	switch f.manner() {
	case "matron":
		return "Mrs. " + choose(r, "Hargreaves", "Thistlewood", "Pembroke", "Bellamy", "Quimby", "Featherstone", "Ashcombe", "Wimbleton")
	case "grand":
		return "Lady " + choose(r, "Agatha", "Winifred", "Beatrice", "Cordelia", "Henrietta", "Philippa", "Eugenia", "Araminta")
	case "young":
		return "Miss " + choose(r, "Pettigrew", "Fairfax", "Lovelace", "Primrose", "Daintree", "Honeywell", "Larkin", "Merriwether")
	case "officer":
		return choose(r, "Colonel", "Major", "Captain") + " " + choose(r, "Bramble", "Ashby", "Hartley", "Fenwick", "Blackwood", "Pemberton", "Thorne", "Harcourt")
	case "scholar":
		return choose(r, "Professor", "Dr.") + " " + choose(r, "Quill", "Hobbs", "Penrose", "Larkspur", "Abernathy", "Grimsby", "Whitlow")
	case "gent":
		if r.IntN(2) == 0 {
			return "Sir " + choose(r, "Reginald", "Percival", "Augustus", "Barnaby", "Montague", "Rupert")
		}
		return "Lord " + choose(r, "Wexley", "Pomfrey", "Ashworth", "Danvers", "Fotheringay")
	}
	return "Mr. " + choose(r, "Whitcombe", "Pickering", "Tumnell", "Fogg", "Bellweather", "Crumb", "Dobbins", "Puddifoot")
}

var manners = map[string]*Person{
	"grand": people["agatha"], "officer": people["colonel"], "young": people["pettigrew"],
	"matron": {pair: []string{"Lovely, dear.", "There we are.", "Just so.", "Oh, that's nice."},
		draw: []string{"Now, which shall it be…", "Let me see, dear…", "Eeny, meeny…"},
		out:  []string{"I'll put the kettle on.", "Time for tea, I think."},
		maid: []string{"Oh, my stars!", "Well, I'll be!"},
		took: []string{"Ta, dear.", "Oh, how kind."}},
	"scholar": {pair: []string{"Elementary!", "As I theorised.", "Fascinating.", "Precisely."},
		draw: []string{"Let us observe…", "Probability suggests… this one.", "Hmm, a hypothesis…"},
		out:  []string{"Q.E.D.!", "My work here is done."},
		maid: []string{"Most irregular!", "Statistically improbable!"},
		took: []string{"Thank you kindly.", "Hmm, curious."}},
	"gent": {pair: []string{"Jolly good.", "Rather!", "Well played, what?", "Splendid, splendid."},
		draw: []string{"Allow me…", "Now then…", "If I may…"},
		out:  []string{"I bid you good evening!", "That concludes my business."},
		maid: []string{"Good heavens!", "Dash it all!"},
		took: []string{"Obliged, I'm sure.", "Most kind."}},
	"mister": {pair: []string{"Ah, a pair!", "Good-oh!", "There we go!", "Lovely jubbly."},
		draw: []string{"Hmm, hmm…", "Righto, this one.", "Come on, lucky…"},
		out:  []string{"That's me done!", "Off I pop!"},
		maid: []string{"Oh, crumbs!", "Blast!"},
		took: []string{"Cheers!", "Ta very much."}},
}

// randomGuests picks n guests with different names (a mix of ladies and gentlemen).
func randomGuests(n int) []Face {
	var out []Face
	seen := map[string]bool{}
	for len(out) < n {
		f := Face{Female: rand.IntN(2) == 0, Seed: rand.Uint64()}
		nm := f.name()
		if seen[nm] {
			continue
		}
		seen[nm] = true
		out = append(out, f)
	}
	return out
}

var cameoCache = map[Face]*ebiten.Image{}

func cameoFor(f Face) *ebiten.Image {
	if im, ok := cameoCache[f]; ok {
		return im
	}
	if len(cameoCache) > 40 {
		for k, im := range cameoCache {
			im.Deallocate()
			delete(cameoCache, k)
		}
	}
	im := renderCameo(f.look(), 304)
	cameoCache[f] = im
	return im
}
