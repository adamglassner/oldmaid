// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// The results card, the multiplayer screens (Pass & Play, Play Online, the lobby and the
// "Pass to …" screen), text boxes, and the input that drives them all.
package main

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

/* ---------------- text boxes ---------------- */

// field draws a one-line text box bound to *val; returns its height.
func (g *Game) field(u *uiPass, id string, val *string, placeholder string, x, y, w float64) float64 {
	h := 36.0
	u.addHit("field:"+id, x, y, w, h)
	if u.dst == nil {
		return h
	}
	dst := u.dst
	focused := g.focus == id
	fill(dst, rrPath(x, y, w, h, 5), rgba(255, 252, 242, 1))
	bc := rgba(138, 106, 46, .7)
	if focused {
		bc = cWine
	}
	stroke(dst, rrPath(x+.5, y+.5, w-1, h-1, 5), 1.2, bc)
	f := font(fSerif, 18)
	s := *val
	if s == "" && !focused {
		txt(dst, placeholder, font(fItalic, 17), x+12, y+h/2, 0, rgba(122, 96, 72, .7), nil)
	} else {
		tw := txt(dst, s, f, x+12, y+h/2, 0, cInk, nil)
		if focused && (time.Now().UnixMilli()/530)%2 == 0 {
			vector.DrawFilledRect(dst, float32(x+13+tw), float32(y+8), 1.6, float32(h-16), cInk, true)
		}
	}
	return h
}

func (g *Game) fieldPtr(id string) (*string, int) {
	s := &g.save.Settings
	switch {
	case id == "name":
		return &s.Name, 16
	case id == "addr":
		return &s.Host, 60
	case strings.HasPrefix(id, "seat"):
		i := int(id[4] - '0')
		for len(s.Seats) <= i {
			s.Seats = append(s.Seats, SeatSetting{})
		}
		return &s.Seats[i].Name, 16
	}
	return nil, 0
}

func (g *Game) typeInto() {
	if g.focus == "" {
		return
	}
	p, limit := g.fieldPtr(g.focus)
	if p == nil {
		return
	}
	changed := false
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= 32 && utf8.RuneCountInString(*p) < limit {
			*p += string(r)
			changed = true
		}
	}
	if repeatKey(ebiten.KeyBackspace) && *p != "" {
		_, n := utf8.DecodeLastRuneInString(*p)
		*p = (*p)[:len(*p)-n]
		changed = true
	}
	if changed {
		g.persist(false)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		switch g.focus {
		case "addr":
			g.action("join")
		case "name":
			g.focus = ""
		default:
			g.focus = ""
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		order := []string{"name", "addr"}
		if g.showHotseat {
			order = []string{"seat0", "seat1", "seat2", "seat3"}[:g.save.Settings.Opponents+1]
		}
		for i, id := range order {
			if id == g.focus {
				g.focus = order[(i+1)%len(order)]
				break
			}
		}
	}
}

func repeatKey(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d > 30 && d%3 == 0)
}

/* ---------------- shared bits ---------------- */

func heading(u *uiPass, s string, cx, y float64) float64 {
	txt(u.dst, s, font(fSC, 32), cx, y+24, .5, cWine, nil)
	return 54
}

func (g *Game) buttonRow(u *uiPass, cx, y float64, ids, labels []string, ghost []bool) float64 {
	meas := &uiPass{g: g, measure: true}
	var ws []float64
	total, h := 0.0, 0.0
	for i := range ids {
		w, hh := g.bigButton(meas, ids[i], labels[i], 0, 0, true, ghost[i])
		ws = append(ws, w)
		total += w
		h = hh
	}
	total += 14 * float64(len(ids)-1)
	x := cx - total/2
	for i := range ids {
		g.bigButton(u, ids[i], labels[i], x+ws[i]/2, y, true, ghost[i])
		x += ws[i] + 14
	}
	return h + 4
}

