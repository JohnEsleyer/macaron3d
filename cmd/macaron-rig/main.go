package main

import (
	"os"

	"macaron/pkg/engine"
	"github.com/AllenDang/cimgui-go/imgui"
)

type RigTool struct{}
func (t *RigTool) Name() string { return "Macaron Rig" }
func (t *RigTool) Description() string { return "Bone & skeleton poser (stub)" }
func (t *RigTool) Init(_ *engine.Context) error { return nil }
func (t *RigTool) OnSave(_ *engine.Context) error { return nil }
func (t *RigTool) Update(_ *engine.Context, _ float32) {}
func (t *RigTool) Draw3D(_ *engine.Context) {}
func (t *RigTool) DrawUI(_ *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10,30), imgui.CondFirstUseEver, imgui.NewVec2(0,0))
	if imgui.BeginV("Rig — Bones", nil, 0) { imgui.Text("TODO: click to place bones, weight paint."); }
	imgui.End()
}
func main(){ dir:="."; if len(os.Args)>1 {dir=os.Args[1]}; engine.Launch(dir,&RigTool{})}
