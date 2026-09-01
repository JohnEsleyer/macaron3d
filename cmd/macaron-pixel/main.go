package main

import (
	"os"

	"macaron/pkg/engine"
	"github.com/AllenDang/cimgui-go/imgui"
)

type PixelTool struct{}
func (t *PixelTool) Name() string { return "Macaron Pixel" }
func (t *PixelTool) Description() string { return "3D → 2D sprite / pixel-art exporter (stub)" }
func (t *PixelTool) Shortcuts() []engine.ShortcutHelp {
	return []engine.ShortcutHelp{
		{Key: "Space", Description: "Toggle 8-directional turntable preview"},
		{Key: "F12", Description: "Export 2D sprite sheet PNG"},
	}
}
func (t *PixelTool) Init(_ *engine.Context) error { return nil }
func (t *PixelTool) OnSave(_ *engine.Context) error { return nil }
func (t *PixelTool) Update(_ *engine.Context, _ float32) {}
func (t *PixelTool) Draw3D(_ *engine.Context) {}
func (t *PixelTool) DrawUI(_ *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10,30), imgui.CondFirstUseEver, imgui.NewVec2(0,0))
	if imgui.BeginV("Pixel — Export", nil, 0) {
		imgui.Text("TODO: 8-direction turntable + pixelation shader + PNG sheet export.")
		if imgui.Button("Export sprites (stub)") {}
	}
	imgui.End()
}
func main(){ dir:="."; if len(os.Args)>1 {dir=os.Args[1]}; engine.Launch(dir,&PixelTool{})}