const formW = 560.0

/* ---------------- Pass & Play ---------------- */

func (g *Game) hotseatCard(u *uiPass, x0, y0 float64) (w, h float64) {
	w = formW + 92
	x, cx := x0+46, x0+w/2
	y := y0 + 30
	dst := u.dst
	y += heading(u, "Pass & Play", cx, y)
	y += para(u, "Share this computer with friends. Each hand stays hidden until it's that player's turn, and computer guests fill any empty chairs.", 16, fItalic, hex("#6a4a2a"), x, y, formW, 22) + 14
	s := &g.save.Settings
	n := s.Opponents + 1
	txt(dst, "Players at the table", font(fItalic, 15), x, y+10, 0, hex("#6a4a2a"), nil)
	g.chips(u, "opp", []string{"Two", "Three", "Four"}, []string{"1", "2", "3"}, itoa(s.Opponents), x, y+25)
	y += 25 + 34 + 18
	for len(s.Seats) < 4 {
		s.Seats = append(s.Seats, SeatSetting{Human: len(s.Seats) < 2})
	}
	s.Seats[0].Human = true
	for i := 0; i < n; i++ {
		txt(dst, "Seat "+itoa(i+1), font(fSC, 18), x, y+18, 0, cInk, nil)
		fx := x + 80
		cw, _ := g.chips(&uiPass{g: g, measure: true}, "kind"+itoa(i), []string{"Person", "Computer"}, []string{"p", "c"}, "p", 0, 0)
		if s.Seats[i].Human {
			g.field(u, "seat"+itoa(i), &s.Seats[i].Name, "Player "+itoa(i+1), fx, y, formW-80-cw-14)
		} else {
			txt(dst, "A computer guest", font(fItalic, 17), fx+12, y+18, 0, cMuted, nil)
		}
		if i > 0 {
			on := "c"
			if s.Seats[i].Human {
				on = "p"
			}
			g.chips(u, "kind"+itoa(i)+":", []string{"Person", "Computer"}, []string{"p", "c"}, on, x+formW-cw, y+1)
		} else {
			txt(dst, "always a person", font(fItalic, 15), x+formW, y+18, 1, cMuted, nil)
		}
		y += 36 + 12
	}
	y += 8
	y += g.buttonRow(u, cx, y, []string{"hsdeal", "back"}, []string{"Deal the Cards", "Back"}, []bool{false, true})
	y += 26
	return w, y - y0
}

/* ---------------- Play Online ---------------- */

func (g *Game) onlineCard(u *uiPass, x0, y0 float64) (w, h float64) {
	w = formW + 92
	x, cx := x0+46, x0+w/2
	y := y0 + 30
	dst := u.dst
	y += heading(u, "Play Online", cx, y)
	y += para(u, "Play with friends on other computers. One of you hosts the table; everyone else joins with the host's address.", 16, fItalic, hex("#6a4a2a"), x, y, formW, 22) + 14
	txt(dst, "Your name", font(fItalic, 15), x, y+10, 0, hex("#6a4a2a"), nil)
	y += 24
	y += g.field(u, "name", &g.save.Settings.Name, "Your name", x, y, formW) + 22

	// host
	txt(dst, "Host a table", font(fSC, 21), x, y+13, 0, cWine, nil)
	y += 30
	y += para(u, "Open a table on this computer, then tell your friends its address.", 16, fSerif, cInk, x, y, formW, 22) + 8
	_, bh := g.bigButton(u, "host", "Host a Table", cx, y, true, false)
	y += bh + 4 + 20
	if dst != nil {
		for dx := 0.0; dx < formW; dx += 4 {
			vector.DrawFilledRect(dst, float32(x+dx), float32(y), 1.2, 1.2, rgba(138, 106, 46, .5), true)
		}
	}
	y += 16
	// join
	txt(dst, "Join a table", font(fSC, 21), x, y+13, 0, cWine, nil)
	y += 30
	txt(dst, "Host's address", font(fItalic, 15), x, y+10, 0, hex("#6a4a2a"), nil)
	y += 24
	meas := &uiPass{g: g, measure: true}
	jw, _ := g.bigButton(meas, "join", "Join", 0, 0, true, false)
	g.field(u, "addr", &g.save.Settings.Host, "For example 192.168.1.23", x, y, formW-jw-14)
	label := "Join"
	if g.joining {
		label = "…"
	}
	g.bigButton(u, "join", label, x+formW-jw/2, y-2, true, false)
	y += 36 + 14
	if g.netStatus != "" {
		y += para(u, g.netStatus, 16, fItalic, cWine, x, y, formW, 22) + 6
	}
	y += 10
	y += g.buttonRow(u, cx, y, []string{"back"}, []string{"Back"}, []bool{true})
	y += 26
	return w, y - y0
}

