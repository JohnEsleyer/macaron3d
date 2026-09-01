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
│   ├── macaron-ps1/              # PS1 low-poly doll modeler & 256×256 painter
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
macaron init <name>           # scaffold project (accepts path or bare name)
macaron status [path|dev]     # show manifest + mesh presence (dev = ./playground)
macaron doctor [path|dev]     # validate project layout
macaron dev                   # shortcut: status for playground/

macaron-ps1 [path|dev]        # PS1 low-poly doll modeler & 256×256 painter ⭐
macaron-model [path|dev]      # blockout — default "." 
macaron-sculpt [path|dev]     # sculpt
macaron-uv [path|dev]         # uv / palette
macaron-rig [path|dev]        # rig (stub)
macaron-pixel [path|dev]      # pixel / sprite (stub)
```

All `macaron-*` tools accept a single optional argument: project directory (like `code .`). If you pass a file, its parent dir is used. Pass `dev` or `--dev` to auto-resolve the local `playground/` (walks up parents to repo root, falls back to `./playground`).

### Shared hotkeys (every tool)

- `G / R / S` — grab / rotate / scale (modal)
- `Tab` — mode toggle (reserved)
- `Numpad 1 / 3 / 7` (+ `Ctrl` for opposite), `Numpad 5` ortho, `F` focus
- `Z` — shading, `MMB` orbit, `Shift+MMB` pan

---

## Tool UI & Window Conventions

Every Macaron micro-tool adheres to a unified interface standard:

### 1. Summonable & Hideable Panels (`Window` Menu)
- All panels (toolbars, inspectors, scene trees, palettes) can be closed or hidden using standard window buttons (`X`).
- Every tool registers its windows under the top menu bar: **`Window → [✓] <Panel Name>`**, allowing you to summon or dismiss panels on demand for a clean, distraction-free viewport.

### 2. Context-Aware "About" & Shortcuts Reference (`Help → About`)
- Because Macaron tools are specialized, **keyboard shortcuts are context-dependent**. For instance, keys `1`, `2`, `3` lock axes in `macaron-edit`, select limb presets in `macaron-ps1`, or switch brush profiles in `macaron-sculpt`.
- Every tool provides an **`About <Tool Name>`** window (accessible under **`Help → About`**) containing:
  1. A summary of the tool's specialized purpose.
  2. A reference table of key bindings and modal operations.
  3. Standard viewport navigation controls.

### 3. Unified Multi-Selection Standard
- **`Shift + Click`**: Toggles item selection without clearing the rest of your set (applies across Object, Vertex, Edge, and Face modes).
- **`Single Click`**: Clears previous selection and selects the target element.
- **Transform operations** (`G` Grab, `S` Scale, `D` Delete) apply across all multi-selected elements simultaneously.

---

## Development

### Playground project (`playground/`)

The repo ships with a gitignored `playground/` at the root for local testing — it is **not committed**. It is scaffolded like any macaron project:

```
playground/
├── macaron.json
├── textures/ references/
└── .macaron/      # ephemeral cache
```

Recreate or verify it anytime:

```bash
./bin/macaron init playground   # create if missing
./bin/macaron status dev        # same as status playground/
./bin/macaron doctor dev
./bin/macaron dev               # shortcut for status dev
```

`playground/` is listed in `.gitignore:57` and disabled in `textify.yaml:34` (`playground: enabled: false`), so it never pollutes commits or LLM context (`codebase.txt`).

### The `dev` shortcut

Every `macaron-*` tool and the `macaron` manager accept `dev` (or `--dev`) instead of a path:

```bash
# from repo root — no path needed
go run ./cmd/macaron-model dev
go run ./cmd/macaron-sculpt dev
go run ./cmd/macaron-uv dev
go run ./cmd/macaron-rig dev
go run ./cmd/macaron-pixel dev

# installed binaries
macaron-model dev
macaron-sculpt --dev

