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
	"log"
	"net/http"
	"sort"

	"github.com/svend4/infon/pkg/bench"
	"github.com/svend4/infon/pkg/tangram"
)

func main() {
	addr := flag.String("addr", ":8086", "listen address")
	flag.Parse()

	http.HandleFunc("/", index)
	http.HandleFunc("/api/bench", apiBench)
	http.HandleFunc("/api/figures", apiFigures)
	http.HandleFunc("/api/figure", apiFigure)

	log.Printf("TVCP benchserver — open http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func apiBench(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(bench.Run())
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
loadFigs(); runBench();
</script>
</div></body></html>`
