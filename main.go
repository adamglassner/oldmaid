// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
//
// A native Windows game written in Go with Ebitengine. Your record and settings are saved in
// %APPDATA%\Ironite\Old Maid\save.json.
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const productName = "Old Maid"

var version = "dev" // set at build time from VERSION

/* ---------------- saved settings & record ---------------- */

type Settings struct {
	Music     bool   `json:"music"`
	Sfx       bool   `json:"sfx"`
	Vol       int    `json:"vol"`
	Opponents int    `json:"opponents"`
	Speed     string `json:"speed"`
	// multiplayer
	Name  string        `json:"name,omitempty"`
	Host  string        `json:"host,omitempty"`
	Seats []SeatSetting `json:"seats,omitempty"`
}

// SeatSetting remembers a Pass & Play chair.
type SeatSetting struct {
	Name  string `json:"name"`
	Human bool   `json:"human"`
}

type Stats struct {
	Played int `json:"played"`
	Safe   int `json:"safe"`
	Maid   int `json:"maid"`
	First  int `json:"first"`
	Streak int `json:"streak"`
	Best   int `json:"best"`
}

type SaveData struct {
	Settings Settings `json:"settings"`
	Stats    Stats    `json:"stats"`
}

func dataDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "Ironite", productName)
}

func loadSave() SaveData {
	s := SaveData{Settings: Settings{Music: true, Sfx: true, Vol: 60, Opponents: 3, Speed: "normal"}}
	if d := dataDir(); d != "" {
		if b, err := os.ReadFile(filepath.Join(d, "save.json")); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	if _, ok := paceOf[s.Settings.Speed]; !ok {
		s.Settings.Speed = "normal"
	}
	if s.Settings.Opponents < 1 || s.Settings.Opponents > 3 {
		s.Settings.Opponents = 3
	}
	s.Settings.Vol = max(0, min(100, s.Settings.Vol))
	return s
}

func (g *Game) writeSave() {
	g.saveDue = time.Time{}
	d := dataDir()
	if d == "" || g.noSave {
		return
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return
	}
	b, _ := json.MarshalIndent(g.save, "", "  ")
	tmp := filepath.Join(d, "save.json.tmp")
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, filepath.Join(d, "save.json"))
	}
}

func (g *Game) persist(now bool) {
	if now {
		g.writeSave()
	} else {
		g.saveDue = time.Now().Add(250 * time.Millisecond)
	}
}

/* ---------------- the game ---------------- */

type Game struct {
	mu sync.Mutex

	save   SaveData
	noSave bool
	snd    *Sound

	// hand state (see game.go)
	gameID           int
	done             chan struct{}
	deck, table      []*Card
	players          []*Player
	turn             int
	finished         bool
	started          bool
	rush, pace       float64
	testSpeed        float64
	placeN, discards int
	busy             *pickReq

	// who is playing
	mode     int
	plan     []seatPlan
	viewer   int // the seat this screen shows (you)
	turnSeat int
	revealed bool // pass and play: the viewer's hand may be shown
	passTo   int  // pass and play: waiting for this seat to take the computer
	readyCh  chan bool
	promptFn func(v int) string
	results  map[int]*Result

	// online
	host              *hostState
	client            *clientState
	clientPrompt      string
	clientPick        int
	clientRes         *Result
	netStatus, notice string
	joining           bool
	focus             string

	// screens
	showTitle, showRules, showCredits, showResult bool
	showHotseat, showOnline, showLobby            bool
	scroll, scrollMax                             float64

	// table layout (logical pixels)
	W, H           int
	Wd, Ht, cw, ch float64

	// effects
	promptText                        string
	promptOn                          bool
	promptA                           anim
	bannerTitle, bannerSub, bannerKey string
	bannerA, bannerS                  anim
	bannerUntil                       time.Time
	bannerImg                         *ebiten.Image
	parts                             []*particle
	motes                             []*mote
	bubbles                           map[string]*ebiten.Image
	room, veil, feltBG, layer, layer2 *ebiten.Image
	roomW, roomH                      int

	// input
	hits        []hit
	mx, my      float64
	pressed     string
	pressedCard *Card
	volX, volW  float64

	nowPlaying string
	lastFrame  time.Time
	lastTune   time.Time
	saveDue    time.Time

	// automated checks
	autoplay    bool
	facesOut    string
	facesFrames int
	shotName    string
	shotAt      time.Time
	shotCount   map[string]int
	lastFPS     time.Time
	born        time.Time
	games       int
	gamesLeft   int
	quit        bool
}

