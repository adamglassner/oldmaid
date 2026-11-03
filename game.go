// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// The rules and the flow of a hand. The flow runs as a goroutine that holds g.mu while it
// works and releases it while it waits (g.sleep), so the frame loop always sees a consistent
// table. Dealing a new hand cancels the old one at its next wait.
//
// The flow only records where each card is (a hand, the pile, the deck); every screen works
// out the positions for its own seat (view.go). Messages are written per seat too, so in an
// online game you read "You" while your friends read your name.
package main

import (
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

/* ---------------- the company ---------------- */

type Person struct {
	name, cam                   string
	pair, draw, out, maid, took []string
}

var people = map[string]*Person{
	"agatha": {name: "Lady Agatha", cam: "agatha",
		pair: []string{"Splendid.", "As expected.", "Quite so.", "Naturally.", "A matched set."},
		draw: []string{"Let me see…", "Hmm. This one.", "Do hold still, dear."},
		out:  []string{"I shall retire to the drawing room.", "That will be all, thank you."},
		maid: []string{"Well! I never!", "How utterly vexing."},
		took: []string{"Thank you, dear.", "Hmph."}},
	"colonel": {name: "Colonel Bramble", cam: "colonel",
		pair: []string{"Ha! Another brace!", "Capital!", "Bully for me!", "Splendid shot!", "Right-ho!"},
		draw: []string{"Eeny, meeny…", "Steady now…", "Into the breach!"},
		out:  []string{"Strategic withdrawal, what!", "Mission accomplished!"},
		maid: []string{"Ambushed, by Jove!", "Confound it!"},
		took: []string{"Hm! Interesting.", "Ha! Much obliged."}},
	"pettigrew": {name: "Miss Pettigrew", cam: "pettigrew",
		pair: []string{"Oh, how lovely!", "A match!", "Delightful!", "Oh, goody!", "Hee hee!"},
		draw: []string{"Oh, which one?", "This one looks lucky!", "Ooh, eenie…"},
		out:  []string{"I'm out! Oh, what fun!", "All done! Hooray!"},
		maid: []string{"Oh, bother!", "Oh no, not her!"},
		took: []string{"Oh! Thank you!", "Ooh."}},
}

var paceOf = map[string]float64{"relaxed": 1.35, "normal": 1, "brisk": 0.62}

func pick[T any](a []T) T { return a[rand.Intn(len(a))] }

/* ---------------- game modes ---------------- */

const (
	modeSolo    = iota // you against the computer
	modeHotseat        // pass and play on one computer
	modeHost           // online, this computer runs the table
	modeClient         // online, joined someone else's table
)

const (
	pkAI     = iota
	pkLocal  // a person at this computer
	pkRemote // a person at another computer
)

/* ---------------- cards and players ---------------- */

const (
	modeNormal = iota
	modeFast
	modeInstant
)

// where a card is
const (
	lDeck   = iota // stacked in the middle (ux, uy in pixels)
	lSplit         // cut in two while shuffling (ux = side)
	lHand          // in a player's hand (owner, idx of n)
	lPile          // on the discard pile (ux, uy in card heights from the pile centre)
	lReveal        // the Old Maid, shown large at the end
)

type anim struct {
	from, to float64
	t0       time.Time
	dur      float64 // seconds
	ease     func(float64) float64
}

func (a *anim) val(now time.Time) float64 {
	if a.dur <= 0 {
		return a.to
	}
	t := now.Sub(a.t0).Seconds() / a.dur
	if t >= 1 {
		return a.to
	}
	return a.from + (a.to-a.from)*a.ease(t)
}

func (a *anim) set(to float64, now time.Time, dur float64, ease func(float64) float64) {
	if a.to == to {
		if dur == 0 {
			a.from, a.dur = to, 0
		}
		return
	}
	a.from = a.val(now)
	a.to, a.t0, a.dur, a.ease = to, now, dur, ease
}

func (a *anim) jump(v float64) { a.from, a.to, a.dur = v, v, 0 }

type Card struct {
	id        string
	r         int
	s         byte
	red, maid bool
	img       *ebiten.Image

	// the table (shared by every screen)
	L                  int
	owner, idx, n      int
	ux, uy, rot, sc    float64
	up                 bool // face up for everyone (pairs on the pile, the reveal)
	z                  int
	mode               int
	glow, isNew, tempt bool

	// this screen
	ax, ay, ar, as anim
	flip           anim // 0 face up .. 1 face down
	lift           anim // 0 resting .. 1 lifted
	pick, hover    bool
}

type Player struct {
	seat    int // place in the turn order
	name    string
	kind    int
	face    Face
	hasFace bool
	voice   *Person
	peer    *peer // remote players

	hand       []*Card
	out        bool
	place      int
	maidHolder bool
	label      string // overrides the card count ("The Old Maid!")

	// this screen
	sx, sy   float64
	turn     anim
	say      string
	sayOn    anim
	sayUntil time.Time
}

func (p *Player) human() bool { return p.kind != pkAI }

/* ---------------- timing ---------------- */

type cancelled struct{}

func (g *Game) T(k string) float64 {
	base := map[string]float64{"deal": 55, "move": 430, "flip": 380, "pair": 520, "think": 700, "tempt": 380, "quip": 1900}[k]
	return base * g.pace * g.rush * g.testSpeed
}

// sleep waits ms milliseconds with the table unlocked, then stops this hand if a new one was dealt.
func (g *Game) sleep(ms float64) {
	id, done := g.gameID, g.done
	g.mu.Unlock()
	select {
	case <-time.After(time.Duration(ms * float64(time.Millisecond))):
	case <-done:
	}
	g.mu.Lock()
	if g.gameID != id {
		panic(cancelled{})
	}
}

// wait blocks until ch delivers (table unlocked), or the hand is cancelled.
func waitFor[T any](g *Game, ch chan T) T {
	id, done := g.gameID, g.done
	g.mu.Unlock()
	var v T
	select {
	case v = <-ch:
	case <-done:
	}
	g.mu.Lock()
	if g.gameID != id {
		panic(cancelled{})
	}
	return v
}

// spawn runs fn as part of the current hand, alongside the main flow.
func (g *Game) spawn(fn func()) {
	id := g.gameID
	go func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(cancelled); !ok {
					crash(r)
				}
			}
		}()
		if g.gameID != id {
			return
		}
		fn()
	}()
}

