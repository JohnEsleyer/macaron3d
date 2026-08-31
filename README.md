# Macaron

> **Micro-apps for 3D — do one thing and do it well.**

Macaron is a monorepo of tiny, single-purpose 3D tools that share one fast core. Think `vscode .` for 3D: `macaron-model .`, `macaron-sculpt .`, `macaron-uv .` — same project, right ergonomics for the task at hand.

No Swiss-army bloat. No 5,000-button startup. Just custom woodcarving knives instead of an intimidating multi-tool.

**Stack:** Go · [raylib-go](https://github.com/gen2brain/raylib-go) · [cimgui-go](https://github.com/AllenDang/cimgui-go) · OBJ/glTF · Immediate-mode UI

---

## Why this exists

### Blender fatigue is real

Monolithic DCCs (Blender, Maya, Max) must serve archviz, VFX, scripting, simulation, video editing — all at once. Result: infinite scope, thousands of context-dependent controls, steep cognitive wall. Most artists use 5–10% of it for their style.

### When a tool does one thing obsessively well, it wins

| Tool | Beat | By doing |
|------|------|----------|
| **Aseprite** | Photoshop/GIMP for pixel art | Frame + palette workflow only |
| **Blockbench** | General modelers for low-poly/Minecraft | Box modeling + UV + instant export |
| **MagicaVoxel** | Full 3D suites for voxels | Voxels, ultra-clean UI |
| **Marmoset Toolbag** | Bakers/renderers | Fast, beautiful bake & preview |

Macaron applies that same **Unix philosophy** to 3D.

---

## Principles

1. **One task, perfect ergonomics** — each app exposes only the controls that matter.
2. **Zero-lag startup** — <150ms, no splash screen, zen canvas until you summon UI.
3. **Deterministic muscle memory** — `G/R/S`, `Tab`, `Numpad 1/3/7`, `Z` shading behave identically everywhere.
4. **Lossless open exchange** — `mesh.obj` / `mesh.gltf` + `textures/*.png` is the source of truth. No proprietary lock-in. Godot, Unreal, web viewers read the folder directly.
5. **One core, many apps** — `pkg/` is the SDK. Fix a normal or OBJ bug once, every tool gets it.

---

## Quick start

### Prerequisites

- Go 1.22+
- C toolchain (`gcc`/`clang`), required by raylib + cimgui-go
- Linux: `libgl1-mesa-dev libwayland-dev libxkbcommon-dev libx11-dev` (raylib deps)

### Install

```bash
git clone <this-repo> macaron
cd macaron

# builds every micro-tool to $GOPATH/bin (or $HOME/go/bin)
go install ./cmd/...

# or run without installing
go run ./cmd/macaron --help
```

### Create and open a project

```bash
# 1. scaffold
macaron init knight-shield
cd knight-shield
# knight-shield/
# ├── macaron.json
# ├── mesh.obj          ← canonical geometry
# ├── textures/
# ├── references/
# └── .macaron/         ← cache, gitignored

# 2. open with any tool — like `code .`
macaron-model .     # low-poly blockout
macaron-sculpt .    # digital clay
macaron-uv .        # palette UV
macaron-pixel .     # 3D → 2D sprite sheet

# 3. pipeline (file-watch keeps tools in sync)
macaron-model . | macaron-uv --palette picos-8.png   # (planned)
```

---

## The project specification

`macaron init hero` creates a Git-friendly folder:

```
hero/
├── macaron.json          # { "name": "hero", "version": "0.1.0", "mesh": "mesh.obj" }
├── mesh.obj              # canonical geometry — commit this
├── mesh.gltf             # (optional) — same role, richer
├── textures/
│   └── palette.png
├── references/           # orthographic sheets
│   ├── front.png
│   └── side.png
└── .macaron/
    └── state.json        # window/camera cache — gitignored
```

`macaron.json` is a sidecar: modifier stacks, bone weights, palette swatches live there without corrupting the standard geometry. `.macaron/` is ephemeral and ignored.

---

## Repository architecture

### Monorepo > multi-repo > fork

**Never fork for tool variants** — fixing one OBJ or viewport bug would mean cherry-picking across N repos.

A Go monorepo gives you one `go.mod`, atomic updates to `pkg/`, and a new tool is just a folder in `cmd/`:

```bash
go install ./cmd/...   # builds macaron, macaron-model, macaron-sculpt, ...
```

```
macaron/
├── go.mod
├── go.sum
├── README.md
│
├── cmd/                          # independent CLI binaries
│   ├── macaron/                  # project manager: init, status, doctor
│   ├── macaron-model/            # low-poly blockout & kitbashing
│   ├── macaron-sculpt/           # pure sculpt + voxel remesh
│   ├── macaron-uv/               # palette / UV unwrap
│   ├── macaron-rig/              # bone & poser (stub)
│   └── macaron-pixel/            # 3D → 2D sprite exporter (stub)
│
└── pkg/                          # macaron-sdk — the shared core
    ├── project/                  # load/validate macaron.json
    ├── engine/                   # window loop, raylib backend, Tool lifecycle
    ├── camera/                   # orbit/pan/ortho + Front/Back/Left/Right/Top/Bottom/Iso snap
    ├── mesh/                     # Data, Object, primitives, raycast, normals
    ├── io/                       # OBJ (glTF/STL/PLY planned)
    ├── render/                   # grid, gizmo, checker, matcap, eevee shading
    ├── undo/                     # history stack (delta/command planned)
    └── ui/                       # shared theme, status bar, top menu
```

### What `pkg` guarantees

- `project` — load/validate/scaffold.
- `engine` — `Launch()` handles window, input, camera, 3D pass, ImGui, save-to-`mesh.obj`.
- `camera` — one orbit model, snap presets, `Numpad 5` ortho toggle.
- `mesh` — topology decoupled from rendering (headless/CLI-friendly).
- `io` — robust standard-format serializers.

---

## The SDK — how a micro-tool is built

### The `Tool` interface (`pkg/engine/tool.go`)

```go
package engine

type Tool interface {
    Name() string
    Description() string

    Init(ctx *Context) error
    Update(ctx *Context, dt float32)

    Draw3D(ctx *Context) // 3D pass: bones, brush rings, etc.
    DrawUI(ctx *Context) // ImGui panels (summonable from top bar)

    OnSave(ctx *Context) error
}
```

### The shared `Context` (`pkg/engine/context.go`)

```go
type Context struct {
    Project *project.Project
    Camera  *camera.Viewport
    Objects []mesh.Object   // active scene
    NextID  int
    SelID   int
    History undo.Stack

    RayHit      bool        // live mouse ray
    HitPos      rl.Vector3
    HitNormal   rl.Vector3
    HoveredFace int
}
```

Helpers: `ctx.ActiveObject()`, `ctx.AddObject(name, mesh.Data)`, `ctx.SetStatus(msg)`, `io.ExportOBJ(proj.MeshPath, ctx.Objects)`.

### New tool in 10 minutes

Create `cmd/macaron-paint/main.go`:

```go
package main

import (
    "os"
    "macaron/pkg/engine"
    "github.com/AllenDang/cimgui-go/imgui"
    rl "github.com/gen2brain/raylib-go/raylib"
)

type PaintTool struct {
    BrushColor [4]float32
    BrushSize  float32
}

func (t *PaintTool) Name() string        { return "Macaron Paint" }
func (t *PaintTool) Description() string { return "Pixel & palette painter" }

func (t *PaintTool) Init(_ *engine.Context) error {
    t.BrushColor = [4]float32{1, 0, 0, 1}
    t.BrushSize = 0.2
    return nil
}

func (t *PaintTool) Update(ctx *engine.Context, _ float32) {
    if rl.IsMouseButtonDown(rl.MouseButtonLeft) && ctx.HoveredFace != -1 {
        // paint logic…
    }
}

func (t *PaintTool) Draw3D(ctx *engine.Context) {
    if ctx.RayHit {
        rl.DrawSphereWires(ctx.HitPos, t.BrushSize, 12, 12, rl.Red)
    }
}

func (t *PaintTool) DrawUI(_ *engine.Context) {
    if imgui.BeginV("Paint Palette", nil, 0) {
        imgui.ColorEdit4("Color", &t.BrushColor)
        imgui.SliderFloat("Size", &t.BrushSize, 0.05, 1.0)
    }
    imgui.End()
}

func (t *PaintTool) OnSave(ctx *engine.Context) error {
    return nil // or ctx.SaveActiveMeshAndTextures()
}

func main() {
    dir := "."
    if len(os.Args) > 1 { dir = os.Args[1] }
    engine.Launch(dir, &PaintTool{})
}
```

```bash
go run ./cmd/macaron-paint --help   # or: go install ./cmd/macaron-paint && macaron-paint .
```

`engine.Launch` does the rest: parse `dir`, load `macaron.json` + `mesh.obj`, init raylib/imgui, zen viewport, shared hotkeys, menu/status.

---

## Inter-tool ergonomics (planned / in progress)

Monorepo alone isn't enough — the tools must feel like one pipeline.

| Mechanism | Idea | Status |
|-----------|------|--------|
| **System 3D clipboard** | `Ctrl+C` in model copies glTF buffer, `Ctrl+V` in sculpt pastes | planned |
| **Live file-watch** | `mesh.obj` modified on disk → hot-reload in <16ms, camera preserved | planned |
| **Micro-pipelines** | `macaron-model hero.obj \| macaron-uv --palette picos-8.png` | planned |

For now: all tools read/write the same `mesh.obj` — just re-open or `File → Save` to sync.

---

## CLI reference

```bash
macaron init <name>      # scaffold project (accepts path or bare name)
macaron status [path]    # show manifest + mesh presence
macaron doctor [path]    # validate project layout

macaron-model [path]     # blockout — default "." 
macaron-sculpt [path]    # sculpt
macaron-uv [path]        # uv / palette
macaron-rig [path]       # rig (stub)
macaron-pixel [path]     # pixel / sprite (stub)
```

All `macaron-*` tools accept a single optional argument: project directory (like `code .`). If you pass a file, its parent dir is used.

### Shared hotkeys (every tool)

- `G / R / S` — grab / rotate / scale (modal)
- `Tab` — mode toggle (reserved)
- `Numpad 1 / 3 / 7` (+ `Ctrl` for opposite), `Numpad 5` ortho, `F` focus
- `Z` — shading, `MMB` orbit, `Shift+MMB` pan

---

## Roadmap

- [ ] `pkg/io` — glTF 2.0 import/export (canonical alongside OBJ)
- [ ] `pkg/undo` — command/delta history (avoid deep-copy for large meshes)
- [ ] System clipboard + file-watch
- [ ] `macaron-model` — bevel, mirror, inset, boolean
- [ ] `macaron-sculpt` — dyntopo / voxel remesh, brush falloffs, MatCaps
- [ ] `macaron-uv` — palette snapping, island packing
- [ ] `macaron-pixel` — turntable + outline/pixelate shader + PNG sheet

Ideas for future micro-apps: `macaron-retopo` (quad draw on surface), `macaron-blockout` variants, `macaron-spritegen`.

---

## Contributing

New tool = new folder in `cmd/` + implement `engine.Tool`. Keep `pkg/` rendering-free where possible so CLI/headless transforms work. Open an issue with your niche first — the best tools are the most opinionated.

## License

MIT (or your choice) — standard formats keep your data yours.

---

*Macaron — like the pastry, small, precise, and best in a box set.*