func newGameState() *Game {
	g := &Game{done: make(chan struct{}), save: loadSave(), rush: 1, pace: 1, testSpeed: 1, showTitle: true,
		bubbles: map[string]*ebiten.Image{}, motes: newMotes(), lastFrame: time.Now(), W: 1920, H: 1080,
		born: time.Now(), passTo: -1, revealed: true, shotCount: map[string]int{}, turnSeat: -1, clientPick: -1, results: map[int]*Result{}}
	for _, s := range "SHDC" {
		for r := 1; r <= 13; r++ {
			if r == 12 && s == 'C' {
				continue // one queen leaves the deck...
			}
			id := string(s) + itoa(r)
			g.deck = append(g.deck, &Card{id: id, r: r, s: byte(s), red: s == 'H' || s == 'D', maid: r == 12 && s == 'S', img: imgFace[id]}) // ...so her black sister has no partner
		}
	}
	g.testHooks()
	g.snd = newSound(float64(g.save.Settings.Vol)/100, g.save.Settings.Music, g.save.Settings.Sfx)
	g.measure()
	return g
}

func (g *Game) Update() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	dt := math.Min(50, float64(now.Sub(g.lastFrame).Microseconds())/1000)
	g.lastFrame = now
	g.snd.update()
	g.animate(dt)
	g.updateView()
	g.handleInput()
	g.flushNet()
	if now.Sub(g.lastTune) > time.Second {
		g.lastTune = now
		g.nowPlaying = g.snd.tuneName()
	}
	if !g.saveDue.IsZero() && now.After(g.saveDue) {
		g.writeSave()
	}
	if g.autoplay && now.Sub(g.lastFPS) > 3*time.Second {
		g.lastFPS = now
		fmt.Printf("FPS %.1f TPS %.1f\n", ebiten.ActualFPS(), ebiten.ActualTPS())
	}
	if g.quit {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.room == nil || g.roomW != g.W || g.roomH != g.H {
		g.roomW, g.roomH = g.W, g.H
		g.room = renderRoom(g.W, g.H, g.Wd, g.Ht)
		g.veil = renderVeil(g.W, g.H)
	}
	if g.facesOut != "" {
		g.facesSheet(screen)
		if g.facesFrames++; g.facesFrames == 3 {
			saveScreen(screen, g.facesOut)
			g.quit = true
		}
		return
	}
	screen.DrawImage(g.room, nil)
	g.drawTable(screen)
	g.overlay(screen)
	if p := os.Getenv("OLDMAID_SNAP"); p != "" && time.Since(g.born) > g.snapAfter() {
		saveScreen(screen, p)
		os.Setenv("OLDMAID_SNAP", "")
		if os.Getenv("OLDMAID_AUTOPLAY") != "1" {
			g.quit = true
		}
	}
	if g.shotName != "" && time.Now().After(g.shotAt) {
		saveScreen(screen, g.shotName)
		g.shotName = ""
	}
}

