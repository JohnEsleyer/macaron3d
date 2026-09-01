package main

import (
	"os"

	"macaron/pkg/engine"
	"github.com/AllenDang/cimgui-go/imgui"
)

type UVTool struct{}
func (t *UVTool) Name() string { return "Macaron UV" }
func (t *UVTool) Description() string { return "Palette & UV unwrap" }
func (t *UVTool) Shortcuts() []engine.ShortcutHelp {
	return []engine.ShortcutHelp{
		{Key: "LMB", Description: "Select UV islands / polygon face swatches"},
		{Key: "U", Description: "Unwrap projection (Box / Planar / Cylindrical)"},
	}
}
func (t *UVTool) Init(_ *engine.Context) error { return nil }
func (t *UVTool) OnSave(_ *engine.Context) error { return nil }
func (t *UVTool) Update(_ *engine.Context, _ float32) {}
func (t *UVTool) Draw3D(_ *engine.Context) {}
func (t *UVTool) DrawUI(_ *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10,30), imgui.CondFirstUseEver, imgui.NewVec2(0,0))
	if imgui.BeginV("UV — Palette", nil, 0) {
		imgui.Text("Select faces and snap UV islands to palette swatches.")
		imgui.Text("TODO: box/spherical/cylindrical projection.")
	}
	imgui.End()
}
func main(){ dir:="."; if len(os.Args)>1 {dir=os.Args[1]}; engine.Launch(dir,&UVTool{})}