/* ---------------- who sees what ---------------- */

// you reports whether seat v reads player p as "you".
func (g *Game) you(p *Player, v int) bool { return p.seat == v && g.mode != modeHotseat }

// nm is p's name as seat v reads it.
func (g *Game) nm(p *Player, v int) string {
	if g.you(p, v) {
		return "You"
	}
	return p.name
}

// Ev is something that happens on screen: a sound, a banner, a remark or a puff of smoke.
type Ev struct {
	K    string  `json:"k"`
	S    string  `json:"s,omitempty"` // sound name, burst kind
	A    string  `json:"a,omitempty"` // banner title, remark
	B    string  `json:"b,omitempty"` // banner subtitle
	Seat int     `json:"p,omitempty"`
	Card string  `json:"c,omitempty"`
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
	N    int     `json:"n,omitempty"`
	MS   float64 `json:"ms,omitempty"`
}

// emit shows an event on this screen and sends it to everyone online; f makes the event
// as seat v should see it (or nil to show them nothing).
func (g *Game) emit(f func(v int) *Ev) {
	if g.mode != modeHost || g.players == nil || g.players[g.viewer].human() {
		if ev := f(g.viewer); ev != nil {
			g.applyEv(ev)
		}
	}
	for _, p := range g.players {
		if p.kind == pkRemote && p.peer != nil {
			if ev := f(p.seat); ev != nil {
				p.peer.pending = append(p.peer.pending, ev)
			}
		}
	}
}

func (g *Game) sound(name string) { g.emit(func(int) *Ev { return &Ev{K: "snd", S: name} }) }

func (g *Game) soundFor(seat int, name string) {
	g.emit(func(v int) *Ev {
		if v != seat {
			return nil
		}
		return &Ev{K: "snd", S: name}
	})
}

