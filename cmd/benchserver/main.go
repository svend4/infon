// Command benchserver is the browser-facing live stand for the tvcp-ai layer. It
// is a tiny local HTTP server (no sandbox) that actually RUNS infon capabilities
// on demand and streams the results to a page in your browser:
//
//	go run ./cmd/benchserver           # then open http://localhost:8086
//	go run ./cmd/benchserver -addr :9000
//
// Endpoints:
//
//	GET /              the interactive page (self-contained HTML)
//	GET /api/bench     runs pkg/bench and returns the Report as JSON
//	GET /api/figures   the catalog figure names
//	GET /api/figure?name=cat   a live glyph render of one figure (+ its address/gate)
//
// Same engine as cmd/testbench (pkg/bench), a different face: click to run, see
// it live. Nothing here is sandboxed, so it is the real interactive test bench.
package main

import (
	"encoding/json"
	"flag"
	"image"
	"image/png"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/svend4/infon/internal/codec/babe"
	"github.com/svend4/infon/pkg/arena"
	"github.com/svend4/infon/pkg/bench"
	"github.com/svend4/infon/pkg/blazon"
	"github.com/svend4/infon/pkg/brain"
	"github.com/svend4/infon/pkg/glyphqr"
	"github.com/svend4/infon/pkg/relief"
	"github.com/svend4/infon/pkg/republic"
	"github.com/svend4/infon/pkg/tangram"
	"github.com/svend4/infon/pkg/tangram7"
	"github.com/svend4/infon/pkg/world"
)