/* ---------------- the lobby ---------------- */

func (g *Game) lobbyCard(u *uiPass, x0, y0 float64) (w, h float64) {
	w = formW + 92
	x, cx := x0+46, x0+w/2
	y := y0 + 30
	dst := u.dst
	var l *Lobby
	if g.mode == modeHost {
		l = g.lobbyInfo()
		y += heading(u, "Your Table Is Open", cx, y)
		ips := localIPs()
		addr := "this computer's network address"
		if len(ips) > 0 {
			addr = strings.Join(ips, "  or  ")
		}
		y += para(u, "Friends join from Play Online with this address:", 16, fItalic, hex("#6a4a2a"), x, y, formW, 22) + 6
		txt(dst, addr, font(fSC, 30), cx, y+18, .5, cInk, &txtOpt{tracked: 1})
		y += 44
		y += para(u, "On the same Wi-Fi or network, that's all they need. To play over the internet, forward port "+itoa(netPort)+" on your router to this computer. If Windows asks, allow Old Maid on private networks.", 15, fSerif, cMuted, x, y, formW, 20) + 16
		txt(dst, "Players at the table", font(fItalic, 15), x, y+10, 0, hex("#6a4a2a"), nil)
		g.chips(u, "opp", []string{"Two", "Three", "Four"}, []string{"1", "2", "3"}, itoa(g.save.Settings.Opponents), x, y+25)
		txt(dst, "Pace of play", font(fItalic, 15), x+300, y+10, 0, hex("#6a4a2a"), nil)
		g.chips(u, "speed", []string{"Leisurely", "Proper", "Brisk"}, []string{"relaxed", "normal", "brisk"}, g.save.Settings.Speed, x+300, y+25)
		y += 25 + 34 + 18
	} else {
		if g.client != nil {
			l = g.client.lobby
		}
		if l == nil {
			l = &Lobby{Host: "the host", Seats: 2}
		}
		y += heading(u, "At "+l.Host+"'s Table", cx, y)
	}
	seats := max(l.Seats, len(l.Names))
	for i := 0; i < seats; i++ {
		s := "A computer guest (until a friend joins)"
		c := cMuted
		if g.mode == modeClient {
			s = "A computer guest"
		}
		if i < len(l.Names) {
			s, c = l.Names[i], cInk
			if i == 0 {
				s += " — host"
			}
			if g.mode == modeClient && g.client != nil && i == g.client.seat {
				s += " (you)"
			}
			if g.mode == modeHost && i == 0 {
				s += " (you)"
			}
		}
		txt(dst, itoa(i+1)+".", font(fSerif, 18), x+18, y+13, 1, c, nil)
		txt(dst, s, font(fSerif, 18), x+26, y+13, 0, c, nil)
		y += 28
	}
	y += 14
	if g.mode == modeHost {
		y += g.buttonRow(u, cx, y, []string{"hostdeal", "closetable"}, []string{"Deal the Cards", "Close Table"}, []bool{false, true})
	} else {
		y += para(u, "Waiting for "+l.Host+" to deal…", 17, fItalic, hex("#6a4a2a"), x, y, formW, 24) + 10
		y += g.buttonRow(u, cx, y, []string{"leave"}, []string{"Leave Table"}, []bool{true})
	}
	y += 26
	return w, y - y0
}