func (g *Game) say(p *Player, kind string, chance float64) {
	if p.voice == nil || rand.Float64() > chance {
		return
	}
	lines := map[string][]string{"pair": p.voice.pair, "draw": p.voice.draw, "out": p.voice.out, "maid": p.voice.maid, "took": p.voice.took}[kind]
	line := pick(lines)
	g.emit(func(int) *Ev { return &Ev{K: "say", Seat: p.seat, A: line, MS: g.T("quip") + 400} })
}

func (g *Game) setPrompt(f func(v int) string) { g.promptFn = f }

func (g *Game) promptAll(s string) { g.promptFn = func(int) string { return s } }

/* ---------------- the table ---------------- */

func (g *Game) active() []*Player {
	var a []*Player
	for _, p := range g.players {
		if !p.out {
			a = append(a, p)
		}
	}
	return a
}

func (g *Game) nextActive(i int) int {
	for k := 1; k <= len(g.players); k++ {
		j := (i + k) % len(g.players)
		if !g.players[j].out {
			return j
		}
	}
	return -1
}

func (g *Game) localHumans() int {
	n := 0
	for _, p := range g.players {
		if p.kind == pkLocal && !p.out {
			n++
		}
	}
	return n
}

func (g *Game) humansIn() bool {
	for _, p := range g.players {
		if p.human() && !p.out {
			return true
		}
	}
	return false
}

// seatPlan describes who sits where before the cards are dealt.
type seatPlan struct {
	kind int
	name string
	peer *peer
}

// makeTable seats everyone; the computer guests get fresh faces and names every hand.
func (g *Game) makeTable() {
	plan := g.plan
	if len(plan) < 2 {
		n := g.save.Settings.Opponents + 1
		plan = []seatPlan{{kind: pkLocal, name: "You"}}
		for len(plan) < n {
			plan = append(plan, seatPlan{kind: pkAI})
		}
	}
	nAI := 0
	for _, s := range plan {
		if s.kind == pkAI {
			nAI++
		}
	}
	guests := randomGuests(nAI)
	g.players = nil
	for i, s := range plan {
		p := &Player{seat: i, kind: s.kind, name: s.name, peer: s.peer}
		if s.kind == pkAI {
			f := guests[0]
			guests = guests[1:]
			p.face, p.hasFace, p.name, p.voice = f, true, f.name(), manners[f.manner()]
		}
		if p.peer != nil {
			p.peer.seat = i
		}
		g.players = append(g.players, p)
	}
	// the screen starts at the first person at this computer
	g.viewer = 0
	for _, p := range g.players {
		if p.kind == pkLocal {
			g.viewer = p.seat
			break
		}
	}
}

func sortHuman(p *Player) {
	sort.SliceStable(p.hand, func(i, j int) bool {
		a, b := p.hand[i], p.hand[j]
		if a.r != b.r {
			return a.r < b.r
		}
		if a.red != b.red {
			return !a.red
		}
		return a.s < b.s
	})
}

func (g *Game) layoutHand(p *Player) {
	for i, c := range p.hand {
		c.L, c.owner, c.idx, c.n = lHand, p.seat, i, len(p.hand)
		c.sc = 1
		c.z = 100 + i
	}
}

func setLoc(c *Card, L int, ux, uy, rot, sc float64, z int) {
	c.L, c.ux, c.uy, c.rot, c.sc = L, ux, uy, rot, sc
	if z >= 0 {
		c.z = z
	}
}

/* ---------------- game flow ---------------- */

