// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// Embedded artwork and fonts, plus the drawing helpers the table and panels share:
// rounded rectangles, gradients, soft shadows and text.
package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/art/*.jpg assets/art/*.png assets/fonts/*.ttf assets/ironite-logo.png CHANGELOG.md
var files embed.FS

const artW, artH = 360, 504 // card sprite size (portrait, 5:7)

var (
	imgBack, imgDoily, imgLogo, imgIcon *ebiten.Image
	imgFace                             = map[string]*ebiten.Image{}
	iconImage                           image.Image
	changelogText                       string
)

func mustRead(name string) []byte {
	b, err := files.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return b
}

func decodeRGBA(name string) *image.RGBA {
	im, _, err := image.Decode(bytes.NewReader(mustRead(name)))
	if err != nil {
		panic(name + ": " + err.Error())
	}
	out := image.NewRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out
}

func loadAssets() {
	mask := decodeRGBA("assets/art/card-mask.png")
	for _, s := range "SHDC" {
		for r := 1; r <= 13; r++ {
			id := string(s) + itoa(r)
			name := "assets/art/card-" + id + ".jpg"
			if _, err := files.Open(name); err != nil {
				continue // the queen of clubs left the deck
			}
			face := decodeRGBA(name)
			for i := 0; i+3 < len(face.Pix); i += 4 { // round the corners (premultiplied)
				a := uint32(mask.Pix[i+3])
				face.Pix[i] = uint8(uint32(face.Pix[i]) * a / 255)
				face.Pix[i+1] = uint8(uint32(face.Pix[i+1]) * a / 255)
				face.Pix[i+2] = uint8(uint32(face.Pix[i+2]) * a / 255)
				face.Pix[i+3] = uint8(a)
			}
			imgFace[id] = ebiten.NewImageFromImage(face)
		}
	}
	imgBack = ebiten.NewImageFromImage(decodeRGBA("assets/art/back.png"))
	imgDoily = ebiten.NewImageFromImage(decodeRGBA("assets/art/doily.png"))
	imgLogo = ebiten.NewImageFromImage(decodeRGBA("assets/ironite-logo.png"))
	ic := decodeRGBA("assets/art/icon.png")
	iconImage = ic
	imgIcon = ebiten.NewImageFromImage(ic)
	changelogText = string(mustRead("CHANGELOG.md"))
	loadFonts()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

/* ---------------- fonts ---------------- */

const (
	fSerif = iota
	fItalic
	fSC
)

var (
	fontSrc   [3]*text.GoTextFaceSource
	ornSrc    *text.GoTextFaceSource
	faceCache = map[[2]int]text.Face{}
)

func loadFonts() {
	for i, n := range []string{"IMFellEnglish-Regular", "IMFellEnglish-Italic", "IMFellEnglishSC-Regular"} {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(mustRead("assets/fonts/" + n + ".ttf")))
		if err != nil {
			panic(err)
		}
		fontSrc[i] = s
	}
	// The IM Fell fonts have no fleuron or music notes; these come from a tiny DejaVu Sans subset.
	s, err := text.NewGoTextFaceSource(bytes.NewReader(mustRead("assets/fonts/Ornaments.ttf")))
	if err != nil {
		panic(err)
	}
	ornSrc = s
}

func font(kind int, size float64) text.Face {
	k := [2]int{kind, int(size * 4)}
	if f, ok := faceCache[k]; ok {
		return f
	}
	mf, err := text.NewMultiFace(&text.GoTextFace{Source: fontSrc[kind], Size: size}, &text.GoTextFace{Source: ornSrc, Size: size * 0.9})
	if err != nil {
		panic(err)
	}
	faceCache[k] = mf
	return mf
}

func textW(s string, f text.Face) float64 { return text.Advance(s, f) }

// txt draws one line of text. ax is the horizontal anchor (0 left, .5 centre, 1 right);
// y is the vertical centre of the line.
type txtOpt struct {
	alpha   float64
	shadow  color.RGBA // drawn first when its alpha is set
	sdx     float64
	sdy     float64
	bold    bool
	tracked float64 // extra letter spacing, in px
}

