package main

import (
	"os"

	"macaron/pkg/engine"
	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type SculptTool struct {
	Radius   float32
	Strength float32
}

func (t *SculptTool) Name() string        { return "Macaron Sculpt" }
func (t *SculptTool) Description() string { return "Digital clay — voxel remesh placeholder" }
func (t *SculptTool) Init(_ *engine.Context) error { t.Radius = 0.6; t.Strength = 0.5; return nil }
func (t *SculptTool) OnSave(_ *engine.Context) error { return nil }
func (t *SculptTool) Update(_ *engine.Context, _ float32) {}
func (t *SculptTool) Draw3D(ctx *engine.Context) {
	if ctx.RayHit {
		rl.DrawSphereWires(ctx.HitPos, t.Radius, 12, 12, rl.NewColor(255, 200, 50, 180))
	}
}
func (t *SculptTool) DrawUI(_ *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10, 30), imgui.CondFirstUseEver, imgui.NewVec2(0,0))
	if imgui.BeginV("Sculpt — Brush", nil, 0) {
		imgui.SliderFloat("Radius", &t.Radius, 0.1, 2.0)
		imgui.SliderFloat("Strength", &t.Strength, 0.05, 1.0)
	}
	imgui.End()
}
func main(){ dir:="."; if len(os.Args)>1 {dir=os.Args[1]}; engine.Launch(dir,&SculptTool{})}