/* ---------------- pass to the next player ---------------- */

func (g *Game) passCard(u *uiPass, x0, y0 float64) (w, h float64) {
	name := "the next player"
	if g.passTo >= 0 && g.passTo < len(g.players) {
		name = g.players[g.passTo].name
	}
	ft := font(fSC, 54)
	w = math.Max(textW("Pass to "+name, ft), 440) + 92
	cx := x0 + w/2
	y := y0 + 30
	y += ornament(u, "❧", cx, y)
	txt(u.dst, "Pass to "+name, ft, cx, y+30, .5, cWine, &txtOpt{shadow: rgba(255, 255, 255, .6), sdy: 2})
	y += 66
	txt(u.dst, "Everyone else, no peeking!", font(fItalic, 20), cx, y+14, .5, hex("#6a4a2a"), nil)
	y += 44
	_, bh := g.bigButton(u, "ready", "I'm "+name+" — Show My Cards", cx, y, false, false)
	y += bh + 4 + 26
	return w, y - y0
}

/* ---------------- notices ---------------- */

func (g *Game) noticeCard(u *uiPass, x0, y0 float64) (w, h float64) {
	w = 480 + 92
	x, cx := x0+46, x0+w/2
	y := y0 + 30
	y += ornament(u, "❧", cx, y)
	y += para(u, g.notice, 19, fSerif, cInk, x, y, 480, 26) + 18
	y += g.buttonRow(u, cx, y, []string{"noticeok"}, []string{"Very Good"}, []bool{false})
	y += 26
	return w, y - y0
}

/* ---------------- results ---------------- */

func (g *Game) curResult() *Result {
	if g.mode == modeClient {
		return g.clientRes
	}
	return g.results[g.viewer]
}

func (g *Game) resultCard(u *uiPass, x0, y0 float64) (w, h float64) {
	r := g.curResult()
	if r == nil {
		r = &Result{Title: "…"}
	}
	dst := u.dst
	ft, fs, fl := font(fSC, 60), font(fItalic, 22), font(fSerif, 19)
	cw := math.Max(textW(r.Title, ft), textW(r.Sub, fs))
	listW := 0.0
	for i, l := range r.Places {
		listW = math.Max(listW, textW(itoa(i+1)+".  "+l.Text, fl))
	}
	rec := g.recordText()
	showRec := g.mode != modeHotseat
	if showRec {
		cw = math.Max(cw, textW(rec, font(fItalic, 15)))
	}
	cw = math.Max(math.Max(cw, listW), 420)
	w = cw + 92
	cx := x0 + w/2
	y := y0 + 30
	y += ornament(u, "❧", cx, y)
	y += 6
	txt(dst, r.Title, ft, cx, y+30, .5, cWine, &txtOpt{shadow: rgba(255, 255, 255, .6), sdy: 2})
	y += 60 + 6
	txt(dst, r.Sub, fs, cx, y+15, .5, hex("#6a4a2a"), nil)
	y += 30 + 10
	lx := cx - listW/2
	for i, l := range r.Places {
		c, f := cInk, fl
		if l.Maid {
			c, f = cWine, font(fItalic, 19)
		}
		num := itoa(i+1) + ".  "
		txt(dst, num, fl, lx, y+13, 0, c, nil)
		txt(dst, l.Text, f, lx+textW(num, fl), y+13, 0, c, &txtOpt{bold: l.You})
		y += 27
	}
	y += 10
	if showRec {
		txt(dst, rec, font(fItalic, 15), cx, y+10, .5, hex("#6a4a2a"), nil)
		y += 20 + 16
	}
	switch g.mode {
	case modeClient:
		host := "the host"
		if g.client != nil && g.client.lobby != nil {
			host = g.client.lobby.Host
		}
		txt(dst, "Waiting for "+host+" to deal again…", font(fItalic, 17), cx, y+12, .5, hex("#6a4a2a"), nil)
		y += 34
		y += g.buttonRow(u, cx, y, []string{"leave"}, []string{"Leave Table"}, []bool{true})
	case modeHost:
		y += g.buttonRow(u, cx, y, []string{"again", "closetable"}, []string{"Deal Again", "Close Table"}, []bool{false, true})
	default:
		meas := &uiPass{g: g, measure: true}
		b1, bh := g.bigButton(meas, "again", "Deal Again", 0, 0, false, false)
		b2, _ := g.bigButton(meas, "tomenu", "Parlor", 0, 0, false, true)
		bx := cx - (b1+14+b2)/2
		g.bigButton(u, "again", "Deal Again", bx+b1/2, y, false, false)
		g.bigButton(u, "tomenu", "Parlor", bx+b1+14+b2/2, y, false, true)
		y += bh + 4
	}
	y += 26
	return w, y - y0
}

