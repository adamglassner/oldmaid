// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// Online play. One player hosts a table and runs the game; friends join by the host's
// address. The host sends each guest the table (and what they should see and hear) as it
// changes, and guests send back the card they choose. Messages are lines of JSON over TCP.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

const netPort = 47820

type Msg struct {
	T     string  `json:"t"`
	Name  string  `json:"name,omitempty"`
	Ver   string  `json:"ver,omitempty"`
	Text  string  `json:"text,omitempty"`
	Seat  int     `json:"seat,omitempty"`
	Card  string  `json:"card,omitempty"`
	Snap  *Snap   `json:"snap,omitempty"`
	Ev    []*Ev   `json:"ev,omitempty"`
	Res   *Result `json:"res,omitempty"`
	Lobby *Lobby  `json:"lobby,omitempty"`
}

type Lobby struct {
	Host  string   `json:"host"`
	Names []string `json:"names"` // everyone at the table, host first
	Seats int      `json:"seats"`
}

type SnapP struct {
	Name  string `json:"n"`
	Kind  int    `json:"k"`
	Face  *Face  `json:"f,omitempty"`
	Out   bool   `json:"o,omitempty"`
	Place int    `json:"p,omitempty"`
	Maid  bool   `json:"m,omitempty"`
	Label string `json:"l,omitempty"`
}

type SnapC struct {
	ID    string  `json:"i"`
	L     int     `json:"l"`
	Owner int     `json:"o"`
	Idx   int     `json:"x"`
	N     int     `json:"n"`
	UX    float64 `json:"ux"`
	UY    float64 `json:"uy"`
	Rot   float64 `json:"r"`
	Sc    float64 `json:"s"`
	Up    bool    `json:"u,omitempty"`
	Z     int     `json:"z"`
	Mode  int     `json:"d,omitempty"`
	Glow  bool    `json:"g,omitempty"`
	New   bool    `json:"w,omitempty"`
	Tempt bool    `json:"tp,omitempty"`
}

type Snap struct {
	Hand    int     `json:"hand"`
	Seat    int     `json:"seat"`
	Players []SnapP `json:"players"`
	Cards   []SnapC `json:"cards"`
	Turn    int     `json:"turn"`
	Prompt  string  `json:"prompt,omitempty"`
	Pick    int     `json:"pick"`
	Pace    float64 `json:"pace"`
}

/* ---------------- a connection ---------------- */

type peer struct {
	conn     net.Conn
	out      chan []byte
	name     string
	seat     int
	pending  []*Ev
	lastSnap string
	res      *Result // results card waiting to be sent
	closed   bool
}

func newPeer(c net.Conn) *peer {
	p := &peer{conn: c, out: make(chan []byte, 512)}
	go func() {
		w := bufio.NewWriter(c)
		for b := range p.out {
			c.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := w.Write(b); err != nil {
				break
			}
			if len(p.out) == 0 {
				if w.Flush() != nil {
					break
				}
			}
		}
		c.Close()
	}()
	return p
}

func (p *peer) send(m *Msg) {
	if p.closed {
		return
	}
	b, _ := json.Marshal(m)
	b = append(b, '\n')
	select {
	case p.out <- b:
	default: // far behind: give up on this connection
		p.close()
	}
}

func (p *peer) close() {
	if !p.closed {
		p.closed = true
		close(p.out)
		p.conn.Close()
	}
}

// read delivers each message to fn (called with g.mu held); done runs when the line drops.
func (g *Game) readPeer(c net.Conn, fn func(*Msg), done func()) {
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var m Msg
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		g.mu.Lock()
		fn(&m)
		g.mu.Unlock()
	}
	g.mu.Lock()
	done()
	g.mu.Unlock()
}

/* ---------------- hosting ---------------- */

type hostState struct {
	ln     net.Listener
	lobby  []*peer // guests who have joined, in order
	closed bool
}

// hostTable opens this computer for guests (g.mu held).
func (g *Game) hostTable() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", netPort))
	if err != nil {
		return fmt.Errorf("Couldn't open a table on port %d. Is another copy of Old Maid hosting already?", netPort)
	}
	h := &hostState{ln: ln}
	g.host, g.mode, g.showLobby = h, modeHost, true
	g.started = false
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go g.serveGuest(h, c)
		}
	}()
	return nil
}