func txt(dst *ebiten.Image, s string, f text.Face, x, y, ax float64, clr color.RGBA, o *txtOpt) float64 {
	alpha := 1.0
	if o != nil && o.alpha != 0 {
		alpha = o.alpha
	}
	w := textW(s, f)
	if o != nil && o.tracked != 0 {
		w += o.tracked * float64(len([]rune(s))-1)
	}
	if dst == nil {
		return w
	}
	x0 := x - w*ax
	one := func(c color.RGBA, dx, dy, a float64) {
		draw1 := func(str string, px float64) {
			op := &text.DrawOptions{}
			op.GeoM.Translate(px+dx, y+dy)
			op.LayoutOptions.SecondaryAlign = text.AlignCenter
			tint(&op.ColorScale, c)
			op.ColorScale.ScaleAlpha(float32(a))
			text.Draw(dst, str, f, op)
			if o != nil && o.bold {
				op.GeoM.Translate(0.7, 0)
				text.Draw(dst, str, f, op)
			}
		}
		if o != nil && o.tracked != 0 {
			px := x0
			for _, r := range s {
				draw1(string(r), px)
				px += textW(string(r), f) + o.tracked
			}
		} else {
			draw1(s, x0)
		}
	}
	if o != nil && o.shadow.A != 0 {
		one(o.shadow, o.sdx, o.sdy, alpha)
	}
	one(clr, 0, 0, alpha)
	return w
}

/* ---------------- colours ---------------- */

func hex(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	v := func(i int) uint8 {
		var n uint8
		for _, c := range s[i : i+2] {
			n <<= 4
			switch {
			case c >= '0' && c <= '9':
				n |= uint8(c - '0')
			case c >= 'a' && c <= 'f':
				n |= uint8(c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				n |= uint8(c - 'A' + 10)
			}
		}
		return n
	}
	return color.RGBA{v(0), v(2), v(4), 255}
}

// rgba makes a straight-alpha colour description; used where a colour is passed to ColorScale.
func rgba(r, g, b uint8, a float64) color.RGBA {
	return color.RGBA{r, g, b, uint8(math.Round(a * 255))}
}

// tint scales a white (or any) image to a straight-alpha colour.
func tint(cs *ebiten.ColorScale, c color.RGBA) {
	a := float32(c.A) / 255
	cs.Scale(float32(c.R)/255*a, float32(c.G)/255*a, float32(c.B)/255*a, a)
}

func withAlpha(c color.RGBA, a float64) color.RGBA { c.A = uint8(math.Round(a * 255)); return c }

var (
	cInk    = hex("#2b1d14")
	cCream  = hex("#f6ecd2")
	cGold   = hex("#d9b45a")
	cGold2  = hex("#f0d68e")
	cWine   = hex("#6e1f2a")
	cBrass  = hex("#8a6a2e")
	cBrown  = hex("#6a4a2a")
	cMuted  = hex("#7a6048")
	cParch  = hex("#fbf3dc")
	cParch2 = hex("#efdfb8")
)

/* ---------------- shapes ---------------- */

var whitePxImg *ebiten.Image

// whitePx is a one-pixel white source for shapes (made on first use, once the game runs).
func whitePx() *ebiten.Image {
	if whitePxImg == nil {
		im := ebiten.NewImage(3, 3)
		im.Fill(color.White)
		whitePxImg = im.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	}
	return whitePxImg
}

func rrPath(x, y, w, h, r float64) *vector.Path {
	r = math.Min(r, math.Min(w, h)/2)
	p := &vector.Path{}
	X, Y, W, H, R := float32(x), float32(y), float32(w), float32(h), float32(r)
	p.MoveTo(X+R, Y)
	p.LineTo(X+W-R, Y)
	p.Arc(X+W-R, Y+R, R, -math.Pi/2, 0, vector.Clockwise)
	p.LineTo(X+W, Y+H-R)
	p.Arc(X+W-R, Y+H-R, R, 0, math.Pi/2, vector.Clockwise)
	p.LineTo(X+R, Y+H)
	p.Arc(X+R, Y+H-R, R, math.Pi/2, math.Pi, vector.Clockwise)
	p.LineTo(X, Y+R)
	p.Arc(X+R, Y+R, R, math.Pi, 3*math.Pi/2, vector.Clockwise)
	p.Close()
	return p
}

// fillGrad fills a path with a vertical gradient from top (at y0) to bot (at y1).
func fillGrad(dst *ebiten.Image, p *vector.Path, top, bot color.RGBA, y0, y1 float64) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		t := 0.0
		if y1 > y0 {
			t = clamp((float64(vs[i].DstY)-y0)/(y1-y0), 0, 1)
		}
		c := lerpColor(top, bot, t)
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	op := &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero}
	dst.DrawTriangles(vs, is, whitePx(), op)
}

func fill(dst *ebiten.Image, p *vector.Path, c color.RGBA) { fillGrad(dst, p, c, c, 0, 0) }