// newGame deals a fresh hand (called with g.mu held).
func (g *Game) newGame() {
	g.gameID++
	close(g.done)
	g.done = make(chan struct{})
	g.finished, g.started, g.rush = false, true, 1
	g.pace = paceOf[g.save.Settings.Speed]
	if g.pace == 0 {
		g.pace = 1
	}
	g.placeN, g.discards, g.busy, g.turnSeat = 1, 0, nil, -1
	g.showResult, g.passTo, g.revealed = false, -1, g.mode != modeHotseat
	g.results = map[int]*Result{}
	g.makeTable()
	g.measure()
	deck := append([]*Card(nil), g.deck...)
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	for i, c := range deck {
		c.mode, c.glow, c.isNew, c.tempt, c.up = modeInstant, false, false, false, false
		c.flip.jump(1)
		c.lift.jump(0)
		setLoc(c, lDeck, float64(i-25)*0.25, -float64(i)*0.3, (rand.Float64()-.5)*4, 1, 20+i)
	}
	g.table = deck
	g.promptAll("Shuffling…")
	g.spawn(func() {
		g.sleep(60)
		for _, c := range deck {
			c.mode = modeNormal
		}
		g.shuffleAnim(deck)
		g.promptAll("Dealing…")
		for k, c := range deck {
			p := g.players[k%len(g.players)]
			p.hand = append(p.hand, c)
			c.mode = modeFast
			if p.human() {
				sortHuman(p)
			}
			g.layoutHand(p)
			if k%2 == 0 {
				g.sound("deal")
			}
			g.sleep(g.T("deal"))
		}
		for _, c := range deck {
			c.mode = modeNormal
		}
		g.sleep(g.T("move"))
		g.promptAll("Laying down pairs…")
		for _, p := range g.players {
			g.layDownAll(p)
		}
		for _, p := range g.players {
			if len(p.hand) == 0 && !p.out {
				g.markOut(p)
			}
		}
		g.turn = 0
		g.play()
	})
}

func (g *Game) shuffleAnim(deck []*Card) {
	g.sound("shuffle")
	for rep := 0; rep < 3; rep++ {
		for i, c := range deck {
			side := -1.0
			if i%2 == 1 {
				side = 1
			}
			setLoc(c, lSplit, side, -float64(i)*0.3, side*8, 1, -1)
		}
		g.sleep(g.T("flip") * 0.55)
		for i, c := range deck {
			setLoc(c, lDeck, (rand.Float64()-.5)*3, -float64(i)*0.3, (rand.Float64()-.5)*4, 1, 20+i)
		}
		g.sleep(g.T("flip") * 0.55)
	}
}

func findPair(p *Player, c *Card) *Card {
	for _, x := range p.hand {
		if x != c && x.r == c.r && x.red == c.red {
			return x
		}
	}
	return nil
}

func without(h []*Card, a, b *Card) []*Card {
	out := h[:0:0]
	for _, x := range h {
		if x != a && x != b {
			out = append(out, x)
		}
	}
	return out
}

func (g *Game) discardPair(p *Player, a, b *Card, quick bool) {
	p.hand = without(p.hand, a, b)
	for _, c := range []*Card{a, b} {
		c.z = 900
		c.up = true
	}
	if !quick {
		a.glow, b.glow = true, true
		g.sleep(g.T("flip") * 0.9)
	}
	ang, d := rand.Float64()*math.Pi*2, math.Sqrt(rand.Float64())*0.34
	sx, sy, sr := math.Cos(ang)*d, math.Sin(ang)*d*0.8, (rand.Float64()-.5)*70
	off := 0.14 * artW / artH // 0.14 card widths, in card heights
	setLoc(a, lPile, sx-off, sy, sr-8, 0.8, 30+g.discards)
	g.discards++
	setLoc(b, lPile, sx+off, sy, sr+8, 0.8, 30+g.discards)
	g.discards++
	g.layoutHand(p)
	finish := func() {
		g.sleep(g.T("move") * 0.85)
		a.glow, b.glow = false, false
		g.sound("pair")
		n := 16
		if quick {
			n = 8
		}
		g.emit(func(int) *Ev { return &Ev{K: "burst", S: "spark", X: sx, Y: sy, N: n} })
	}
	if quick {
		g.spawn(finish)
	} else {
		finish()
	}
}

func (g *Game) layDownAll(p *Player) {
	any := false
	for found := true; found; {
		found = false
		for _, c := range p.hand {
			if m := findPair(p, c); m != nil {
				found, any = true, true
				g.discardPair(p, c, m, true)
				g.sleep(g.T("pair") * 0.45)
				break
			}
		}
	}
	if any {
		g.sleep(g.T("move"))
	}
}