func (g *Game) serveGuest(h *hostState, c net.Conn) {
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(10 * time.Second)
	}
	p := newPeer(c)
	joined := false
	g.readPeer(c, func(m *Msg) {
		if h.closed || g.host != h {
			p.send(&Msg{T: "bye", Text: "The host closed the table."})
			p.close()
			return
		}
		switch m.T {
		case "hello":
			switch {
			case joined:
			case m.Ver != version:
				p.send(&Msg{T: "bye", Text: fmt.Sprintf("Your Old Maid is version %s, but the host has %s. Update both games to play together.", m.Ver, version)})
				p.close()
			case g.started:
				p.send(&Msg{T: "bye", Text: "That table is already playing. Ask the host to go back to the lobby, then join again."})
				p.close()
			case len(h.lobby) >= 3:
				p.send(&Msg{T: "bye", Text: "That table is full (four players)."})
				p.close()
			default:
				joined = true
				p.name = g.uniqueName(cleanName(m.Name, "Guest"))
				h.lobby = append(h.lobby, p)
				if g.save.Settings.Opponents+1 < len(h.lobby)+1 {
					g.save.Settings.Opponents = len(h.lobby)
				}
				g.snd.play("tick")
				g.sendLobby()
			}
		case "pick":
			for _, pl := range g.players {
				if pl.peer == p {
					for _, cd := range g.table {
						if cd.id == m.Card {
							g.resolvePick(pl.seat, cd)
						}
					}
				}
			}
		}
	}, func() {
		p.close()
		g.guestLeft(h, p)
	})
}

func (g *Game) uniqueName(n string) string {
	taken := map[string]bool{cleanName(g.save.Settings.Name, "Host"): true}
	if g.host != nil {
		for _, p := range g.host.lobby {
			taken[p.name] = true
		}
	}
	name := n
	for i := 2; taken[name]; i++ {
		name = n + " " + itoa(i)
	}
	return name
}

func cleanName(s, def string) string {
	s = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, s))
	if len([]rune(s)) > 16 {
		s = string([]rune(s)[:16])
	}
	if s == "" {
		return def
	}
	return s
}

func (g *Game) guestLeft(h *hostState, p *peer) {
	if g.host != h {
		return
	}
	for i, q := range h.lobby {
		if q == p {
			h.lobby = append(h.lobby[:i], h.lobby[i+1:]...)
			break
		}
	}
	for _, pl := range g.players {
		if pl.peer == p {
			g.dropPlayer(pl)
		}
	}
	if !g.started {
		g.sendLobby()
	}
}

func (g *Game) lobbyInfo() *Lobby {
	l := &Lobby{Host: cleanName(g.save.Settings.Name, "Host"), Seats: g.save.Settings.Opponents + 1}
	l.Names = []string{l.Host}
	if g.host != nil {
		for _, p := range g.host.lobby {
			l.Names = append(l.Names, p.name)
		}
	}
	return l
}

func (g *Game) sendLobby() {
	if g.host == nil {
		return
	}
	l := g.lobbyInfo()
	for i, p := range g.host.lobby {
		p.send(&Msg{T: "lobby", Seat: i + 1, Lobby: l})
	}
}

// hostDeal starts a hand for everyone at the table.
func (g *Game) hostDeal() {
	h := g.host
	n := max(g.save.Settings.Opponents+1, len(h.lobby)+1)
	n = min(n, 4)
	g.save.Settings.Opponents = n - 1
	g.plan = []seatPlan{{kind: pkLocal, name: cleanName(g.save.Settings.Name, "Host")}}
	for _, p := range h.lobby {
		if !p.closed {
			g.plan = append(g.plan, seatPlan{kind: pkRemote, name: p.name, peer: p})
		}
	}
	for len(g.plan) < n {
		g.plan = append(g.plan, seatPlan{kind: pkAI})
	}
	g.showLobby = false
	g.newGame()
}

// closeTable ends hosting and tells everyone (g.mu held).
func (g *Game) closeTable() {
	if g.host == nil {
		return
	}
	h := g.host
	h.closed = true
	h.ln.Close()
	for _, p := range h.lobby {
		p.send(&Msg{T: "bye", Text: "The host closed the table."})
		p.close()
	}
	g.host = nil
}

