// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// The top bar, the parchment panels (title card, rules, credits & changelog, results) and input.
package main

import (
	"image"
	"image/color"
	"math"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type hit struct {
	id         string
	x, y, w, h float64
	clip       *image.Rectangle
}

func (h hit) has(mx, my float64) bool {
	if h.clip != nil && !(image.Pt(int(mx), int(my)).In(*h.clip)) {
		return false
	}
	return mx >= h.x && mx < h.x+h.w && my >= h.y && my < h.y+h.h
}

// ui state for one pass over the interface
type uiPass struct {
	g       *Game
	dst     *ebiten.Image // nil while only measuring / hit testing
	measure bool          // true while finding a panel's height (no hits recorded)
	clip    *image.Rectangle
}

func (u *uiPass) addHit(id string, x, y, w, h float64) {
	if u.measure {
		return
	}
	u.g.hits = append(u.g.hits, hit{id, x, y, w, h, u.clip})
}

func (g *Game) hovering(id string) bool {
	for i := len(g.hits) - 1; i >= 0; i-- {
		h := g.hits[i]
		if h.has(g.mx, g.my) {
			return h.id == id
		}
	}
	return false
}

func (g *Game) hitAt(mx, my float64) string {
	for i := len(g.hits) - 1; i >= 0; i-- {
		if g.hits[i].has(mx, my) {
			return g.hits[i].id
		}
	}
	return ""
}

/* ---------------- top bar ---------------- */

func (g *Game) bar(u *uiPass) {
	dst := u.dst
	W := float64(g.W)
	if dst != nil {
		// shadow under the bar, then the bar
		shadowRR(dst, -40, -20, W+80, barH+20, 0, 0, 2, 14, 0, rgba(0, 0, 0, .5))
		fillGrad(dst, rrPath(0, 0, W, barH, 0), hex("#3d2216"), hex("#26130b"), 0, barH)
		vector.DrawFilledRect(dst, 0, barH-2, float32(W), 2, hex("#8a6a2e"), false)
	}
	cy := (barH - 2) / 2
	x := 14.0
	x += txt(dst, "Old Maid", font(fSC, 25), x, cy, 0, cGold2, &txtOpt{tracked: 1}) + 10
	if W > 1100 {
		x += txt(dst, "A Victorian Parlor Game", font(fItalic, 13), x, cy+3, 0, hex("#c9ab78"), nil)
	}
	x += 10 + 8
	button := func(id, label string, off bool) {
		f := font(fSerif, 15)
		w := textW(label, f) + 24 + 2
		h := 31.0
		y := cy - h/2
		u.addHit(id, x, y, w, h)
		if dst != nil {
			hov := g.hovering(id)
			alpha := 1.0
			if off {
				alpha = .45
			}
			fillGrad(dst, rrPath(x, y, w, h, 4), withAlpha(hex("#5a3622"), alpha), withAlpha(hex("#3a2014"), alpha), y, y+h)
			bc := cBrass
			tc := cCream
			if hov {
				bc, tc = cGold2, hex("#ffffff")
			}
			stroke(dst, rrPath(x+.5, y+.5, w-1, h-1, 4), 1, withAlpha(bc, alpha))
			tw := txt(dst, label, f, x+w/2, cy, .5, tc, &txtOpt{alpha: alpha})
			if off {
				vector.StrokeLine(dst, float32(x+w/2-tw/2), float32(cy+1), float32(x+w/2+tw/2), float32(cy+1), 1.2, withAlpha(tc, alpha), true)
			}
		}
		x += w + 8
	}
	if g.mode != modeClient {
		button("new", "New Deal", false)
	}
	switch g.mode {
	case modeClient:
		button("menu", "Leave Table", false)
	case modeHost:
		button("menu", "Close Table", false)
	default:
		button("menu", "Parlor", false)
	}
	button("rules", "Rules", false)
	x += 10 + 8
	button("music", "♫ Music", !g.save.Settings.Music)
	// volume slider
	{
		w, y := 90.0, cy
		u.addHit("vol", x-4, y-12, w+8, 24)
		if dst != nil {
			v := float64(g.save.Settings.Vol) / 100
			vector.DrawFilledRect(dst, float32(x), float32(y-2), float32(w), 4, hex("#1d0e07"), true)
			vector.DrawFilledRect(dst, float32(x), float32(y-2), float32(w*v), 4, cGold, true)
			vector.DrawFilledCircle(dst, float32(x+w*v), float32(y), 7.5, cGold, true)
			vector.StrokeCircle(dst, float32(x+w*v), float32(y), 7.5, 1, hex("#8a6a2e"), true)
		}
		g.volX, g.volW = x, w
		x += w + 8
	}
	button("sfx", "Sounds", !g.save.Settings.Sfx)

	// stats on the right
	right := W - 14
	type st struct{ label, n string }
	stats := []st{{"Played", itoa(g.save.Stats.Played)}, {"Safe", itoa(g.save.Stats.Safe)}, {"Old Maid", itoa(g.save.Stats.Maid)}}
	rx := right
	for i := len(stats) - 1; i >= 0; i-- {
		s := stats[i]
		nw := textW(s.n, font(fSerif, 17))
		lw := textW(s.label+" ", font(fSerif, 14))
		if dst != nil {
			txt(dst, s.n, font(fSerif, 17), rx, cy, 1, cGold2, nil)
			txt(dst, s.label+" ", font(fSerif, 14), rx-nw, cy+1, 1, hex("#c9ab78"), nil)
		}
		rx -= nw + lw + 14
	}
	// now playing, centred in the space between
	if dst != nil && g.save.Settings.Music && g.snd.musicReady() && W > 1100 {
		s := "♫  " + g.nowPlaying
		f := font(fItalic, 13)
		space := rx - x
		if textW(s, f) < space-10 {
			txt(dst, s, f, x+space/2, cy+1, .5, hex("#b8956a"), nil)
		}
	}
}

/* ---------------- panels ---------------- */

func (g *Game) panelFrame(dst *ebiten.Image, x, y, w, h float64) {
	shadowRR(dst, x-10, y-10, w+20, h+20, 18, 0, 30, 80, 0, rgba(0, 0, 0, .7))
	fill(dst, rrPath(x-10, y-10, w+20, h+20, 20), cBrass)
	fill(dst, rrPath(x-8, y-8, w+16, h+16, 18), cCream)
	fill(dst, rrPath(x-2, y-2, w+4, h+4, 12), cBrass)
	fillGrad(dst, rrPath(x, y, w, h, 10), cParch, cParch2, y, y+h)
	stroke(dst, rrPath(x+10.5, y+10.5, w-21, h-21, 6), 1, rgba(138, 106, 46, .6))
}

func ornament(u *uiPass, s string, cx, y float64) float64 {
	txt(u.dst, s, font(fSerif, 20), cx, y+13, .5, cBrass, &txtOpt{tracked: 8})
	return 26
}

func (g *Game) bigButton(u *uiPass, id, label string, cx, y float64, small, ghost bool) (w, h float64) {
	fs, px, py := 24.0, 34.0, 11.0
	if small {
		fs, px, py = 20, 28, 8
	}
	f := font(fSC, fs)
	w, h = textW(label, f)+2*px+2, fs*1.25+2*py+2
	x := cx - w/2
	u.addHit(id, x, y, w, h+4)
	if u.dst == nil {
		return
	}
	dst := u.dst
	hov := g.hovering(id)
	down := hov && g.pressed == id
	dy := 0.0
	if down {
		dy = 2
	}
	if ghost {
		stroke(dst, rrPath(x+.5, y+.5+dy, w-1, h-1, 6), 1, cBrass)
		c := cWine
		if hov {
			c = lerpColor(cWine, hex("#a03040"), .5)
		}
		txt(dst, label, f, cx, y+h/2+dy, .5, c, &txtOpt{tracked: fs * .03})
		return
	}
	shadowRR(dst, x, y+dy, w, h, 6, 0, 8, 18, 0, rgba(0, 0, 0, .3))
	sd := 4.0
	if down {
		sd = 2
	}
	fill(dst, rrPath(x, y+dy+sd, w, h, 6), hex("#3a0c12"))
	t, b := hex("#8a2a36"), hex("#5a1620")
	if hov {
		t, b = lerpColor(t, hex("#ffffff"), .08), lerpColor(b, hex("#ffffff"), .06)
	}
	fillGrad(dst, rrPath(x, y+dy, w, h, 6), t, b, y+dy, y+dy+h)
	stroke(dst, rrPath(x+.5, y+dy+.5, w-1, h-1, 6), 1, hex("#3a0c12"))
	txt(dst, label, f, cx, y+dy+h/2, .5, hex("#fbecc4"), &txtOpt{tracked: fs * .03})
	return
}

// chips draws a segmented choice; returns its width and height.
func (g *Game) chips(u *uiPass, prefix string, labels, values []string, on string, x, y float64) (float64, float64) {
	f := font(fSerif, 16)
	h := 34.0
	total := 0.0
	for _, l := range labels {
		total += textW(l, f) + 28
	}
	total += float64(len(labels) - 1)
	if u.dst != nil {
		fill(u.dst, rrPath(x, y, total+2, h, 6), withAlpha(cParch, 0))
	}
	cx := x + 1
	for i, l := range labels {
		w := textW(l, f) + 28
		id := prefix + values[i]
		u.addHit(id, cx, y, w, h)
		if u.dst != nil {
			if values[i] == on {
				r := 0.0
				p := rrPath(cx, y+1, w, h-2, r)
				if i == 0 {
					p = leftRound(cx, y+1, w, h-2, 5)
				} else if i == len(labels)-1 {
					p = rightRound(cx, y+1, w, h-2, 5)
				}
				fill(u.dst, p, cWine)
				txt(u.dst, l, f, cx+w/2, y+h/2, .5, hex("#fbecc4"), nil)
			} else {
				c := cInk
				if g.hovering(id) {
					c = cWine
				}
				txt(u.dst, l, f, cx+w/2, y+h/2, .5, c, nil)
			}
			if i > 0 {
				vector.DrawFilledRect(u.dst, float32(cx-1), float32(y+1), 1, float32(h-2), rgba(138, 106, 46, .5), false)
			}
		}
		cx += w + 1
	}
	if u.dst != nil {
		stroke(u.dst, rrPath(x+.5, y+.5, total+1, h-1, 6), 1, cBrass)
	}
	return total + 2, h
}

func leftRound(x, y, w, h, r float64) *vector.Path {
	p := &vector.Path{}
	X, Y, W, H, R := float32(x), float32(y), float32(w), float32(h), float32(r)
	p.MoveTo(X+R, Y)
	p.LineTo(X+W, Y)
	p.LineTo(X+W, Y+H)
	p.LineTo(X+R, Y+H)
	p.Arc(X+R, Y+H-R, R, math.Pi/2, math.Pi, vector.Clockwise)
	p.LineTo(X, Y+R)
	p.Arc(X+R, Y+R, R, math.Pi, 3*math.Pi/2, vector.Clockwise)
	p.Close()
	return p
}

func rightRound(x, y, w, h, r float64) *vector.Path {
	p := &vector.Path{}
	X, Y, W, H, R := float32(x), float32(y), float32(w), float32(h), float32(r)
	p.MoveTo(X, Y)
	p.LineTo(X+W-R, Y)
	p.Arc(X+W-R, Y+R, R, -math.Pi/2, 0, vector.Clockwise)
	p.LineTo(X+W, Y+H-R)
	p.Arc(X+W-R, Y+H-R, R, 0, math.Pi/2, vector.Clockwise)
	p.LineTo(X, Y+H)
	p.Close()
	return p
}

func (g *Game) link(u *uiPass, id, label string, x, y float64) float64 {
	f := font(fSerif, 16)
	w := textW(label, f)
	u.addHit(id, x-2, y-11, w+4, 22)
	if u.dst != nil {
		c := cWine
		if g.hovering(id) {
			c = hex("#a03040")
		}
		txt(u.dst, label, f, x, y, 0, c, nil)
		vector.StrokeLine(u.dst, float32(x), float32(y+8), float32(x+w), float32(y+8), 1, c, true)
	}
	return w
}

/* ---------------- title card ---------------- */

func (g *Game) titleCard(u *uiPass, x0, y0 float64) (w, h float64) {
	lede1, lede2 := "Pair your cards, pass them round the parlor,", "and whatever you do… don't be left holding the Old Maid."
	fl := font(fItalic, 18)
	cw := math.Max(math.Max(textW(lede1, fl), textW(lede2, fl)), 560)
	w = cw + 92
	cx := x0 + w/2
	y := y0 + 30
	dst := u.dst
	y += ornament(u, "❧ ❧ ❧", cx, y)
	y += 6
	txt(dst, "Old Maid", font(fSC, 74), cx, y+37, .5, cWine, &txtOpt{shadow: rgba(255, 255, 255, .6), sdy: 2, tracked: 1.5})
	y += 74
	y += 6
	txt(dst, "A Victorian Parlor Game", font(fItalic, 22), cx, y+15, .5, hex("#6a4a2a"), nil)
	y += 30 + 10
	// the fan of cards
	y += 6
	if dst != nil {
		show := []string{"H13", "D12", "S12", "S11", "C1"}
		order := []int{0, 1, 3, 4, 2}
		for _, i := range order {
			im := imgFace[show[i]]
			fw, fh := 92.0, 129.0
			bottom := y + 150
			if i == 2 {
				bottom -= 12
			}
			// rotate around (50%, 120%)
			px, py := fw/2, fh*1.2
			var m ebiten.GeoM
			m.Translate(-px, -py)
			m.Rotate(float64(i-2) * 13 * math.Pi / 180)
			m.Translate(cx, bottom-fh+py)
			sh := soft(fw, fh, 6, 0, 10, 0)
			sm := float64(softPad(0, 10, 0))
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Translate(-fw/2-sm+fw/2, -sm+4)
			op.GeoM.Concat(m)
			tint(&op.ColorScale, rgba(0, 0, 0, .35))
			dst.DrawImage(sh, op)
			op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(fw/artW, fh/artH)
			op.GeoM.Concat(m)
			dst.DrawImage(im, op)
		}
	}
	y += 150 + 2
	y += 22
	txt(dst, lede1, fl, cx, y+13, .5, hex("#4a3424"), nil)
	txt(dst, lede2, fl, cx, y+13+26, .5, hex("#4a3424"), nil)
	y += 52 + 14 + 4
	// options
	{
		f := font(fItalic, 15)
		cnt := map[int]string{1: "1", 2: "2", 3: "3"}[g.save.Settings.Opponents]
		meas := &uiPass{g: g, measure: true}
		w1, _ := g.chips(meas, "opp", []string{"Two", "Three", "Four"}, []string{"1", "2", "3"}, cnt, 0, 0)
		w2, _ := g.chips(meas, "speed", []string{"Leisurely", "Proper", "Brisk"}, []string{"relaxed", "normal", "brisk"}, g.save.Settings.Speed, 0, 0)
		l1, l2 := "Players at the table", "Pace of play"
		g1 := math.Max(w1, textW(l1, f))
		g2 := math.Max(w2, textW(l2, f))
		ox := cx - (g1+26+g2)/2
		txt(dst, l1, f, ox, y+10, 0, hex("#6a4a2a"), nil)
		g.chips(u, "opp", []string{"Two", "Three", "Four"}, []string{"1", "2", "3"}, cnt, ox, y+25)
		ox2 := ox + g1 + 26
		txt(dst, l2, f, ox2, y+10, 0, hex("#6a4a2a"), nil)
		g.chips(u, "speed", []string{"Leisurely", "Proper", "Brisk"}, []string{"relaxed", "normal", "brisk"}, g.save.Settings.Speed, ox2, y+25)
		y += 25 + 34 + 18
	}
	_, bh := g.bigButton(u, "start", "Deal the Cards", cx, y, false, false)
	y += bh + 4 + 14
	// play with friends
	{
		meas := &uiPass{g: g, measure: true}
		b1, sh := g.bigButton(meas, "hotseat", "Pass & Play", 0, 0, true, true)
		b2, _ := g.bigButton(meas, "online", "Play Online", 0, 0, true, true)
		bx := cx - (b1+14+b2)/2
		g.bigButton(u, "hotseat", "Pass & Play", bx+b1/2, y, true, true)
		g.bigButton(u, "online", "Play Online", bx+b1+14+b2/2, y, true, true)
		y += sh + 16
	}
	{
		f := font(fSerif, 16)
		a, mid, b := "How to Play", " · ", "Credits & Changelog"
		tw := textW(a, f) + textW(mid, f) + textW(b, f)
		lx := cx - tw/2
		lx += g.link(u, "how", a, lx, y+11)
		lx += txt(dst, mid, f, lx, y+11, 0, hex("#6a4a2a"), nil)
		g.link(u, "credits", b, lx, y+11)
		y += 22
	}
	y += 10
	txt(dst, g.recordText(), font(fItalic, 15), cx, y+10, .5, hex("#6a4a2a"), nil)
	y += 20 + 14
	{
		f := font(fItalic, 15)
		lw := 22 * float64(imgLogo.Bounds().Dx()) / float64(imgLogo.Bounds().Dy())
		tw := textW("An", f) + 8 + lw + 8 + textW("game", f)
		lx := cx - tw/2
		lx += txt(dst, "An", f, lx, y+11, 0, hex("#6a4a2a"), nil) + 8
		if dst != nil {
			var cm colorm.ColorM
			cm.Scale(0, 0, 0, 1)
			cm.Translate(0, 46.0/255, 33.0/255, 0)
			op := &colorm.DrawImageOptions{Filter: ebiten.FilterLinear}
			s := 22 / float64(imgLogo.Bounds().Dy())
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(lx, y)
			colorm.DrawImage(dst, imgLogo, cm, op)
		}
		lx += lw + 8
		txt(dst, "game", f, lx, y+11, 0, hex("#6a4a2a"), nil)
		y += 22
	}
	y += 26
	return w, y - y0
}

/* ---------------- rich text (rules, credits, changelog) ---------------- */

var boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)

