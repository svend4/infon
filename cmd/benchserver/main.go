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
	"sort"
	"strconv"
	"sync"
	"time"

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
	flag.Parse()

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
	worldOnce   sync.Once
	worldFrames []world.State
)

func imgWorld(w http.ResponseWriter, r *http.Request) {
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
	writePNG(w, worldFrames[((f%n)+n)%n].Render(380, 260))
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
   <img id="worldimg" alt="world" style="display:block;max-width:100%;border-radius:8px;background:#0b1118">
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
(function(){ const wimg=document.getElementById('worldimg'); let wf=0;
  wimg.onload=function(){ setTimeout(function(){ wf++; wimg.src='/img/world?f='+wf; }, 90); };
  wimg.onerror=function(){ setTimeout(function(){ wimg.src='/img/world?f='+wf; }, 400); };
  wimg.src='/img/world?f=0';
})();
loadFigs(); runBench(); runRepublic(); arenaStep(true); loadGallery(); showImg(0);
</script>
</div></body></html>`