# manager aliases
macaron status dev
macaron doctor --dev
macaron dev              # same as above
```

How it resolves (`pkg/project/manifest.go:63`):

- `project.IsDevArg(s)` → `s == "dev" || s == "--dev"`
- `project.DevDir()` walks up 6 parents from `os.Getwd()` looking for `playground/`, returns the absolute path or falls back to `"playground"`
- `project.ResolveDir(arg)` is used by all `cmd/*/main.go:62` — `dev` maps to `DevDir()`, everything else passes through unchanged
- `engine.Launch(dir, tool)` then loads `playground/macaron.json` + `mesh.obj` as usual

Works from subdirectories too (e.g. `pkg/`, `playground/textures/`), and `go run ./cmd/... dev` does not require installing binaries.

---

## Macaron PS1 — Low-Poly PS1 Doll Modeler

> **Dedicated micro-tool for PS1-style characters:** reference-sheet contour modeling + doll-part limbs + 256×256 retro painter with smudge/blur.

### Quick start — PS1

```bash
macaron init ps1-hero
cd ps1-hero

# drop reference sheets
cp ~/designs/front.png references/front.png
cp ~/designs/side.png  references/side.png
cp ~/designs/back.png  references/back.png

# launch (or use dev shortcut from repo root)
macaron-ps1 .          # or: macaron-ps1 dev
go run ./cmd/macaron-ps1 dev
```

### What it gives you

| Area | Feature | How |
|------|---------|-----|
| **Reference sheets** | Front / Back / Side image planes | `references/front.png` etc. auto-loaded; opacity/scale sliders + `Numpad 1` (Front/Back), `Numpad 3` (Left/Right), `Numpad 7` (Top) snap to ortho for tracing contours |
| **Doll parts** | Each limb as separate object (Head, Torso, Pelvis, Arm.L/R, Forearm, Hand, Thigh, Shin, Foot) | **Spawn Complete Doll Set** button — cubes with box UVs, independent pivots/transforms, hierarchy list for selection |
| **Polygons** | Create & edit faces | `mesh.Cube` presets + `ExtrudeFace(E)`, `InsetFace(I)`, `SubdivideFace(Shift+D)` in `pkg/mesh/ops.go:11` |
| **Ergonomic modals** | Extrude / Inset / Subdivide without menus | `E` extrudes hovered face by `Extrude Distance`, `I` insets toward center, `Shift+D` subdivides; all save undo (`ctx.History.Save`) |
| **Auto-smooth** | Flat retro vs soft normals | **Flat Shading** = `RecalculateNormals()`, **Auto Smooth** = `CalculateAutoSmoothNormals(creaseAngle)` (`pkg/mesh/ops.go:108`); slider 10°–90° crease threshold |
| **UVs** | PS1 box unwrap | **Auto Box Unwrap UVs** → `AutoGeneratePlanarUVs()` (`pkg/mesh/ops.go:156`) planar projection by face normal |
| **Texture paint** | 256×256 PS1-style canvas | Brush / Eyedropper (`Alt+Click` pick), **Smudge/Blur** tool averages neighborhood with `SmudgePower`; `Nearest Neighbor` filter for pixel look |
| **Save** | `textures/ps1_texture.png` + `mesh.obj` | `OnSave` (`cmd/macaron-ps1/main.go:456`) writes PNG on `File → Save` or **Save Texture PNG** button |

### PS1 workflow (5 steps)

1. **Reference contours** — place sheets in `references/`; `Numpad 1/3` to snap ortho, tune `Opacity`/`Scale` in **Reference Sheet Planes** panel, trace silhouette.
2. **Doll assembly** — **Spawn Complete Doll Set** or add parts one-by-one; select in **Doll Rig & Limbs** list, `DragFloat3` position/rotation per part.
3. **Ergonomic modeling** — hover face → `E` (extrude), `I` (inset), `Shift+D` (subdivide); tweak sliders for distance/factor; undo stack kept.
4. **Shading** — toggle **Flat** (PS1 faceted) vs **Auto Smooth**; adjust `Crease Angle` for hard edges (e.g. 45° keeps 90° box edges sharp, smooths 30° bevels).
5. **256×256 paint** — pick color, set `Brush Radius` 1–16, `Brush` vs `Smudge/Blur` (smudge blends `avg` with `SmudgePower` 0.05–1.0), `Alt+Click` eyedrop, **Clear/Fill** then **Save Texture PNG**.

### Why PS1?

Low-poly PS1 needs *different* ergonomics than general `macaron-model`: fixed 256 canvas, point filtering, reference sheets always-on, limbs as dolls, one-key extrude/inset. `macaron-ps1` removes everything else — woodcarving knife for PS1.

Source: `cmd/macaron-ps1/main.go:1` (`PS1Tool` impl of `engine.Tool`), ops: `pkg/mesh/ops.go:1`.

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
