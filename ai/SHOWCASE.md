# TVCP-AI — showcase

Connecting **any neural network** to terminal graphics via the open
`tvcp-ai/1` format. Everything below was compiled with Go 1.24 on the PC and
run for real; previews are faithful renders of the terminal output.

> Images are hosted on the [`assets`](https://github.com/svend4/infon/tree/assets)
> orphan branch (kept out of the code history); they render inline below.

## 1. The rendering core (draw-DSL → terminal)
![sunset](https://raw.githubusercontent.com/svend4/infon/assets/previews/sunset_real.png)
Real `aidraw.exe` renders a JSON draw-DSL scene I authored. `aidraw scenes/sunset.json`

## 2. Pixel path (any image → blocks)
![aurora](https://raw.githubusercontent.com/svend4/infon/assets/previews/aurora_terminal.png)
`babe.ImageToFrame` turns a raster into 2×2 quadrant blocks. `aiimg assets/aurora.png 80 50`

## 3. AI plays a game
![tictactoe](https://raw.githubusercontent.com/svend4/infon/assets/previews/ttt.gif)
The repo's tic-tac-toe engine + minimax, rendered as colored frames. `aiplay`

## 4. A visual language (cards + effects)
![cards](https://raw.githubusercontent.com/svend4/infon/assets/previews/cards.png)
Glyph "cards" + a truecolor sweep. `aicards`

## 5. AI as a video source
![brain video](https://raw.githubusercontent.com/svend4/infon/assets/previews/brain_video.gif)
A model paints frames (sketch → scene). `tvcp ai -brain URL` / `aicam -brain URL`

## 6. Open protocol — any model plugs in
![brain game](https://raw.githubusercontent.com/svend4/infon/assets/previews/brain_game.gif)
A full game over real HTTP using `tvcp-ai/1`. `braindemo -brain URL`

## 7. Live with Claude Haiku (your API key)
![vs haiku](https://raw.githubusercontent.com/svend4/infon/assets/previews/vs_haiku_captioned.gif)
You beat Haiku in the terminal (Haiku plays O). `aiturn -brain URL <r> <c>`

![haiku sketch](https://raw.githubusercontent.com/svend4/infon/assets/previews/haiku_sketch.png)
Haiku draws via the simple `sketch` format. `aidraw -brain URL -sketch -prompt "..."`

![haiku react](https://raw.githubusercontent.com/svend4/infon/assets/previews/haiku_react2.png)
Haiku reacts with symbol cards. `aicards -brain URL -msg "..."`

## 8. A new game in the same protocol — Wordle
![wordle](https://raw.githubusercontent.com/svend4/infon/assets/previews/wordle_ref.gif)
The brain guesses; the repo's Wordle engine gives feedback. `aiwordle -word GHOST`

## Binaries (infon/bin)
| binary | what | run |
|---|---|---|
| aidraw | draw-DSL / brain draw / sketch | `aidraw scenes/sunset.json` |
| aiimg | image → terminal | `aiimg assets/aurora.png 80 50` |
| aiplay / aiturn | tic-tac-toe (vs minimax / vs brain) | `aiturn -brain URL 1 1` |
| aicards | cards + effect / brain react | `aicards -brain URL -msg "..."` |
| aicam | AI video (plasma / brain) | `aicam -brain URL` |
| aiwordle | Wordle (brain guesses) | `aiwordle -brain URL -word WATER` |
| braindemo | full game + draw over HTTP | `braindemo -brain URL` |
| brainserver | reference tvcp-ai/1 server | `brainserver 127.0.0.1:8088` |
| tvcp | full platform; `tvcp ai [-brain URL]` | `tvcp help` |

## Plug in a model
`adapters/anthropic_brain.py` (Haiku), `ollama_brain.py`, `openai_brain.py` —
all speak `tvcp-ai/1`. See `BRAIN_PROTOCOL.md`. Swap brains by changing the URL.