// Layout keeps the table 1080 logical pixels tall and as wide as the window allows,
// like the original, which filled the window at any shape.
func (g *Game) Layout(ow, oh int) (int, int) {
	H := 1080
	W := 1920
	if oh > 0 {
		W = int(math.Round(float64(H) * float64(ow) / float64(oh)))
	}
	W = max(1280, min(2700, W))
	if W != g.W || H != g.H {
		g.mu.Lock()
		g.W, g.H = W, H
		g.measure()
		if g.started {
			g.layoutAll()
		}
		g.mu.Unlock()
	}
	return W, H
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			crash(r)
		}
	}()
	if !singleInstance() {
		return
	}
	loadAssets()
	initShaders()
	g := newGameState()
	ebiten.SetWindowTitle(productName + " " + version)
	ebiten.SetWindowIcon([]image.Image{iconImage})
	mw, mh := 1920, 1080
	if m := ebiten.Monitor(); m != nil {
		if w, h := m.Size(); w > 0 && h > 0 {
			mw, mh = w, h
		}
	}
	ww := min(1600, mw*86/100)
	wh := min(ww*9/16, mh*80/100)
	ebiten.SetWindowSize(ww, wh)
	ebiten.SetWindowSizeLimits(1024, 600, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	if os.Getenv("OLDMAID_NOVSYNC") == "1" { // virtual displays in the automated checks
		ebiten.SetVsyncEnabled(false)
	}
	if pf := os.Getenv("OLDMAID_CPUPROF"); pf != "" {
		if f, err := os.Create(pf); err == nil {
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
	}
	err := ebiten.RunGame(g)
	g.mu.Lock()
	g.closeTable()
	if g.client != nil {
		g.client.p.close()
	}
	g.writeSave()
	g.mu.Unlock()
	if err != nil && err != ebiten.Termination {
		fatal(err.Error())
	}
}

// crash records an unexpected error so it can be reported, then tells the player.
func crash(r any) {
	msg := fmt.Sprintf("Old Maid %s stopped unexpectedly: %v\n\n%s", version, r, debug.Stack())
	_ = os.WriteFile(filepath.Join(os.TempDir(), "IroniteOldMaid.log"), []byte(msg), 0o644)
	fatal(fmt.Sprintf("Sorry, Old Maid stopped unexpectedly.\n\n%v\n\nDetails were saved to %s", r, filepath.Join(os.TempDir(), "IroniteOldMaid.log")))
	os.Exit(1)
}

/* ---------------- automated checks ----------------
   OLDMAID_AUTOPLAY=1 plays your turns for you, OLDMAID_SPEED scales every delay,
   OLDMAID_GAMES=n plays n hands and quits, OLDMAID_PANEL opens a panel at start,
   OLDMAID_NOSAVE=1 leaves the saved record alone. Used by the release checks only. */

func (g *Game) testHooks() {
	if os.Getenv("OLDMAID_NOSAVE") == "1" {
		g.noSave = true
	}
	if v, err := strconv.ParseFloat(os.Getenv("OLDMAID_SPEED"), 64); err == nil && v > 0 {
		g.testSpeed = v
	}
	g.facesOut = os.Getenv("OLDMAID_FACES")
	if os.Getenv("OLDMAID_AUTOPLAY") != "1" {
		switch os.Getenv("OLDMAID_PANEL") {
		case "rules":
			g.showRules = true
		case "credits":
			g.showCredits = true
		case "hotseat", "online":
			g.action(os.Getenv("OLDMAID_PANEL"))
		}
		return
	}
	g.autoplay = true
	g.games, _ = strconv.Atoi(os.Getenv("OLDMAID_GAMES"))
	g.gamesLeft = g.games
	if n, err := strconv.Atoi(os.Getenv("OLDMAID_PLAYERS")); err == nil {
		g.save.Settings.Opponents = n - 1
	}
	if n := os.Getenv("OLDMAID_NAME"); n != "" {
		g.save.Settings.Name = n
	}
	g.showTitle = false
	switch {
	case os.Getenv("OLDMAID_HOTSEAT") != "": // number of people
		k, _ := strconv.Atoi(os.Getenv("OLDMAID_HOTSEAT"))
		g.save.Settings.Seats = nil
		for i := 0; i < 4; i++ {
			g.save.Settings.Seats = append(g.save.Settings.Seats, SeatSetting{Name: []string{"Ada", "Ben", "Cora", "Dan"}[i], Human: i < k})
		}
		g.spawn(func() { g.sleep(400); g.startHotseat() })
	case os.Getenv("OLDMAID_HOST") != "": // number of guests to wait for
		k, _ := strconv.Atoi(os.Getenv("OLDMAID_HOST"))
		g.spawn(func() {
			if err := g.hostTable(); err != nil {
				fmt.Println("HOSTERR", err)
				g.quit = true
				return
			}
			fmt.Println("HOSTING")
			for len(g.host.lobby) < k {
				g.sleep(100)
			}
			g.sleep(500)
			g.hostDeal()
		})
	case os.Getenv("OLDMAID_JOIN") != "":
		g.save.Settings.Host = os.Getenv("OLDMAID_JOIN")
		g.spawn(func() { g.sleep(300); g.joinTable() })
		go g.clientBot()
	default:
		g.spawn(func() { g.sleep(400); g.newGame() })
	}
}

// clientBot plays a joined guest's turns and reports what it sees (automated checks only).
func (g *Game) clientBot() {
	leaveAt := time.Time{}
	if ms, err := strconv.Atoi(os.Getenv("OLDMAID_LEAVE_MS")); err == nil {
		leaveAt = time.Now().Add(time.Duration(ms) * time.Millisecond)
	}
	results, lastHand := 0, -1
	for {
		time.Sleep(150 * time.Millisecond)
		g.mu.Lock()
		if !leaveAt.IsZero() && time.Now().After(leaveAt) {
			fmt.Println("CLIENT leaving mid-hand")
			g.quit = true
			g.mu.Unlock()
			return
		}
		if g.notice != "" {
			fmt.Println("CLIENT notice:", g.notice)
			g.quit = true
			g.mu.Unlock()
			return
		}
		if g.clientPick >= 0 && g.clientPick < len(g.players) && len(g.players[g.clientPick].hand) > 0 && g.client != nil {
			h := g.players[g.clientPick].hand
			g.pickCard(h[time.Now().Nanosecond()%len(h)])
			g.shot("clientpick")
		}
		if g.clientRes != nil && g.client != nil && g.client.hand != lastHand {
			lastHand = g.client.hand
			results++
			if g.games > 0 && results >= g.games {
				g.quit = true
			}
		}
		g.mu.Unlock()
	}
}

func (g *Game) shot(name string) {
	if g.autoplay {
		fmt.Println("SHOT", name)
		if d := os.Getenv("OLDMAID_SHOTDIR"); d != "" && g.shotCount[name] < 2 {
			g.shotCount[name]++
			g.shotAt = time.Now().Add(350 * time.Millisecond)
			g.shotName = filepath.Join(d, fmt.Sprintf("%s-%s-%d.png", os.Getenv("OLDMAID_NAME"), name, g.shotCount[name]))
		}
	}
}

func (g *Game) testLog(loser *Player) {
	if !g.autoplay {
		return
	}
	cards := 0
	for _, p := range g.players {
		cards += len(p.hand)
	}
	order := ""
	for _, p := range g.players {
		order += fmt.Sprintf(" %s(%d)=%d", p.name, p.kind, p.place)
	}
	fmt.Printf("GAME mode=%d players=%d loser=%q places:%s left=%d discards=%d stats=%+v\n", g.mode, len(g.players), loser.name, order, cards, g.discards, g.save.Stats)
}

func (g *Game) afterResult() {
	if !g.autoplay || g.games == 0 {
		return
	}
	g.gamesLeft--
	if g.gamesLeft <= 0 {
		fmt.Println("DONE")
		g.sleep(2500)
		g.quit = true
		return
	}
	g.sleep(700 * g.testSpeed)
	g.action("again")
}

// facesSheet draws the original three guests and a crowd of random ones (automated checks).
func (g *Game) facesSheet(screen *ebiten.Image) {
	orig := []Look{
		{Lady: true, Stern: true, Skin: hex("#f1d0b5"), Hair: hex("#9a9aa4"), Brow: hex("#6d6d78"), HairStyle: "bun", Coat: hex("#4b2a5c"), Dark: hex("#2e1838"), Accent: hex("#d9b44a"), Collar: "lace", Pearls: true, Sleeves: true, Hat: "feather", HatColor: hex("#2e1838")},
		{Skin: hex("#eec3a2"), Hair: hex("#e8e4dc"), MustacheColor: hex("#f2eee6"), Brow: hex("#d8d2c6"), HairStyle: "short", Coat: hex("#a3242b"), Dark: hex("#6b1218"), Accent: hex("#d9b44a"), Collar: "military", Hat: "military", Mustache: true, Muttonchops: true},
		{Lady: true, Skin: hex("#f6dbc4"), Hair: hex("#b5651d"), HairStyle: "ringlets", Coat: hex("#2f7a7a"), Dark: hex("#1d4f50"), Accent: hex("#e8b4b8"), Collar: "lace", Sleeves: true, Hat: "bonnet", HatColor: hex("#f1e4c4")},
	}
	screen.Fill(hex("#2a160e"))
	sz := 150.0
	for i := 0; i < 24; i++ {
		var im *ebiten.Image
		label := ""
		if i < 3 {
			im = renderCameo(orig[i], 304)
			label = []string{"Agatha (port)", "Colonel (port)", "Pettigrew (port)"}[i]
		} else {
			f := Face{Female: i%2 == 0, Seed: uint64(i * 7919)}
			im = cameoFor(f)
			label = f.name()
		}
		x, y := 20+float64(i%8)*(sz+40), 20+float64(i/8)*(sz+60)
		drawCentered(screen, im, x+sz/2, y+sz/2, sz, 1)
		txt(screen, label, font(fSerif, 15), x+sz/2, y+sz+16, .5, cCream, nil)
	}
}

func saveScreen(screen *ebiten.Image, path string) {
	b := screen.Bounds()
	img := image.NewRGBA(b)
	screen.ReadPixels(img.Pix)
	if f, err := os.Create(path); err == nil {
		png.Encode(f, img)
		f.Close()
	}
}

func (g *Game) snapAfter() time.Duration {
	v, err := strconv.ParseFloat(os.Getenv("OLDMAID_SNAP_AT"), 64)
	if err != nil {
		v = 4
	}
	return time.Duration(v * float64(time.Second))
}