type word struct {
	s    string
	bold bool
}

func splitWords(s string) []word {
	var out []word
	last := 0
	for _, m := range boldRe.FindAllStringSubmatchIndex(s, -1) {
		for _, w := range strings.Fields(s[last:m[0]]) {
			out = append(out, word{w, false})
		}
		for _, w := range strings.Fields(s[m[2]:m[3]]) {
			out = append(out, word{w, true})
		}
		// punctuation glued to the end of a bold run stays on the same word
		last = m[1]
	}
	for _, w := range strings.Fields(s[last:]) {
		out = append(out, word{w, false})
	}
	return out
}

// para draws wrapped text from x to x+w starting at y (top); returns the height used.
func para(u *uiPass, s string, size float64, kind int, clr color.RGBA, x, y, w, lh float64) float64 {
	f := font(kind, size)
	sp := textW(" ", f)
	words := splitWords(s)
	line := []word{}
	lw := 0.0
	yy := y
	flush := func() {
		px := x
		for i, wd := range line {
			c := clr
			if wd.bold {
				c = cInk
			}
			txt(u.dst, wd.s, f, px, yy+lh/2, 0, c, &txtOpt{bold: wd.bold})
			px += textW(wd.s, f)
			if i < len(line)-1 {
				px += sp
			}
		}
		yy += lh
		line, lw = line[:0], 0
	}
	for _, wd := range words {
		ww := textW(wd.s, f)
		// keep punctuation that follows a bold word attached (", " etc.)
		if len(line) > 0 && lw+sp+ww > w {
			flush()
		}
		if len(line) > 0 {
			lw += sp
		}
		line = append(line, wd)
		lw += ww
	}
	if len(line) > 0 {
		flush()
	}
	return yy - y
}

