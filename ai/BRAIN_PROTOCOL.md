# tvcp-ai/1 — an open "brain" protocol

Connect ANY neural network as a partner (game play, visual messages) behind one
small JSON-over-HTTP contract. Swap brains by changing a URL: local Ollama,
cloud OpenAI/Anthropic, an OpenAI-compatible server (llama.cpp, vLLM), or this
model. No engine lock-in.

## Transport
`POST <endpoint>`  — reference server: `http://127.0.0.1:8088/v1/decide`
`Content-Type: application/json`. Body = Request; reply = Response.

## Request
```json
{ "protocol":"tvcp-ai/1", "kind":"move|draw|react",
  "game":"tictactoe",
  "state": { ... },
  "prompt":"a calm horizon at dusk",
  "canvas": { "width":64, "height":24 } }
```

## Response
```json
{ "protocol":"tvcp-ai/1", "kind":"move|draw|react",
  "move":  { "row":1, "col":1 },
  "scene": { "width":64, "height":20, "ops":[ ... ] },
  "cards": ["★","♥","✓"],
  "reasoning":"...", "error":"" }
```

## Game state — `tictactoe`
```json
"state": { "board":[["","",""],["","X",""],["","O",""]],
           "turn":"X", "you":"X", "legal":[[0,0],[0,1]] }
```
Reply with `move`. Trivially extended to other games (define a `state` shape).

## Draw
Reply `scene` is the draw-DSL (see README): a canvas plus ops
`fill / rect / vgradient / hgradient / text / box / quad / disc`.

## Implement a brain (adapter)
Produce the Response shape — that's it. Examples:
- `bin\brainserver.exe`              reference (minimax + scene), no model
- `adapters/ollama_brain.py`         local model via Ollama
- `adapters/openai_brain.py`         any OpenAI-compatible endpoint (OpenAI,
                                     Anthropic-compatible gateways, llama.cpp, vLLM)

## Swap brains
Clients take an endpoint URL (env `BRAIN_URL`). Point it at any of the above;
the game/draw clients (`braindemo`, future `aiplay --brain`) are unchanged.

## Verified
`bin\braindemo.exe` runs a full game (X = brain over HTTP) and a draw over real
HTTP using this format. See
[brain_game.gif](https://raw.githubusercontent.com/svend4/infon/assets/previews/brain_game.gif)
and [brain_draw.png](https://raw.githubusercontent.com/svend4/infon/assets/previews/brain_draw.png).

## Update: sketch, react, robustness (points 1-3)

### kind = sketch  (easy for small models)
A high-level alternative to `draw`: named colors + a few shapes. The client
translates it to the full draw-DSL (`pkg/sketch`).
```json
"sketch": { "sky":"orange", "ground":"navy", "shapes":[
   {"kind":"sun","x":48,"y":6,"w":3,"color":"gold"},
   {"kind":"mountain","x":16,"h":8,"color":"darkgreen"},
   {"kind":"text","x":2,"y":1,"s":"a calm harbor","color":"white"} ] }
```
shape kinds: sun moon star hill mountain building cloud band text.
colors (names): white black gray red orange yellow gold amber green darkgreen
teal blue navy cyan skyblue purple pink brown dusk.

### kind = react
`{ "cards": ["star","heart","check"] }` — names map to glyph cards (★ ♥ ✓ …).

### Robustness (point 1)
- Adapters retry up to 3x, validate the move is legal, validate a scene has ops
  / a sketch has shapes; safe fallbacks otherwise.
- Clients enforce the requested canvas size (a model can't blow up the frame);
  `scene.RenderScene` also hard-clamps to 400x200.

### Client flags
- `aidraw -brain URL -prompt "..." [-sketch] [-canvas WxH]`
- `aiturn -brain URL <row> <col>`   # O is played by the brain (play vs Haiku)
- `aicards -brain URL -msg "..."`   # react cards chosen by the brain
- `braindemo -brain URL -prompt "..."`  # full game + draw over the protocol