func (g *Game) markOut(p *Player) {
	p.out = true
	p.place = g.placeN
	g.placeN++
	g.sound("out")
	if p.human() {
		first := p.place == 1
		g.emit(func(v int) *Ev {
			sub := "Safe and sound."
			if first {
				sub = "First to finish — splendid!"
			}
			if g.you(p, v) {
				return &Ev{K: "banner", A: "You're out!", B: sub, MS: 2400}
			}
			if g.players[v].human() || g.mode == modeHotseat {
				return &Ev{K: "banner", A: p.name + " is out!", B: sub, MS: 2000}
			}
			return nil
		})
		if !g.humansIn() {
			g.rush = 0.55
		}
	} else {
		g.say(p, "out", 1)
	}
}

func (g *Game) play() {
	for {
		if len(g.active()) <= 1 {
			g.endGame()
			return
		}
		p := g.players[g.turn]
		if !p.out {
			from := g.players[g.nextActive(g.turn)]
			g.turnSeat = p.seat
			var c *Card
			if p.human() {
				c = g.humanPick(p, from)
			} else {
				c = g.aiPick(p, from)
			}
			g.takeCard(p, from, c)
			// pass and play: hide this hand before the next person looks
			if g.mode == modeHotseat && p.kind == pkLocal && g.localHumans() > 1 {
				g.sleep(g.T("move") * 1.2)
				g.revealed = false
			}
		}
		n := g.nextActive(g.turn)
		if n < 0 {
			g.endGame()
			return
		}
		g.turn = n
	}
}

type pickReq struct {
	seat int
	from *Player
	ch   chan *Card
}

func (g *Game) humanPick(p, from *Player) *Card {
	// pass and play: hand the computer over first
	if g.mode == modeHotseat && p.kind == pkLocal {
		if g.viewer != p.seat && g.localHumans() > 1 {
			if g.revealed { // turn the last hand face down before anyone takes the seat
				g.revealed = false
				g.sleep(450)
			}
			g.passTo = p.seat
			g.readyCh = make(chan bool, 1)
			if g.autoplay {
				ch := g.readyCh
				go func() { time.Sleep(time.Duration(g.T("think")) * time.Millisecond); ch <- true }()
			}
			g.promptAll("")
			g.shot("pass")
			waitFor(g, g.readyCh)
			g.passTo = -1
		}
		g.viewer, g.revealed = p.seat, true
	}
	g.setPrompt(func(v int) string {
		switch {
		case v == p.seat && g.mode == modeHotseat:
			return p.name + ", your turn — draw a card from " + from.name
		case v == p.seat:
			return "Your turn — draw a card from " + g.nm(from, v)
		case g.you(from, v):
			return p.name + " is choosing one of your cards…"
		}
		return p.name + " draws from " + g.nm(from, v)
	})
	req := &pickReq{seat: p.seat, from: from, ch: make(chan *Card, 1)}
	g.busy = req
	if (g.autoplay && p.kind == pkLocal) || p.kind == pkAI { // automated checks, or the player left the table
		g.autoPick(req)
	}
	g.shot("pick")
	return waitFor(g, req.ch)
}

func (g *Game) autoPick(req *pickReq) {
	go func() {
		time.Sleep(time.Duration(g.T("think")*0.5) * time.Millisecond)
		g.mu.Lock()
		if g.busy == req && len(req.from.hand) > 0 {
			g.resolvePick(req.seat, pick(req.from.hand))
		}
		g.mu.Unlock()
	}()
}

// resolvePick takes seat's chosen card (g.mu held).
func (g *Game) resolvePick(seat int, c *Card) {
	req := g.busy
	if req == nil || req.seat != seat || c == nil {
		return
	}
	found := false
	for _, x := range req.from.hand {
		if x == c {
			found = true
		}
	}
	if !found {
		return
	}
	g.busy = nil
	g.promptAll("")
	req.ch <- c
}