func stroke(dst *ebiten.Image, p *vector.Path, width float64, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: float32(width), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound})
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, whitePx(), &ebiten.DrawTrianglesOptions{AntiAlias: true, FillRule: ebiten.FillRuleNonZero})
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

/* ---------------- soft shadows and glows ---------------- */

// sdRR is the signed distance from (px,py) to a rounded rectangle centred on the origin.
func sdRR(px, py, hw, hh, r float64) float64 {
	qx, qy := math.Abs(px)-hw+r, math.Abs(py)-hh+r
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

type softKey struct{ w, h, r, spread, blur, ring int }

var softCache = map[softKey]*ebiten.Image{}

// soft renders a white mask: a CSS-style box-shadow (spread + gaussian blur) of a w x h rounded
// rectangle, or, when ring > 0, a hard ring of that width around it. The image has a margin of
// pad(…) pixels on each side so it is drawn at (x - pad, y - pad).
func soft(w, h, r, spread, blur, ring float64) *ebiten.Image {
	k := softKey{int(w), int(h), int(r), int(spread), int(blur), int(ring)}
	if im, ok := softCache[k]; ok {
		return im
	}
	m := softPad(spread, blur, ring)
	W, H := int(w)+2*m, int(h)+2*m
	px := make([]byte, W*H*4)
	hw, hh := float64(int(w))/2, float64(int(h))/2
	sigma := blur / 2
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			d := sdRR(float64(x)+0.5-float64(W)/2, float64(y)+0.5-float64(H)/2, hw, hh, r)
			var a float64
			if ring > 0 {
				a = clamp(ring-d+0.5, 0, 1)
			} else if sigma < 0.5 {
				a = clamp(spread-d+0.5, 0, 1)
			} else {
				a = 0.5 * math.Erfc((d-spread)/(sigma*math.Sqrt2))
			}
			v := uint8(math.Round(a * 255))
			i := (y*W + x) * 4
			px[i], px[i+1], px[i+2], px[i+3] = v, v, v, v
		}
	}
	im := ebiten.NewImage(W, H)
	im.WritePixels(px)
	softCache[k] = im
	if len(softCache) > 400 { // window resizes make new sizes; keep the cache from growing forever
		for kk := range softCache {
			if kk != k {
				delete(softCache, kk)
			}
		}
	}
	return im
}

func softPad(spread, blur, ring float64) int { return int(math.Ceil(spread+blur*1.6+ring)) + 2 }

// shadowRR draws a soft shadow / glow for the rounded rectangle at (x, y, w, h).
func shadowRR(dst *ebiten.Image, x, y, w, h, r, dx, dy, blur, spread float64, c color.RGBA) {
	im := soft(math.Round(w), math.Round(h), math.Round(r), math.Round(spread), math.Round(blur), 0)
	m := float64(softPad(math.Round(spread), math.Round(blur), 0))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-m+dx, y-m+dy)
	tint(&op.ColorScale, c)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(im, op)
}

// drawFrom draws an image of size (iw, ih) centred at (cx, cy), scaled to width w.
func drawCentered(dst, im *ebiten.Image, cx, cy, w float64, alpha float64) {
	b := im.Bounds()
	s := w / float64(b.Dx())
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx, cy)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(im, op)
}

/* ---------------- easing ---------------- */

func bezier(x1, y1, x2, y2 float64) func(float64) float64 {
	bx := func(t float64) float64 { u := 1 - t; return 3*x1*t*u*u + 3*x2*t*t*u + t*t*t }
	by := func(t float64) float64 { u := 1 - t; return 3*y1*t*u*u + 3*y2*t*t*u + t*t*t }
	return func(x float64) float64 {
		if x <= 0 {
			return 0
		}
		if x >= 1 {
			return 1
		}
		lo, hi := 0.0, 1.0
		for i := 0; i < 22; i++ {
			mid := (lo + hi) / 2
			if bx(mid) < x {
				lo = mid
			} else {
				hi = mid
			}
		}
		return by((lo + hi) / 2)
	}
}

var (
	easeCard   = bezier(.3, .7, .2, 1)  // card moves
	easeInOut  = bezier(.42, 0, .58, 1) // flips
	easeOut    = bezier(0, 0, .58, 1)   // lifts
	easeCSS    = bezier(.25, .1, .25, 1)
	easeBanner = bezier(.2, 1.4, .4, 1)
	easeLinear = func(t float64) float64 { return clamp(t, 0, 1) }
)