func main() {
	addr := flag.String("addr", ":8086", "listen address")
	worldBrain := flag.String("worldbrain", "", "tvcp-ai/1 URL to let a live brain direct the /img/world walk (game:world)")
	flag.Parse()

	if *worldBrain != "" {
		go generateLiveWorld(*worldBrain)
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/api/bench", apiBench)
	http.HandleFunc("/api/figures", apiFigures)
	http.HandleFunc("/api/figure", apiFigure)
	http.HandleFunc("/api/republic", apiRepublic)
	http.HandleFunc("/api/arena", apiArena)
	http.HandleFunc("/img/figure", imgFigure)
	http.HandleFunc("/img/relief", imgRelief)
	http.HandleFunc("/img/tangram7", imgTangram7)
	http.HandleFunc("/img/glyphqr", imgGlyphQR)
	http.HandleFunc("/img/blazon", imgBlazon)
	http.HandleFunc("/img/world", imgWorld)
	http.HandleFunc("/api/worldblocks", apiWorldBlocks)
	http.HandleFunc("/api/game", apiGame)
	http.HandleFunc("/api/game/summon", apiGameSummon)
	http.HandleFunc("/api/game/step", apiGameStep)
	http.HandleFunc("/api/walk", apiWalk)

	log.Printf("TVCP benchserver — open http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func apiBench(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(bench.Run())
}

func apiRepublic(w http.ResponseWriter, _ *http.Request) {
	cands := []republic.Candidate{
		{Name: "scout", Brain: brain.Local{}, Cost: func(int) int { return 3 }},
		{Name: "planner", Brain: brain.Local{}, Cost: func(int) int { return 4 }},
		{Name: "warden", Brain: brain.Local{}, Cost: func(int) int { return 5 }},
		{Name: "ranger", Brain: brain.Local{}, Cost: func(int) int { return 6 }},
	}
	res := republic.Convene(cands, 7)
	out := struct {
		republic.Result
		Field []string `json:"field"`
	}{Result: res, Field: fieldRows(res.Final)}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func fieldRows(a *arena.Arena) []string {
	if a == nil {
		return nil
	}
	rows := make([]string, a.H)
	for y := 0; y < a.H; y++ {
		line := make([]rune, a.W)
		for x := range line {
			line[x] = '·'
		}
		rows[y] = string(line)
	}
	for _, u := range a.Units {
		if !u.Alive || int(u.Y) >= a.H || int(u.X) >= a.W {
			continue
		}
		r := []rune(rows[u.Y])
		if u.Faction == 1 {
			r[u.X] = 'R'
		} else {
			r[u.X] = 'B'
		}
		rows[u.Y] = string(r)
	}
	return rows
}

func apiFigures(w http.ResponseWriter, _ *http.Request) {
	cat := tangram.Catalog()
	names := make([]string, 0, len(cat))
	for n := range cat {
		names = append(names, n)
	}
	sort.Strings(names)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(names)
}

type figResp struct {
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Gate    string   `json:"gate"`
	Rows    []string `json:"rows"`
}

func apiFigure(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	f, ok := tangram.Named(name)
	if !ok {
		http.Error(w, "unknown figure", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(figResp{Name: name, Address: f.Address(), Gate: f.GateCode(6), Rows: f.GlyphRows()})
}

// ---- live animated arena: the server holds one battle and steps it per request ----

var (
	arMu   sync.Mutex
	arGame *arena.Arena
	arTick int
)

func buildBattle() *arena.Arena {
	a := &arena.Arena{W: 20, H: 12, Terrain: make([]uint8, 20*12)}
	cat := tangram.Catalog()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	roster := []string{"cat", "fox", "owl", "crab", "turtle", "swan"}
	add := func(name string, f, x, y uint8) {
		if fig, ok := cat[name]; ok {
			if u, _, ok2 := arena.Summon(fig, f, x, y); ok2 {
				a.Units = append(a.Units, u)
			}
		}
	}
	for i := 0; i < 5; i++ {
		add(roster[r.Intn(len(roster))], 0, 2, uint8(1+i*2))
		add(roster[r.Intn(len(roster))], 1, 17, uint8(1+i*2))
	}
	return a
}

type arenaFrame struct {
	Tick   int   `json:"tick"`
	W      int   `json:"w"`
	H      int   `json:"h"`
	Cells  []int `json:"cells"` // row-major, 0 empty / 1 faction0 / 2 faction1
	Alive0 int   `json:"alive0"`
	Alive1 int   `json:"alive1"`
	Done   bool  `json:"done"`
}

// apiArena returns the next animation frame: it resets on ?reset=1 (or first
// call), otherwise advances the held battle by one tick.
func apiArena(w http.ResponseWriter, r *http.Request) {
	arMu.Lock()
	defer arMu.Unlock()
	if r.URL.Query().Get("reset") == "1" || arGame == nil {
		arGame = buildBattle()
		arTick = 0
	} else if arGame.AliveCount(0) > 0 && arGame.AliveCount(1) > 0 && arTick < 300 {
		arGame.Step(arena.RefCommander{}, arena.RefCommander{})
		arTick++
	}
	cells := make([]int, arGame.W*arGame.H)
	for _, u := range arGame.Units {
		if u.Alive && int(u.X) < arGame.W && int(u.Y) < arGame.H {
			cells[int(u.Y)*arGame.W+int(u.X)] = int(u.Faction) + 1
		}
	}
	a0, a1 := arGame.AliveCount(0), arGame.AliveCount(1)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(arenaFrame{
		Tick: arTick, W: arGame.W, H: arGame.H, Cells: cells,
		Alive0: a0, Alive1: a1, Done: a0 == 0 || a1 == 0 || arTick >= 300,
	})
}

// ---- live gallery: each capability rendered to a PNG on demand ----

func qstr(r *http.Request, key, def string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return def
}

func writePNG(w http.ResponseWriter, img image.Image) {
	w.Header().Set("Content-Type", "image/png")
	_ = png.Encode(w, img)
}

func imgFigure(w http.ResponseWriter, r *http.Request) {
	f, ok := tangram.Named(qstr(r, "name", "cat"))
	if !ok {
		http.Error(w, "unknown figure", http.StatusNotFound)
		return
	}
	writePNG(w, f.Image(360, 360))
}

func imgRelief(w http.ResponseWriter, r *http.Request) {
	f, ok := tangram.Named(qstr(r, "name", "cat"))
	if !ok {
		http.Error(w, "unknown figure", http.StatusNotFound)
		return
	}
	writePNG(w, relief.Render(f, relief.Gradient(f, 70, 150), 360, 360))
}

func imgTangram7(w http.ResponseWriter, r *http.Request) {
	var img image.Image
	switch qstr(r, "fig", "square") {
	case "tray":
		img = tangram7.Render(tangram7.Tray(), 720, 340)
	case "diamond":
		img = tangram7.Render(tangram7.Place(tangram7.Square(), 1, false, 0, 0), 420, 420)
	default:
		img = tangram7.Render(tangram7.Square(), 420, 420)
	}
	writePNG(w, img)
}

func imgGlyphQR(w http.ResponseWriter, r *http.Request) {
	grid, err := glyphqr.EncodeText(qstr(r, "msg", "TVCP-AI gate address: CAT"), 14)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writePNG(w, glyphqr.Render(grid, 16))
}

func imgBlazon(w http.ResponseWriter, r *http.Request) {
	sp := blazon.Parse(qstr(r, "arms", "Azure, a bend Or"))
	img, err := sp.Image(440, 240)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writePNG(w, img)
}

// ---- a living pseudo-3D world: a controller folds/rises/spins it and orbits the
// camera; we hold the evolving 240-frame sequence and serve any frame as a PNG ----

var (
	worldOnce       sync.Once
	worldFrames     []world.State
	liveMu          sync.Mutex
	liveWorldFrames []world.State
)

// generateLiveWorld asks a live brain (game:world) to fold/spin/walk an 18-frame
// world; it runs once at startup when -worldbrain is set (bounded API calls).
func generateLiveWorld(url string) {
	b := brain.HTTPBrain{URL: url, HTTP: &http.Client{Timeout: 60 * time.Second}}
	frames, _ := world.LoopC(&world.BrainController{B: b}, world.Init(), 18)
	liveMu.Lock()
	liveWorldFrames = frames
	liveMu.Unlock()
}

func imgWorld(w http.ResponseWriter, r *http.Request) {
	var frames []world.State
	if r.URL.Query().Get("live") == "1" {
		liveMu.Lock()
		frames = liveWorldFrames
		liveMu.Unlock()
	}
	if len(frames) == 0 {
		worldOnce.Do(func() {
			worldFrames, _ = world.LoopC(world.RefController{}, world.Init(), 240)
		})
		frames = worldFrames
	}
	if len(frames) == 0 {
		http.Error(w, "no world", http.StatusInternalServerError)
		return
	}
	f := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("f")); err == nil {
		f = v
	}
	n := len(frames)
	writePNG(w, frames[((f%n)+n)%n].Render(380, 260))
}

// ansiColor strips SGR colour codes so the block render shows cleanly in a <pre>.
var ansiColor = regexp.MustCompile("\x1b\\[[0-9;]*m")

// apiWorldBlocks renders a world frame with the ORIGINAL image-to-blocks codec
// (internal/codec/babe -> terminal.Frame quadrant glyphs) and serves it as text:
// the project's first capability applied to its newest one, in the browser.
func apiWorldBlocks(w http.ResponseWriter, r *http.Request) {
	worldOnce.Do(func() {
		worldFrames, _ = world.LoopC(world.RefController{}, world.Init(), 240)
	})
	if len(worldFrames) == 0 {
		http.Error(w, "no world", http.StatusInternalServerError)
		return
	}
	f := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("f")); err == nil {
		f = v
	}
	n := len(worldFrames)
	fr := babe.ImageToFrame(worldFrames[((f%n)+n)%n].Render(120, 72), 60, 30)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(ansiColor.ReplaceAllString(fr.Render(), "")))
}

// ---- a playable browser game: you summon tangram-creatures, the AI commands the
// other side, the arena fights it out. Combines arena + summon + commander. ----

var (
	gameMu     sync.Mutex
	gameA      *arena.Arena
	gameBudget int
	gameStep   int
)

type gameFrame struct {
	Tick   int   `json:"tick"`
	W      int   `json:"w"`
	H      int   `json:"h"`
	Cells  []int `json:"cells"`
	Alive0 int   `json:"alive0"`
	Alive1 int   `json:"alive1"`
	Budget int   `json:"budget"`
	Over   bool  `json:"over"`
	Winner int   `json:"winner"`
}

func gameNewLocked() {
	a := &arena.Arena{W: 18, H: 11, Terrain: make([]uint8, 18*11)}
	cat := tangram.Catalog()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	roster := []string{"cat", "fox", "owl", "crab", "turtle", "swan"}
	for i := 0; i < 5; i++ {
		if fig, ok := cat[roster[r.Intn(len(roster))]]; ok {
			if u, _, ok2 := arena.Summon(fig, 1, 15, uint8(1+i*2)); ok2 {
				a.Units = append(a.Units, u)
			}
		}
	}
	gameA, gameBudget, gameStep = a, 4, 0
}

func gameFreeSlot() uint8 {
	used := map[uint8]bool{}
	for _, u := range gameA.Units {
		if u.Alive && u.Faction == 0 && u.X <= 3 {
			used[u.Y] = true
		}
	}
	for y := uint8(1); y < uint8(gameA.H-1); y++ {
		if !used[y] {
			return y
		}
	}
	return uint8(gameA.H / 2)
}

func gameSnapshot() gameFrame {
	cells := make([]int, gameA.W*gameA.H)
	for _, u := range gameA.Units {
		if u.Alive && int(u.X) < gameA.W && int(u.Y) < gameA.H {
			cells[int(u.Y)*gameA.W+int(u.X)] = int(u.Faction) + 1
		}
	}
	a0, a1 := gameA.AliveCount(0), gameA.AliveCount(1)
	over, win := false, -1
	switch {
	case a1 == 0 && a0 > 0:
		over, win = true, 0
	case a0 == 0 && gameStep > 0:
		over, win = true, 1
	case gameStep >= 300:
		over = true
		if a0 > a1 {
			win = 0
		} else if a1 > a0 {
			win = 1
		}
	}
	return gameFrame{Tick: gameStep, W: gameA.W, H: gameA.H, Cells: cells, Alive0: a0, Alive1: a1, Budget: gameBudget, Over: over, Winner: win}
}

func apiGame(w http.ResponseWriter, r *http.Request) {
	gameMu.Lock()
	defer gameMu.Unlock()
	if r.URL.Query().Get("new") == "1" || gameA == nil {
		gameNewLocked()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(gameSnapshot())
}

func apiGameSummon(w http.ResponseWriter, r *http.Request) {
	gameMu.Lock()
	defer gameMu.Unlock()
	if gameA == nil {
		gameNewLocked()
	}
	if s := gameSnapshot(); !s.Over && gameBudget > 0 {
		name := r.URL.Query().Get("fig")
		if name == "" {
			name = "cat"
		}
		if fig, ok := tangram.Catalog()[name]; ok {
			if u, _, ok2 := arena.Summon(fig, 0, 2, gameFreeSlot()); ok2 {
				gameA.Units = append(gameA.Units, u)
				gameBudget--
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(gameSnapshot())
}

func apiGameStep(w http.ResponseWriter, _ *http.Request) {
	gameMu.Lock()
	defer gameMu.Unlock()
	if gameA == nil {
		gameNewLocked()
	}
	if s := gameSnapshot(); !s.Over {
		gameA.Step(arena.RefCommander{}, arena.RefCommander{})
		gameStep++
		if gameStep%2 == 0 && gameBudget < 6 {
			gameBudget++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(gameSnapshot())
}

// ---- an INTERACTIVE walk: you steer the camera and morph a living surreal
// world with directives; the server holds the state and renders each step. ----

var (
	walkMu    sync.Mutex
	walkState world.State
	walkInit  bool
)

func qdir(r *http.Request, key string) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return 0
	}
	if v < -1 {
		return -1
	}
	if v > 1 {
		return 1
	}
	return v
}

func apiWalk(w http.ResponseWriter, r *http.Request) {
	walkMu.Lock()
	defer walkMu.Unlock()
	q := r.URL.Query()
	if q.Get("reset") == "1" || !walkInit {
		walkState = world.Init()
		walkInit = true
	}
	var d world.Directives
	if q.Get("morph") == "1" {
		d = world.RefController{}.Decide(walkState) // keep it evolving
	}
	if q.Get("cam") != "" {
		d.Camera = qdir(r, "cam")
	}
	if q.Get("fold") != "" {
		d.Fold = qdir(r, "fold")
	}
	if q.Get("rise") != "" {
		d.Rise = qdir(r, "rise")
	}
	if q.Get("spin") != "" {
		d.Spin = qdir(r, "spin")
	}
	walkState = world.Apply(walkState, d)
	writePNG(w, walkState.Render(380, 260))
}

func index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

const page = `<!doctype html><html lang="ru"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TVCP benchserver</title>
<style>
 body{margin:0;background:#0e141c;color:#cfe9d8;font-family:ui-monospace,Menlo,Consolas,monospace;font-size:14px}
 .wrap{max-width:1000px;margin:0 auto;padding:18px}
 h1{font-size:17px;letter-spacing:.04em;margin:0 0 2px} .sub{color:#7f93a8;font-size:12px}
 button{font:inherit;background:#16202c;color:#cfe9d8;border:1px solid #25323f;border-radius:7px;padding:7px 12px;cursor:pointer;margin:0 6px 6px 0}
 button:hover{border-color:#6ad08f;color:#6ad08f}
 .panel{background:#11161d;border:1px solid #1e2731;border-radius:10px;padding:14px;margin:12px 0}
 pre{white-space:pre;overflow:auto;line-height:1.12;font-size:16px;color:#9be8b5;margin:0}
 table{width:100%;border-collapse:collapse} td{padding:4px 8px;border-bottom:1px solid #1a232d;vertical-align:top}
 .ok{color:#6ad08f}.bad{color:#ef6a6a}.g{color:#5ec0e6;text-transform:uppercase;font-size:11px;letter-spacing:.09em}
 .dim{color:#7f93a8}.id{color:#e6b94e;margin:6px 0}
 h2{font-size:12px;color:#7f93a8;text-transform:uppercase;letter-spacing:.08em;margin:0 0 8px}
</style></head><body><div class="wrap">
 <h1>TVCP · benchserver</h1>
 <div class="sub">живой стенд — этот сервер реально выполняет возможности infon и отдаёт их сюда</div>

 <div class="panel">
   <h2>🎮 игра: арена — ты против ИИ</h2>
   <div class="dim" style="margin-bottom:6px">призывай юниты (синие), ИИ командует красными. Победа — выбить всех. «авто» гоняет бой сам.</div>
   <div id="gamebtns"></div>
   <div id="gamestat" class="dim" style="margin-top:6px">—</div>
   <canvas id="gamecv" width="468" height="286" style="display:block;margin-top:8px;background:#0b1118;border-radius:8px;width:100%;max-width:600px"></canvas>
 </div>

 <div class="panel">
   <h2>тест-стенд</h2>
   <button onclick="runBench()">▶ запустить testbench</button>
   <span id="verdict" class="dim">—</span>
   <table id="bench"></table>
 </div>

 <div class="panel">
   <h2>живой рендер фигур (глифы)</h2>
   <div id="figbtns"></div>
   <div id="figid" class="id"></div>
   <pre id="figart">выбери фигуру…</pre>
 </div>

 <div class="panel">
   <h2>республика — discover ▸ negotiate ▸ fight</h2>
   <button onclick="runRepublic()">▶ собрать республику</button>
   <span id="repverdict" class="dim">—</span>
   <div id="repseats" style="margin:8px 0"></div>
   <pre id="repfield"></pre>
 </div>

 <div class="panel">
   <h2>арена — живой бой (анимация)</h2>
   <button onclick="arenaReset()">⟳ новый бой</button>
   <span id="arstat" class="dim">—</span>
   <canvas id="arcanvas" width="420" height="252" style="display:block;margin-top:8px;background:#0b1118;border-radius:8px;width:100%;max-width:560px"></canvas>
 </div>

 <div class="panel">
   <h2>галерея демо — серверный рендер (PNG на лету)</h2>
   <div id="galbtns"></div>
   <img id="galimg" alt="render" style="display:block;margin-top:8px;max-width:100%;border-radius:8px;background:#0b1118">
 </div>

 <div class="panel">
   <h2>прогулка по живому 2.5D-миру</h2>
   <div class="dim" style="margin-bottom:6px">контроллер ведёт мир: fold · rise · spin · камера — 51 байт состояния на кадр</div>
   <button id="wlbtn" onclick="toggleWorldLive()">○ живой мозг</button>
   <span class="dim" style="font-size:10.5px">запусти scripts/worldwalk-live.ps1 (или benchserver -worldbrain URL)</span>
   <img id="worldimg" alt="world" style="display:block;margin-top:8px;max-width:100%;border-radius:8px;background:#0b1118">
   <div class="dim" style="margin-top:8px;font-size:10.5px">первая возможность: тот же мир оригинальным кодеком «картинка → блоки» (babe)</div>
   <pre id="wblocks" style="font-size:8px;line-height:1.05;color:#9be8b5;background:#0b1118;border-radius:6px;padding:6px;overflow:auto;margin:4px 0 0"></pre>
 </div>

 <div class="panel">
   <h2>🚶 прогулка по сюрреалистическому миру</h2>
   <div class="dim" style="margin-bottom:6px">веди сам: стрелки ← → — камера, ↑ ↓ — рельеф; кнопки — склад/вращение; «морф» — мир сам эволюционирует</div>
   <div id="walkbtns"></div>
   <img id="walkimg" alt="walk" style="display:block;margin-top:8px;max-width:100%;border-radius:8px;background:#0b1118">
 </div>

<script>
async function runBench(){
  document.getElementById('verdict').textContent='…';
  const d=await (await fetch('/api/bench')).json();
  const g={}; d.checks.forEach(c=>{(g[c.group]=g[c.group]||[]).push(c);});
  let h='';
  for(const grp in g){
    h+='<tr><td class="g" colspan="4">'+grp+'</td></tr>';
    g[grp].forEach(c=>{ h+='<tr><td class="'+(c.pass?'ok':'bad')+'">'+(c.pass?'✓':'✗')+'</td><td>'+c.name+'</td><td>'+c.metric+'</td><td class="dim">'+((c.detail||'')+' · '+c.ms+'ms')+'</td></tr>'; });
  }
  document.getElementById('bench').innerHTML=h;
  const v=document.getElementById('verdict');
  v.className=(d.passed==d.total)?'ok':'bad';
  v.textContent=d.passed+'/'+d.total+' PASS · '+d.protocol+' · '+d.when;
}
async function loadFigs(){
  const names=await (await fetch('/api/figures')).json();
  document.getElementById('figbtns').innerHTML=names.map(n=>'<button onclick="showFig(\''+n+'\')">'+n+'</button>').join('');
}
async function showFig(n){
  const d=await (await fetch('/api/figure?name='+encodeURIComponent(n))).json();
  document.getElementById('figid').textContent=d.name+'  ·  address '+d.address+'  ·  gate '+d.gate;
  document.getElementById('figart').textContent=d.rows.join('\n');
}
async function runRepublic(){
  document.getElementById('repverdict').textContent='…';
  const d=await (await fetch('/api/republic')).json();
  const seats=(d.seats||[]).map(x=>'faction '+x.faction+' ⇐ <span class="ok">'+x.brain+'</span> (bid '+x.cost+')').join('<br>');
  document.getElementById('repseats').innerHTML='<span class="dim">registered: '+(d.registered||[]).join(', ')+'</span><br>'+seats;
  const win=d.winner===0?'faction 0 holds':d.winner===1?'faction 1 holds':'draw';
  const v=document.getElementById('repverdict'); v.className='ok';
  v.textContent=win+' · '+d.ticks+' ticks · '+d.alive0+' vs '+d.alive1;
  document.getElementById('repfield').textContent=(d.field||[]).join('\n');
}
let arTimer=null;
function drawArena(d){
  const cv=document.getElementById('arcanvas'); if(!cv) return;
  const ctx=cv.getContext('2d'), W=cv.width, H=cv.height; ctx.clearRect(0,0,W,H);
  const cw=W/d.w, ch=H/d.h;
  for(let y=0;y<d.h;y++) for(let x=0;x<d.w;x++){
    const v=d.cells[y*d.w+x];
    if(v===0){ ctx.fillStyle='rgba(120,150,190,.10)'; ctx.fillRect(x*cw+cw*0.42,y*ch+ch*0.42,cw*0.16,ch*0.16); }
    else { ctx.fillStyle=(v===1)?'#4a9bff':'#ff5a5a'; ctx.fillRect(x*cw+1,y*ch+1,cw-2,ch-2); }
  }
}
async function arenaStep(reset){
  let d; try { d=await (await fetch('/api/arena'+(reset?'?reset=1':''))).json(); } catch(e){ return; }
  drawArena(d);
  const st=document.getElementById('arstat');
  const verdict=d.done?(d.alive0>d.alive1?' · blue holds':d.alive1>d.alive0?' · red holds':' · draw'):'';
  st.innerHTML='tick '+d.tick+' · <span style="color:#4a9bff">blue '+d.alive0+'</span> vs <span style="color:#ff5a5a">red '+d.alive1+'</span>'+verdict;
  clearTimeout(arTimer);
  arTimer=setTimeout(()=>arenaStep(d.done), d.done?2000:170);
}
function arenaReset(){ clearTimeout(arTimer); arenaStep(true); }
const galleryItems=[
  ['фигура (растр)','/img/figure?name=cat'],
  ['рельеф 2.5D','/img/relief?name=fox'],
  ['танграм-7','/img/tangram7?fig=square'],
  ['glyph-QR','/img/glyphqr?msg='+encodeURIComponent('TVCP-AI gate address: CAT')],
  ['герб (blazon)','/img/blazon?arms='+encodeURIComponent('Azure, a bend Or')],
];
function loadGallery(){
  document.getElementById('galbtns').innerHTML=galleryItems.map((it,i)=>'<button onclick="showImg('+i+')">'+it[0]+'</button>').join('');
}
function showImg(i){ document.getElementById('galimg').src=galleryItems[i][1]+'&_t='+Date.now(); }
window.worldLive=false;
function toggleWorldLive(){ window.worldLive=!window.worldLive; document.getElementById('wlbtn').textContent=window.worldLive?'● живой мозг: ВКЛ':'○ живой мозг'; }
(function(){ const wimg=document.getElementById('worldimg'); let wf=0;
  function nextSrc(){ return '/img/world?f='+wf+(window.worldLive?'&live=1':''); }
  wimg.onload=function(){ setTimeout(function(){ wf++; wimg.src=nextSrc(); }, 90); };
  wimg.onerror=function(){ setTimeout(function(){ wimg.src=nextSrc(); }, 400); };
  wimg.src=nextSrc();
})();
(function(){ const wb=document.getElementById('wblocks'); if(!wb)return; let bf=0;
  function step(){ fetch('/api/worldblocks?f='+bf).then(function(r){return r.text();}).then(function(t){ wb.textContent=t; }).catch(function(){}); bf++; setTimeout(step, 220); }
  step();
})();
let gameAuto=null;
function drawGame(d){
  const cv=document.getElementById('gamecv'); if(!cv)return; const ctx=cv.getContext('2d'),W=cv.width,H=cv.height; ctx.clearRect(0,0,W,H);
  const cw=W/d.w, ch=H/d.h;
  for(let y=0;y<d.h;y++) for(let x=0;x<d.w;x++){ const v=d.cells[y*d.w+x];
    if(v===0){ ctx.fillStyle='rgba(120,150,190,.10)'; ctx.fillRect(x*cw+cw*0.42,y*ch+ch*0.42,cw*0.16,ch*0.16); }
    else { ctx.fillStyle=(v===1)?'#4a9bff':'#ff5a5a'; ctx.fillRect(x*cw+1,y*ch+1,cw-2,ch-2); } }
}
function gameStatTxt(d){
  const v=d.over?(d.winner===0?' · 🏆 ты победил!':d.winner===1?' · ☠ ИИ победил':' · ничья'):'';
  document.getElementById('gamestat').innerHTML='бюджет призыва '+d.budget+' · <span style="color:#4a9bff">ты '+d.alive0+'</span> vs <span style="color:#ff5a5a">ИИ '+d.alive1+'</span> · ход '+d.tick+v;
}
async function gameFetch(u){ try{ const d=await (await fetch(u)).json(); drawGame(d); gameStatTxt(d); if(d.over&&gameAuto){clearInterval(gameAuto);gameAuto=null;} return d; }catch(e){} }
function gameSummon(f){ gameFetch('/api/game/summon?fig='+f); }
function gameStepOnce(){ gameFetch('/api/game/step'); }
function gameNew(){ if(gameAuto){clearInterval(gameAuto);gameAuto=null;} gameFetch('/api/game?new=1'); }
function gameAutoToggle(){ if(gameAuto){clearInterval(gameAuto);gameAuto=null;} else { gameAuto=setInterval(gameStepOnce,450); } }
function loadGame(){ const figs=['cat','fox','owl','crab','turtle','swan'];
  document.getElementById('gamebtns').innerHTML=figs.map(f=>'<button onclick="gameSummon(\''+f+'\')">+ '+f+'</button>').join('')+' <button onclick="gameStepOnce()">⚔ ход</button><button onclick="gameAutoToggle()">▶ авто</button><button onclick="gameNew()">🔄 новая</button>'; }
let walkMorph=null;
function walkGo(p){ const i=document.getElementById('walkimg'); if(i) i.src='/api/walk?'+p+'&_t='+Date.now(); }
function walkReset(){ if(walkMorph){clearInterval(walkMorph);walkMorph=null;} walkGo('reset=1'); }
function walkMorphToggle(){ if(walkMorph){clearInterval(walkMorph);walkMorph=null;} else { walkMorph=setInterval(function(){walkGo('morph=1');},260); } }
function loadWalk(){ document.getElementById('walkbtns').innerHTML=
  '<button onclick="walkGo(\'cam=-1\')">◄ камера</button><button onclick="walkGo(\'cam=1\')">камера ►</button>'+
  '<button onclick="walkGo(\'fold=1\')">склад +</button><button onclick="walkGo(\'fold=-1\')">склад −</button>'+
  '<button onclick="walkGo(\'spin=1\')">↻</button><button onclick="walkGo(\'spin=-1\')">↺</button>'+
  '<button onclick="walkMorphToggle()">↻ морф</button><button onclick="walkReset()">🌀 новый мир</button>'; }
document.addEventListener('keydown',function(ev){ const k=ev.key;
  if(k==='ArrowLeft'){walkGo('cam=-1');ev.preventDefault();}
  else if(k==='ArrowRight'){walkGo('cam=1');ev.preventDefault();}
  else if(k==='ArrowUp'){walkGo('rise=1');ev.preventDefault();}
  else if(k==='ArrowDown'){walkGo('rise=-1');ev.preventDefault();} });
loadGame(); gameFetch('/api/game?new=1');
loadWalk(); walkGo('reset=1');
loadFigs(); runBench(); runRepublic(); arenaStep(true); loadGallery(); showImg(0);
</script>
</div></body></html>`