func (g *Game) snapFor(seat int) *Snap {
	s := &Snap{Hand: g.gameID, Seat: seat, Turn: g.turnSeat, Pick: -1, Pace: g.pace * g.rush}
	for _, p := range g.players {
		sp := SnapP{Name: p.name, Kind: p.kind, Out: p.out, Place: p.place, Maid: p.maidHolder, Label: p.label}
		if p.hasFace {
			f := p.face
			sp.Face = &f
		}
		s.Players = append(s.Players, sp)
	}
	for _, c := range g.table {
		s.Cards = append(s.Cards, SnapC{ID: c.id, L: c.L, Owner: c.owner, Idx: c.idx, N: c.n, UX: round3(c.ux), UY: round3(c.uy), Rot: round3(c.rot), Sc: c.sc, Up: c.up, Z: c.z, Mode: c.mode, Glow: c.glow, New: c.isNew, Tempt: c.tempt})
	}
	if g.promptFn != nil {
		s.Prompt = g.promptFn(seat)
	}
	if g.busy != nil && g.busy.seat == seat {
		s.Pick = g.busy.from.seat
	}
	return s
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// flushNet sends each guest what changed this frame (g.mu held).
func (g *Game) flushNet() {
	if g.host == nil {
		return
	}
	for _, pl := range g.players {
		p := pl.peer
		if pl.kind != pkRemote || p == nil || p.closed {
			continue
		}
		m := &Msg{T: "tick"}
		snap := g.snapFor(pl.seat)
		b, _ := json.Marshal(snap)
		if string(b) != p.lastSnap {
			p.lastSnap = string(b)
			m.Snap = snap
		}
		m.Ev, p.pending = p.pending, nil
		m.Res, p.res = p.res, nil
		if m.Snap != nil || len(m.Ev) > 0 || m.Res != nil {
			p.send(m)
		}
	}
}

// localIPs lists this computer's network addresses, home-network ones first.
func localIPs() []string {
	var ips []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() && n.IP.To4() != nil && !n.IP.IsLinkLocalUnicast() {
			ips = append(ips, n.IP.String())
		}
	}
	sort.SliceStable(ips, func(i, j int) bool {
		pri := func(s string) int {
			switch {
			case strings.HasPrefix(s, "192.168."):
				return 0
			case strings.HasPrefix(s, "10."):
				return 1
			case strings.HasPrefix(s, "172."):
				return 2
			}
			return 3
		}
		return pri(ips[i]) < pri(ips[j])
	})
	if len(ips) > 3 {
		ips = ips[:3]
	}
	return ips
}

/* ---------------- joining ---------------- */

type clientState struct {
	p     *peer
	seat  int
	lobby *Lobby
	hand  int
}

func joinAddr(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("Type the host's address first, for example 192.168.1.23.")
	}
	if _, _, err := net.SplitHostPort(s); err != nil {
		s = net.JoinHostPort(strings.Trim(s, "[]"), strconv.Itoa(netPort))
	}
	return s, nil
}

// joinTable connects to a host in the background (g.mu held).
func (g *Game) joinTable() {
	addr, err := joinAddr(g.save.Settings.Host)
	if err != nil {
		g.netStatus = err.Error()
		return
	}
	g.netStatus = "Knocking at " + addr + "…"
	g.joining = true
	name := cleanName(g.save.Settings.Name, "Guest")
	go func() {
		c, err := net.DialTimeout("tcp", addr, 6*time.Second)
		g.mu.Lock()
		defer g.mu.Unlock()
		g.joining = false
		if err != nil {
			g.netStatus = "Couldn't reach a table at " + addr + ". Check the address, and that the host has opened their table."
			return
		}
		if tc, ok := c.(*net.TCPConn); ok {
			tc.SetNoDelay(true)
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(10 * time.Second)
		}
		p := newPeer(c)
		cs := &clientState{p: p, seat: -1, hand: -1}
		g.client = cs
		p.send(&Msg{T: "hello", Name: name, Ver: version})
		go g.readPeer(c, func(m *Msg) {
			if g.client != cs {
				return
			}
			g.clientMsg(cs, m)
		}, func() {
			if g.client != cs {
				return
			}
			g.leaveClient()
			if g.notice == "" {
				g.notice = "The connection to the host was lost."
			}
		})
	}()
}