var rulesParas = []string{
	"One queen has been taken out of the deck, so the **Old Maid** card has no partner. Whoever is left holding her when every other card has been paired loses the game.",
}
var rulesSteps = []string{
	"All the cards are dealt out. Everyone lays down any **pairs** in their hand: two cards of the same rank **and the same colour** (two red sevens, two black kings, and so on).",
	"On your turn, **draw one card** from the player on your left. Their cards are face down, so choose wisely. Click a card to take it.",
	"If it makes a pair, the pair is laid down automatically.",
	"Play passes to the left. When a player runs out of cards, they are safe and leave the game.",
	"The last player holding a card is holding **the Old Maid**.",
}

const textPanelW = 620.0

func (g *Game) rulesPanel(u *uiPass, x0, y0 float64) float64 {
	dst := u.dst
	x, w := x0+46, textPanelW-92
	y := y0 + 30
	txt(dst, "How to Play", font(fSC, 32), x+w/2, y+4+20, .5, cWine, nil)
	y += 4 + 40 + 10
	for _, p := range rulesParas {
		y += 17 + para(u, p, 17, fSerif, cInk, x, y, w, 23) + 4
	}
	y += 6
	for i, s := range rulesSteps {
		txt(dst, itoa(i+1)+".", font(fSerif, 17), x+18, y+11.5, 1, cInk, nil)
		y += para(u, s, 17, fSerif, cInk, x+22, y, w-22, 23) + 12
	}
	y += 10
	y += para(u, "Keys: F2 new deal · M music on/off · F11 full screen", 15, fSerif, cMuted, x, y, w, 20) + 17
	return y - y0
}