func (g *Game) aiPick(p, from *Player) *Card {
	if from.human() {
		g.setPrompt(func(v int) string {
			if g.you(from, v) {
				return p.name + " is choosing one of your cards…"
			}
			return p.name + " draws from " + g.nm(from, v)
		})
		g.say(p, "draw", 0.45)
		var last *Card
		for i, n := 0, 2+rand.Intn(2); i < n; i++ {
			if last != nil {
				last.tempt = false
			}
			last = pick(from.hand)
			last.tempt = true
			g.soundFor(from.seat, "hover")
			g.sleep(g.T("tempt"))
		}
		if last != nil {
			last.tempt = false
		}
		choice := pick(from.hand)
		if rand.Float64() < 0.5 && last != nil {
			choice = last
		}
		choice.tempt = true
		g.sleep(g.T("tempt") * 0.6)
		choice.tempt = false
		return choice
	}
	g.setPrompt(func(v int) string { return p.name + " draws from " + g.nm(from, v) })
	g.sleep(g.T("think") * (0.6 + rand.Float64()*0.6))
	return pick(from.hand)
}

func (g *Game) takeCard(p, from *Player, c *Card) {
	from.hand = without(from.hand, c, nil)
	if p.human() {
		p.hand = append(p.hand, c)
		sortHuman(p)
	} else {
		i := rand.Intn(len(p.hand) + 1)
		p.hand = append(p.hand[:i], append([]*Card{c}, p.hand[i:]...)...)
	}
	c.up = false
	c.z = 950
	g.sound("slide")
	g.layoutHand(from)
	g.layoutHand(p)
	c.z = 950
	if p.human() {
		c.isNew = true
	}
	g.sleep(g.T("move"))
	if p.human() {
		g.soundFor(p.seat, "flip")
	}
	switch {
	case c.maid && p.human():
		g.soundFor(p.seat, "maid")
		g.emit(func(v int) *Ev {
			if v != p.seat {
				return nil
			}
			sub := "You have drawn the Old Maid."
			if g.mode == modeHotseat {
				sub = p.name + ", you have drawn the Old Maid."
			}
			return &Ev{K: "burst", S: "puff", Card: c.id, N: 18, A: "Oh dear!", B: sub, MS: 2200}
		})
		g.shot("maid")
		g.sleep(g.T("pair") * 1.4)
	case c.maid && from.human():
		g.emit(func(v int) *Ev {
			if !g.you(from, v) {
				return nil
			}
			return &Ev{K: "banner", S: "phew", A: "Phew!", B: g.nm(p, v) + " took the Old Maid.", MS: 2000}
		})
		g.say(p, "took", 0.6)
	case from.human() && !p.human():
		g.say(p, "took", 0.15)
	}
	if m := findPair(p, c); m != nil {
		g.discardPair(p, c, m, false)
		g.say(p, "pair", 0.4)
	}
	c.isNew = false
	if len(from.hand) == 0 && !from.out {
		g.markOut(from)
	}
	if len(p.hand) == 0 && !p.out {
		g.markOut(p)
	}
	g.sleep(g.T("move") * 0.35)
}

/* ---------------- the end of a hand ---------------- */

type resLine struct {
	Text string `json:"t"`
	Maid bool   `json:"m,omitempty"`
	You  bool   `json:"y,omitempty"`
}

// Result is the closing card as one seat sees it.
type Result struct {
	Title  string    `json:"title"`
	Sub    string    `json:"sub"`
	Places []resLine `json:"places"`
	Maid   bool      `json:"maid"`  // this seat is the Old Maid
	First  bool      `json:"first"` // this seat went out first
	Player bool      `json:"player"`
}

func (g *Game) resultFor(v int, loser *Player) *Result {
	me := g.players[v]
	r := &Result{Maid: loser == me, First: me.place == 1 && loser != me, Player: me.human()}
	switch {
	case g.mode == modeHotseat && loser.human():
		r.Title, r.Sub = loser.name+" is the Old Maid!", "Better luck at the next deal, dear."
	case g.mode == modeHotseat:
		r.Title, r.Sub = "Well Played, All!", loser.name+" is left holding the Old Maid."
	case loser == me:
		r.Title, r.Sub = "You're the Old Maid!", "Better luck at the next deal, dear."
	case me.place == 1:
		r.Title, r.Sub = "Splendid!", g.nm(loser, v)+" is left holding the Old Maid."
	default:
		r.Title, r.Sub = "Safe and Sound!", g.nm(loser, v)+" is left holding the Old Maid."
	}
	order := append([]*Player(nil), g.players...)
	sort.SliceStable(order, func(i, j int) bool { return order[i].place < order[j].place })
	for _, p := range order {
		line := g.nm(p, v)
		if p.maidHolder {
			line += " — the Old Maid"
		} else if p.place == 1 {
			line += " — first out"
		}
		r.Places = append(r.Places, resLine{line, p.maidHolder, g.you(p, v)})
	}
	return r
}