/* ---------------- the whole interface ---------------- */

func (g *Game) panelFn(name string) func(*uiPass, float64, float64) (float64, float64) {
	return map[string]func(*uiPass, float64, float64) (float64, float64){
		"title": g.titleCard, "result": g.resultCard, "hotseat": g.hotseatCard, "online": g.onlineCard,
		"lobby": g.lobbyCard, "pass": g.passCard, "notice": g.noticeCard,
	}[name]
}

// overlay draws (or, with dst nil, just lays out) everything above the table.
func (g *Game) overlay(dst *ebiten.Image) {
	g.hits = g.hits[:0]
	u := &uiPass{g: g, dst: dst}
	g.bar(u)
	stack := g.panels()
	for i, name := range stack {
		g.hits = g.hits[:0] // each veil blocks what is under it
		if dst != nil {
			dst.DrawImage(g.veil, nil)
		}
		g.hits = append(g.hits, hit{id: "veil", x: 0, y: 0, w: float64(g.W), h: float64(g.H)})
		switch name {
		case "rules":
			g.textPanel(u, "rules", g.rulesPanel, "Very Good")
		case "credits":
			g.textPanel(u, "credits", g.creditsPanel, "Close")
		default:
			f := g.panelFn(name)
			w, h := f(&uiPass{g: g, measure: true}, 0, 0)
			x0, y0 := math.Round((float64(g.W)-w)/2), math.Round((float64(g.H)-h)/2)
			if dst != nil {
				g.panelFrame(dst, x0, y0, w, h)
			}
			u.addHit("panel", x0, y0, w, h)
			f(u, x0, y0)
		}
		_ = i
	}
}

// panels lists the open panels, bottom first (at most two: a screen and a help panel over it).
func (g *Game) panels() []string {
	var base string
	switch {
	case g.notice != "":
		base = "notice"
	case g.passTo >= 0 && g.mode == modeHotseat:
		base = "pass"
	case g.showOnline:
		base = "online"
	case g.showHotseat:
		base = "hotseat"
	case g.showLobby:
		base = "lobby"
	case g.showTitle:
		base = "title"
	case g.showResult && g.curResult() != nil:
		base = "result"
	}
	var out []string
	if base != "" {
		out = append(out, base)
	}
	if g.showRules {
		out = append(out, "rules")
	} else if g.showCredits {
		out = append(out, "credits")
	}
	return out
}

func (g *Game) topPanel() string {
	p := g.panels()
	if len(p) == 0 {
		return ""
	}
	return p[len(p)-1]
}

/* ---------------- input ---------------- */