func (g *Game) creditsPanel(u *uiPass, x0, y0 float64) float64 {
	dst := u.dst
	x, w := x0+46, textPanelW-92
	y := y0 + 30
	txt(dst, "Credits", font(fSC, 32), x+w/2, y+4+20, .5, cWine, nil)
	y += 4 + 40 + 10
	row := func(label, val string) {
		h := 30.0
		txt(dst, label, font(fItalic, 17), x, y+h/2, 0, hex("#6a4a2a"), nil)
		lines := wrapRight(val, font(fSerif, 17), w*0.62)
		for i, l := range lines {
			txt(dst, l, font(fSerif, 17), x+w, y+h/2+float64(i)*23, 1, cInk, nil)
		}
		y += h + float64(len(lines)-1)*23
		if dst != nil {
			for dx := 0.0; dx < w; dx += 4 {
				vector.DrawFilledRect(dst, float32(x+dx), float32(y-1), 1.2, 1.2, rgba(138, 106, 46, .5), true)
			}
		}
	}
	y += 0
	row("Developed & Published by", "Ironite")
	row("Game Design", "Ironite")
	row("Card Art & Illustration", "Ironite")
	row("Music", "Ironite")
	var names []string
	for _, t := range tunes {
		names = append(names, "“"+t.name+"”")
	}
	y += 2
	y += para(u, strings.Join(names, ", "), 15, fItalic, hex("#6a4a2a"), x+18, y, w-18, 21) + 4
	row("Sound Effects", "Ironite")
	row("Typeface", "IM Fell English, by Igino Marini (SIL Open Font License)")
	row("Built with", "Ebitengine (Apache License 2.0)")
	y += 16
	y += para(u, "Copyright © 2026 Ironite. All rights reserved.", 15, fSerif, cMuted, x, y, w, 20) + 16
	txt(dst, "Changelog", font(fSC, 32), x+w/2, y+4+20, .5, cWine, nil)
	y += 4 + 40 + 6
	for _, line := range strings.Split(strings.ReplaceAll(changelogText, "\r", ""), "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			s := strings.NewReplacer("[", "", "]", "").Replace(line[3:])
			y += 12
			txt(dst, s, font(fSC, 21), x, y+13, 0, cWine, nil)
			y += 28
		case strings.HasPrefix(line, "### "):
			y += 6
			txt(dst, line[4:], font(fSC, 15), x, y+10, 0, cBrass, &txtOpt{tracked: 0.9})
			y += 22
		case strings.HasPrefix(line, "- "):
			txt(dst, "•", font(fSerif, 16), x+8, y+11, .5, cInk, nil)
			y += para(u, line[2:], 16, fSerif, cInk, x+20, y, w-20, 22) + 3
		}
	}
	y += 10
	return y - y0
}