func (g *Game) endGame() {
	if g.finished {
		return
	}
	g.finished = true
	g.turnSeat = -1
	g.promptAll("")
	var loser *Player
	if a := g.active(); len(a) > 0 {
		loser = a[0]
	}
	var maid *Card
	for _, c := range g.deck {
		if c.maid {
			maid = c
		}
	}
	if loser == nil {
		for _, p := range g.players {
			for _, c := range p.hand {
				if c.maid {
					loser = p
				}
			}
		}
	}
	loser.out, loser.place, loser.maidHolder = true, len(g.players), true
	loser.label = "The Old Maid!"
	loser.hand = nil
	maid.up = true
	setLoc(maid, lReveal, 0, 0, 0, 2.2, 3000)
	g.sleep(g.T("move") * 1.4)
	g.emit(func(int) *Ev { return &Ev{K: "burst", S: "puff", N: 34} })
	g.emit(func(v int) *Ev {
		lost := loser.seat == v || (g.mode == modeHotseat && loser.human())
		if lost {
			return &Ev{K: "snd", S: "lose"}
		}
		n := 60
		if g.players[v].place == 1 {
			n = 140
		}
		return &Ev{K: "burst", S: "petal", N: n, A: "win"}
	})
	if !loser.human() {
		g.say(loser, "maid", 1)
	}
	for _, p := range g.players {
		g.results[p.seat] = g.resultFor(p.seat, loser)
	}
	// your record counts the hands you play as yourself
	if g.mode == modeSolo || g.mode == modeHost {
		g.record(g.results[g.viewer])
	}
	g.testLog(loser)
	g.sleep(1600 * math.Max(g.testSpeed, 0.1))
	g.showResult = true
	for _, p := range g.players {
		if p.kind == pkRemote && p.peer != nil {
			p.peer.res = g.results[p.seat]
		}
	}
	g.shot("result")
	g.afterResult()
}

func (g *Game) record(r *Result) {
	st := &g.save.Stats
	st.Played++
	if r.Maid {
		st.Maid++
		st.Streak = 0
	} else {
		st.Safe++
		st.Streak++
		st.Best = max(st.Best, st.Streak)
		if r.First {
			st.First++
		}
	}
	g.persist(true)
}

func (g *Game) recordText() string {
	s := g.save.Stats
	if s.Played == 0 {
		return "No hands played yet."
	}
	return strings.Join([]string{"Hands played: " + itoa(s.Played), "Safe: " + itoa(s.Safe), "Old Maid: " + itoa(s.Maid), "First out: " + itoa(s.First), "Best safe streak: " + itoa(s.Best)}, " · ")
}

/* ---------------- someone leaves an online table ---------------- */

// dropPlayer hands a departed player's seat to a computer guest (g.mu held).
func (g *Game) dropPlayer(p *Player) {
	if p.kind != pkRemote {
		return
	}
	p.kind, p.peer = pkAI, nil
	p.voice = manners[pick([]string{"gent", "mister", "matron", "young"})]
	name := p.name
	if !strings.HasSuffix(p.name, " (computer)") {
		p.name += " (computer)"
	}
	g.emit(func(int) *Ev {
		return &Ev{K: "banner", A: name + " left the table", B: "A computer player takes their seat.", MS: 2600}
	})
	if g.busy != nil && g.busy.seat == p.seat {
		g.autoPick(g.busy)
	}
	if !g.humansIn() {
		g.rush = 0.55
	}
}