func (g *Game) handleInput() {
	cx, cy := ebiten.CursorPosition()
	g.mx, g.my = float64(cx), float64(cy)
	g.overlay(nil)

	// hover over the cards you may draw
	ox, oy := g.feltOrigin()
	var hc *Card
	if g.topPanel() == "" && g.hitAt(g.mx, g.my) == "" && len(g.table) > 0 {
		hc = g.cardAt(g.mx-ox, g.my-oy)
	}
	for _, c := range g.table {
		was := c.hover
		c.hover = c == hc
		if c.hover && !was && c.pick {
			g.snd.play("hover")
		}
	}
	now := time.Now()
	for _, c := range g.table {
		v := 0.0
		if (c.pick && c.hover) || c.tempt {
			v = 1
		}
		c.lift.set(v, now, 0.16, easeOut)
	}

	id := g.hitAt(g.mx, g.my)
	clickable := id != "" && id != "panel" && id != "veil"
	if hc != nil && hc.pick {
		clickable = true
	}
	switch {
	case strings.HasPrefix(id, "field:"):
		ebiten.SetCursorShape(ebiten.CursorShapeText)
	case clickable:
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
	default:
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.pressed = id
		if id == "" && hc != nil {
			g.pressedCard = hc
		}
		if strings.HasPrefix(id, "field:") {
			g.focus = id[6:]
		} else if id != "" {
			g.focus = ""
		}
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.pressed == "vol" {
		v := int(math.Round(clamp((g.mx-g.volX)/g.volW, 0, 1) * 100))
		if v != g.save.Settings.Vol {
			g.save.Settings.Vol = v
			g.snd.setVolume(float64(v) / 100)
			g.persist(false)
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if g.pressed != "" && g.pressed == id {
			g.action(id)
		} else if g.pressedCard != nil && g.pressedCard == hc && hc.pick {
			g.pickCard(hc)
		}
		g.pressed, g.pressedCard = "", nil
	}
	if _, wy := ebiten.Wheel(); wy != 0 && (g.showRules || g.showCredits) {
		g.scroll = clamp(g.scroll-wy*48, 0, g.scrollMax)
	}

	if g.focus != "" {
		g.typeInto()
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.focus = ""
		}
		return
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF2):
		if g.mode != modeClient && !g.showLobby && g.notice == "" && !g.showOnline && !g.showHotseat {
			g.showTitle, g.showRules, g.showCredits = false, false, false
			g.action("new")
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyM):
		g.action("music")
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		g.showRules, g.showCredits = false, false
	case inpututil.IsKeyJustPressed(ebiten.KeyF11):
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if g.showRules || g.showCredits {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyDown):
			g.scroll = clamp(g.scroll+48, 0, g.scrollMax)
		case inpututil.IsKeyJustPressed(ebiten.KeyUp):
			g.scroll = clamp(g.scroll-48, 0, g.scrollMax)
		case inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
			g.scroll = clamp(g.scroll+300, 0, g.scrollMax)
		case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
			g.scroll = clamp(g.scroll-300, 0, g.scrollMax)
		}
	}
}

// pickCard draws the card you clicked (from the frame loop, g.mu held).
func (g *Game) pickCard(c *Card) {
	if g.mode == modeClient {
		if g.client != nil {
			g.client.p.send(&Msg{T: "pick", Card: c.id})
			g.clientPick = -1
		}
		return
	}
	g.resolvePick(g.viewer, c)
}

// startSolo, startHotseat: deal a hand on this computer.
func (g *Game) startSolo() {
	g.mode, g.plan = modeSolo, nil
	g.showTitle = false
	g.newGame()
}