func (g *Game) clientMsg(cs *clientState, m *Msg) {
	switch m.T {
	case "bye":
		g.notice = m.Text
		g.leaveClient()
	case "lobby":
		if cs.seat < 0 {
			g.snd.play("tick")
		}
		cs.seat, cs.lobby = m.Seat, m.Lobby
		g.mode, g.showOnline, g.showTitle, g.showLobby = modeClient, false, false, !g.started
		g.netStatus = ""
	case "tick":
		if m.Snap != nil {
			g.applySnap(cs, m.Snap)
		}
		for _, ev := range m.Ev {
			g.applyEv(ev)
		}
		if m.Res != nil {
			g.clientRes = m.Res
			if g.autoplay {
				fmt.Printf("CLIENTRES %s | %s\n", m.Res.Title, m.Res.Sub)
				g.shot("result")
			}
			if m.Res.Player {
				g.record(m.Res)
			}
			g.showResult = true
		}
	}
}

// applySnap makes this guest's table match the host's.
func (g *Game) applySnap(cs *clientState, s *Snap) {
	g.mode, g.started, g.showLobby, g.showTitle = modeClient, true, false, false
	fresh := s.Hand != cs.hand
	if fresh {
		cs.hand = s.Hand
		g.showResult, g.clientRes = false, nil
		g.parts = nil
		for _, c := range g.deck {
			c.flip.jump(1)
		}
	}
	g.viewer, g.turnSeat, g.clientPrompt, g.clientPick, g.pace = s.Seat, s.Turn, s.Prompt, s.Pick, s.Pace
	if len(g.players) != len(s.Players) {
		g.players = make([]*Player, len(s.Players))
		for i := range g.players {
			g.players[i] = &Player{seat: i}
		}
	}
	for i, sp := range s.Players {
		p := g.players[i]
		p.name, p.kind, p.out, p.place, p.maidHolder, p.label = sp.Name, sp.Kind, sp.Out, sp.Place, sp.Maid, sp.Label
		p.hasFace = sp.Face != nil
		if sp.Face != nil {
			p.face = *sp.Face
		}
		p.hand = p.hand[:0]
	}
	byID := map[string]*Card{}
	for _, c := range g.deck {
		byID[c.id] = c
	}
	g.table = g.table[:0]
	for _, sc := range s.Cards {
		c := byID[sc.ID]
		if c == nil {
			continue
		}
		c.L, c.owner, c.idx, c.n, c.ux, c.uy, c.rot, c.sc = sc.L, sc.Owner, sc.Idx, sc.N, sc.UX, sc.UY, sc.Rot, sc.Sc
		c.up, c.z, c.mode, c.glow, c.isNew, c.tempt = sc.Up, sc.Z, sc.Mode, sc.Glow, sc.New, sc.Tempt
		g.table = append(g.table, c)
		if c.L == lHand && c.owner >= 0 && c.owner < len(g.players) {
			g.players[c.owner].hand = append(g.players[c.owner].hand, c)
		}
	}
	for _, p := range g.players {
		sort.Slice(p.hand, func(i, j int) bool { return p.hand[i].idx < p.hand[j].idx })
	}
	// a new hand (or just joined): start every card where it is, rather than flying in
	if fresh {
		g.measure()
		for _, c := range g.table {
			x, y, r, sc := g.cardTarget(c)
			c.ax.jump(x)
			c.ay.jump(y)
			c.ar.jump(r)
			c.as.jump(sc)
		}
	}
}

func (g *Game) leaveClient() {
	if g.client != nil {
		g.client.p.close()
		g.client = nil
	}
	g.toTitle()
}

// toTitle puts this screen back to the title card in single-player mode.
func (g *Game) toTitle() {
	g.gameID++
	close(g.done)
	g.done = make(chan struct{})
	g.mode, g.plan = modeSolo, nil
	g.started, g.finished, g.busy = false, false, nil
	g.players, g.table, g.parts = nil, nil, nil
	g.showResult, g.showLobby, g.showOnline, g.showHotseat = false, false, false, false
	g.passTo, g.revealed, g.promptFn, g.clientPrompt, g.clientRes = -1, true, nil, "", nil
	g.bannerUntil = time.Time{}
	g.bannerA.jump(0)
	g.showTitle = true
}