func wrapRight(s string, f text.Face, w float64) []string {
	var lines []string
	cur := ""
	for _, wd := range strings.Fields(s) {
		if cur != "" && textW(cur+" "+wd, f) > w {
			lines = append(lines, cur)
			cur = wd
		} else if cur == "" {
			cur = wd
		} else {
			cur += " " + wd
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// textPanel shows a scrolling parchment panel with a button pinned to the bottom.
func (g *Game) textPanel(u *uiPass, which string, body func(*uiPass, float64, float64) float64, btnLabel string) {
	H := float64(g.H)
	meas := &uiPass{g: g, measure: true}
	contentH := body(meas, 0, 0)
	btnArea := 20.0 + 41 + 26
	maxH := H*0.86 - btnArea
	viewH := math.Min(contentH, maxH)
	w, h := textPanelW, viewH+btnArea
	x0, y0 := math.Round((float64(g.W)-w)/2), math.Round((H-h)/2)
	maxScroll := math.Max(0, contentH-viewH)
	g.scrollMax = maxScroll
	g.scroll = clamp(g.scroll, 0, maxScroll)
	if u.dst != nil {
		g.panelFrame(u.dst, x0, y0, w, h)
	}
	clip := image.Rect(int(x0)+12, int(y0)+12, int(x0+w)-12, int(y0+viewH))
	sub := &uiPass{g: g, measure: u.measure, clip: &clip}
	if u.dst != nil {
		sub.dst = u.dst.SubImage(clip).(*ebiten.Image)
	}
	body(sub, x0, y0-g.scroll)
	if u.dst != nil && maxScroll > 0 {
		// fade hints and a slim scroll bar
		if g.scroll > 1 {
			fillGrad(u.dst, rrPath(x0+12, y0+12, w-24, 26, 0), withAlpha(cParch, 1), withAlpha(cParch, 0), y0+12, y0+38)
		}
		if g.scroll < maxScroll-1 {
			fillGrad(u.dst, rrPath(x0+12, y0+viewH-30, w-24, 30, 0), withAlpha(cParch2, 0), withAlpha(lerpColor(cParch, cParch2, viewH/h), 1), y0+viewH-30, y0+viewH)
		}
		trackH := viewH - 40
		th := math.Max(30, trackH*viewH/contentH)
		ty := y0 + 20 + (trackH-th)*g.scroll/maxScroll
		fill(u.dst, rrPath(x0+w-20, ty, 4, th, 2), rgba(138, 106, 46, .45))
	}
	u.addHit("panel", x0, y0, w, h)
	g.bigButton(u, "close", btnLabel, x0+w/2, y0+viewH+20, true, false)
}