func (g *Game) startHotseat() {
	s := &g.save.Settings
	n := s.Opponents + 1
	g.plan = nil
	people := 0
	for i := 0; i < n; i++ {
		st := s.Seats[i]
		if st.Human || i == 0 {
			people++
			g.plan = append(g.plan, seatPlan{kind: pkLocal, name: cleanName(st.Name, "Player "+itoa(i+1))})
		} else {
			g.plan = append(g.plan, seatPlan{kind: pkAI})
		}
	}
	g.mode = modeHotseat
	if people == 1 { // only one person: that's an ordinary game against the computer
		g.mode = modeSolo
		g.plan[0].name = "You"
	}
	g.showHotseat, g.showTitle = false, false
	g.persist(false)
	g.newGame()
}

func (g *Game) action(id string) {
	s := &g.save.Settings
	switch {
	case id == "start":
		g.startSolo()
	case id == "new":
		switch g.mode {
		case modeClient:
		case modeHost:
			if !g.showLobby {
				g.hostDeal()
			}
		case modeHotseat:
			g.showTitle = false
			g.newGame()
		default:
			g.startSolo()
		}
	case id == "again":
		g.snd.play("tick")
		if g.mode == modeHost {
			g.hostDeal()
		} else {
			g.newGame()
		}
	case id == "menu" || id == "tomenu":
		switch g.mode {
		case modeClient:
			g.leaveClient()
		case modeHost:
			g.closeTable()
			g.toTitle()
		default:
			g.toTitle()
		}
	case id == "hotseat":
		g.showHotseat, g.focus = true, ""
		if s.Opponents < 1 {
			s.Opponents = 1
		}
	case id == "online":
		g.showOnline, g.netStatus, g.focus = true, "", ""
		if s.Name == "" {
			g.focus = "name"
		}
	case id == "back":
		g.showHotseat, g.showOnline, g.focus = false, false, ""
	case id == "hsdeal":
		g.startHotseat()
	case id == "host":
		if strings.TrimSpace(s.Name) == "" {
			g.netStatus, g.focus = "Please type your name first, so your friends know whose table it is.", "name"
			return
		}
		if err := g.hostTable(); err != nil {
			g.netStatus = err.Error()
			return
		}
		g.showOnline = false
		g.persist(false)
	case id == "join":
		if g.joining || g.client != nil {
			return
		}
		if strings.TrimSpace(s.Name) == "" {
			g.netStatus, g.focus = "Please type your name first, so the table knows who you are.", "name"
			return
		}
		g.persist(false)
		g.joinTable()
	case id == "hostdeal":
		g.hostDeal()
	case id == "closetable":
		g.closeTable()
		g.toTitle()
	case id == "leave":
		g.leaveClient()
	case id == "ready":
		if g.readyCh != nil {
			select {
			case g.readyCh <- true:
			default:
			}
		}
	case id == "noticeok":
		g.notice = ""
	case id == "rules" || id == "how":
		g.showRules, g.scroll = true, 0
	case id == "credits":
		g.showCredits, g.scroll = true, 0
	case id == "close":
		if g.showRules {
			g.showRules = false
		} else {
			g.showCredits = false
		}
	case id == "music":
		s.Music = !s.Music
		g.snd.setMusic(s.Music)
		g.persist(false)
	case id == "sfx":
		s.Sfx = !s.Sfx
		g.snd.setSfx(s.Sfx)
		g.persist(false)
		g.snd.play("tick")
	case strings.HasPrefix(id, "opp"):
		v := int(id[3] - '0')
		if g.mode == modeHost && g.host != nil && v < len(g.host.lobby) {
			v = len(g.host.lobby) // can't seat fewer than have joined
		}
		s.Opponents = v
		g.persist(false)
		g.snd.play("tick")
		if g.mode == modeHost {
			g.sendLobby()
		}
	case strings.HasPrefix(id, "speed"):
		s.Speed = id[5:]
		g.persist(false)
		g.snd.play("tick")
	case strings.HasPrefix(id, "kind"):
		i := int(id[4] - '0')
		if i > 0 && i < len(s.Seats) {
			s.Seats[i].Human = strings.HasSuffix(id, "p")
			g.persist(false)
			g.snd.play("tick")
		}
	}
}
