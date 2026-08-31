package main

import (
	"os"

	"macaron/pkg/engine"
	"macaron/pkg/mesh"

	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type ModelTool struct{}

func (t *ModelTool) Name() string        { return "Macaron Model" }
func (t *ModelTool) Description() string { return "Low-poly blockout & box modeling" }
func (t *ModelTool) Init(_ *engine.Context) error { return nil }
func (t *ModelTool) OnSave(_ *engine.Context) error { return nil }

func (t *ModelTool) Update(ctx *engine.Context, _ float32) {
	if rl.IsKeyPressed(rl.KeyTab) {
		ctx.SetStatus("Tab toggles object/edit — reserved for future")
	}
	// G: grab selected object
	if rl.IsKeyPressed(rl.KeyG) {
		if o := ctx.ActiveObject(); o != nil {
			o.Position.X += 0.5
			ctx.SetStatus("Grab: +0.5 X (placeholder — wire to modal)")
		}
	}
}

func (t *ModelTool) Draw3D(_ *engine.Context) {}

func (t *ModelTool) DrawUI(ctx *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10, 30), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(240, 200), imgui.CondFirstUseEver)
	if imgui.BeginV("Model — Tools", nil, imgui.WindowFlagsNone) {
		if imgui.Button("Add Cube") {
			ctx.AddObject("Cube", mesh.Cube(2))
		}
		if imgui.Button("Add Plane") {
			ctx.AddObject("Plane", mesh.Plane(4))
		}
		if imgui.Button("Add UV Sphere") {
			ctx.AddObject("Sphere", mesh.UVSphere(1, 12, 16))
		}
		imgui.Separator()
		imgui.Text("Shading: Z toggles wireframe (todo)")
		if o := ctx.ActiveObject(); o != nil {
			imgui.Text(o.Name)
			col := o.Material.Color
			if imgui.ColorEdit4("Base Color", &col) {
				o.Material.Color = col
			}
		}
	}
	imgui.End()
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	engine.Launch(dir, &ModelTool{})
}
