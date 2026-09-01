package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"macaron/pkg/engine"
	"macaron/pkg/mesh"
	"macaron/pkg/project"

	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	TexResolution = 256
)

type PaintToolMode int

const (
	ToolBrush PaintToolMode = iota
	ToolSmudge
	ToolBucket
	ToolEyedropper
)

type RefSheet struct {
	Name    string
	Path    string
	Texture rl.Texture2D
	Loaded  bool
	Visible bool
	Opacity float32
	Offset  rl.Vector3
	Scale   float32
}

type PS1Tool struct {
	// Reference sheets
	RefFront RefSheet
	RefBack  RefSheet
	RefSide  RefSheet

	// Doll / Hierarchy
	ActivePart string

	// Modeling state
	ShadingMode     int // 0: Flat, 1: Auto-Smooth
	CreaseAngle     float32
	ExtrudeDistance float32
	InsetFactor     float32

	// PS1 Texture Engine
	PaintCanvas  *image.RGBA
	PaintTex     rl.Texture2D
	PaintMode    PaintToolMode
	BrushColor   [4]float32
	BrushRadius  int
	SmudgePower  float32
	TexDirty     bool
	PixelatedTex bool

	// UI flags
	ShowRefsPanel bool
	ShowDollPanel bool
	ShowUVPanel   bool
	ShowPaintBar  bool
}

func (t *PS1Tool) Name() string { return "Macaron PS1" }
func (t *PS1Tool) Description() string {
	return "PS1-style low-poly doll modeler & 256x256 smudge painter"
}
func (t *PS1Tool) Shortcuts() []engine.ShortcutHelp {
	return []engine.ShortcutHelp{
		{Key: "E", Description: "Extrude hovered face"},
		{Key: "I", Description: "Inset hovered face"},
		{Key: "Shift+D", Description: "Subdivide hovered face"},
		{Key: "Numpad 1/3/7", Description: "Snap Front/Right/Top view"},
		{Key: "Numpad 5", Description: "Toggle Ortho/Perspective"},
		{Key: "Brush/Smudge", Description: "Paint 256x256 texture with 1-16 radius"},
	}
}

func (t *PS1Tool) Init(ctx *engine.Context) error {
	t.BrushColor = [4]float32{0.8, 0.75, 0.7, 1.0}
	t.BrushRadius = 2
	t.SmudgePower = 0.5
	t.ExtrudeDistance = 0.5
	t.InsetFactor = 0.25
	t.CreaseAngle = 45.0
	t.PixelatedTex = true
	t.ShowRefsPanel = true
	t.ShowDollPanel = true
	t.ShowPaintBar = true

	// Initialize 256x256 RGBA Canvas
	t.PaintCanvas = image.NewRGBA(image.Rect(0, 0, TexResolution, TexResolution))
	// Base PS1 grid tint
	for y := 0; y < TexResolution; y++ {
		for x := 0; x < TexResolution; x++ {
			c := uint8(210)
			if ((x/16)+(y/16))%2 == 0 {
				c = 230
			}
			t.PaintCanvas.Set(x, y, color.RGBA{R: c, G: c, B: c, A: 255})
		}
	}
	t.uploadTexture()

	// Setup Reference Sheets from references/ directory if present
	refDir := filepath.Join(ctx.Project.Root, "references")
	t.RefFront = RefSheet{Name: "Front", Path: filepath.Join(refDir, "front.png"), Opacity: 0.65, Scale: 6.0, Offset: rl.Vector3{Z: -3.0}}
	t.RefBack = RefSheet{Name: "Back", Path: filepath.Join(refDir, "back.png"), Opacity: 0.65, Scale: 6.0, Offset: rl.Vector3{Z: 3.0}}
	t.RefSide = RefSheet{Name: "Side", Path: filepath.Join(refDir, "side.png"), Opacity: 0.65, Scale: 6.0, Offset: rl.Vector3{X: -3.0}}

	t.loadRefImage(&t.RefFront)
	t.loadRefImage(&t.RefBack)
	t.loadRefImage(&t.RefSide)

	// Try loading existing texture
	texPath := filepath.Join(ctx.Project.Root, "textures", "ps1_texture.png")
	if f, err := os.Open(texPath); err == nil {
		defer f.Close()
		if img, err := png.Decode(f); err == nil {
			for y := 0; y < TexResolution && y < img.Bounds().Dy(); y++ {
				for x := 0; x < TexResolution && x < img.Bounds().Dx(); x++ {
					t.PaintCanvas.Set(x, y, img.At(x, y))
				}
			}
			t.uploadTexture()
		}
	}

	return nil
}

func (t *PS1Tool) loadRefImage(ref *RefSheet) {
	if _, err := os.Stat(ref.Path); err == nil {
		img := rl.LoadImage(ref.Path)
		if img.Width > 0 {
			ref.Texture = rl.LoadTextureFromImage(img)
			rl.SetTextureFilter(ref.Texture, rl.FilterPoint)
			ref.Loaded = true
			ref.Visible = true
		}
		rl.UnloadImage(img)
	}
}

func (t *PS1Tool) uploadTexture() {
	rawImg := rl.NewImage(
		t.PaintCanvas.Pix,
		int32(TexResolution),
		int32(TexResolution),
		1,
		rl.UncompressedR8g8b8a8,
	)
	if t.PaintTex.ID != 0 {
		rl.UnloadTexture(t.PaintTex)
	}
	t.PaintTex = rl.LoadTextureFromImage(rawImg)
	if t.PixelatedTex {
		rl.SetTextureFilter(t.PaintTex, rl.FilterPoint)
	}
}

// -------------------------------------------------------------
// Texture Painting & Smudge Engine
// -------------------------------------------------------------

func (t *PS1Tool) paintPixel(x, y int, col color.RGBA) {
	if x >= 0 && x < TexResolution && y >= 0 && y < TexResolution {
		t.PaintCanvas.Set(x, y, col)
		t.TexDirty = true
	}
}

func (t *PS1Tool) drawBrush(cx, cy int) {
	r := t.BrushRadius
	col := color.RGBA{
		R: uint8(t.BrushColor[0] * 255),
		G: uint8(t.BrushColor[1] * 255),
		B: uint8(t.BrushColor[2] * 255),
		A: uint8(t.BrushColor[3] * 255),
	}

	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				t.paintPixel(cx+dx, cy+dy, col)
			}
		}
	}
}

// Smudge / Blur tool: blends neighboring pixels towards the average with strength falloff
func (t *PS1Tool) applySmudge(cx, cy int) {
	r := t.BrushRadius + 1
	var rSum, gSum, bSum, count float32

	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			px, py := cx+dx, cy+dy
			if px >= 0 && px < TexResolution && py >= 0 && py < TexResolution {
				col := t.PaintCanvas.RGBAAt(px, py)
				rSum += float32(col.R)
				gSum += float32(col.G)
				bSum += float32(col.B)
				count++
			}
		}
	}

	if count == 0 {
		return
	}
	avgR := rSum / count
	avgG := gSum / count
	avgB := bSum / count

	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				px, py := cx+dx, cy+dy
				if px >= 0 && px < TexResolution && py >= 0 && py < TexResolution {
					cur := t.PaintCanvas.RGBAAt(px, py)
					blendedR := float32(cur.R)*(1.0-t.SmudgePower) + avgR*t.SmudgePower
					blendedG := float32(cur.G)*(1.0-t.SmudgePower) + avgG*t.SmudgePower
					blendedB := float32(cur.B)*(1.0-t.SmudgePower) + avgB*t.SmudgePower
					t.paintPixel(px, py, color.RGBA{
						R: uint8(blendedR),
						G: uint8(blendedG),
						B: uint8(blendedB),
						A: 255,
					})
				}
			}
		}
	}
}

func (t *PS1Tool) sampleColor(cx, cy int) {
	if cx >= 0 && cx < TexResolution && cy >= 0 && cy < TexResolution {
		c := t.PaintCanvas.RGBAAt(cx, cy)
		t.BrushColor = [4]float32{float32(c.R) / 255.0, float32(c.G) / 255.0, float32(c.B) / 255.0, 1.0}
	}
}

// -------------------------------------------------------------
// Update Loop & Modal Shortcuts
// -------------------------------------------------------------

func (t *PS1Tool) Update(ctx *engine.Context, _ float32) {
	// Numpad snaps for orthographic view tracing
	if rl.IsKeyPressed(rl.KeyKp1) {
		if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
			ctx.Camera.Snap("Back", true)
			ctx.SetStatus("View: Back (Ortho)")
		} else {
			ctx.Camera.Snap("Front", true)
			ctx.SetStatus("View: Front (Ortho)")
		}
	}
	if rl.IsKeyPressed(rl.KeyKp3) {
		if rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) {
			ctx.Camera.Snap("Left", true)
			ctx.SetStatus("View: Left (Ortho)")
		} else {
			ctx.Camera.Snap("Right", true)
			ctx.SetStatus("View: Right (Ortho)")
		}
	}
	if rl.IsKeyPressed(rl.KeyKp7) {
		ctx.Camera.Snap("Top", true)
		ctx.SetStatus("View: Top (Ortho)")
	}
	if rl.IsKeyPressed(rl.KeyKp5) {
		ctx.Camera.ToggleOrtho()
		ctx.SetStatus("Toggled Ortho / Perspective")
	}

	obj := ctx.ActiveObject()

	// Modeling Hotkey: E = Extrude hovered face
	if rl.IsKeyPressed(rl.KeyE) && obj != nil && ctx.HoveredFace != -1 {
		ctx.History.Save(ctx.Objects, ctx.SelID)
		newIdx := obj.Mesh.ExtrudeFace(ctx.HoveredFace, t.ExtrudeDistance)
		ctx.HoveredFace = newIdx
		ctx.SetStatus(fmt.Sprintf("Extruded face #%d by %.2f", ctx.HoveredFace, t.ExtrudeDistance))
	}

	// Modeling Hotkey: I = Inset face
	if rl.IsKeyPressed(rl.KeyI) && obj != nil && ctx.HoveredFace != -1 {
		ctx.History.Save(ctx.Objects, ctx.SelID)
		obj.Mesh.InsetFace(ctx.HoveredFace, t.InsetFactor)
		ctx.SetStatus(fmt.Sprintf("Inset face #%d (%.2f)", ctx.HoveredFace, t.InsetFactor))
	}

	// Hotkey: Shift+D = Subdivide face
	if rl.IsKeyPressed(rl.KeyD) && (rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)) && obj != nil && ctx.HoveredFace != -1 {
		ctx.History.Save(ctx.Objects, ctx.SelID)
		obj.Mesh.SubdivideFace(ctx.HoveredFace)
		ctx.SetStatus("Subdivided face")
	}

	// 3D Texture Painting Raycast
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) && !rl.IsKeyDown(rl.KeyLeftAlt) && ctx.RayHit && obj != nil && ctx.HoveredFace != -1 {
		f := obj.Mesh.Faces[ctx.HoveredFace]
		if len(f.UVs) > 0 {
			// Find approximate UV from ray hit on face
			center := obj.Mesh.GetFaceCenter(ctx.HoveredFace)
			uv := f.UVs[0]
			if len(f.UVs) > 1 {
				dist := rl.Vector3Distance(ctx.HitPos, obj.TransformPoint(center))
				uv.X = float32(math.Mod(float64(uv.X+dist*0.2), 1.0))
				uv.Y = float32(math.Mod(float64(uv.Y+dist*0.2), 1.0))
			}
			texX := int(uv.X * float32(TexResolution))
			texY := int(uv.Y * float32(TexResolution))

			switch t.PaintMode {
			case ToolBrush:
				t.drawBrush(texX, texY)
			case ToolSmudge:
				t.applySmudge(texX, texY)
			case ToolEyedropper:
				t.sampleColor(texX, texY)
			}
		}
	}

	// Eyedropper on Alt+Left Click
	if rl.IsMouseButtonDown(rl.MouseButtonLeft) && rl.IsKeyDown(rl.KeyLeftAlt) && ctx.RayHit && obj != nil && ctx.HoveredFace != -1 {
		f := obj.Mesh.Faces[ctx.HoveredFace]
		if len(f.UVs) > 0 {
			texX := int(f.UVs[0].X * float32(TexResolution))
			texY := int(f.UVs[0].Y * float32(TexResolution))
			t.sampleColor(texX, texY)
			ctx.SetStatus("Color picked from model")
		}
	}

	// Commit updated texture to GPU
	if t.TexDirty {
		t.uploadTexture()
		t.TexDirty = false
	}
}

// -------------------------------------------------------------
// 3D Viewport Drawing (Reference Sheets & Reticule)
// -------------------------------------------------------------

func (t *PS1Tool) Draw3D(ctx *engine.Context) {
	// Draw Orthographic Reference Image Planes
	t.drawRefPlane(t.RefFront, rl.Vector3{Y: 1})
	t.drawRefPlane(t.RefBack, rl.Vector3{Y: 1})
	t.drawRefPlane(t.RefSide, rl.Vector3{Y: 1})

	// Brush hover indicator
	if ctx.RayHit {
		col := rl.NewColor(
			uint8(t.BrushColor[0]*255),
			uint8(t.BrushColor[1]*255),
			uint8(t.BrushColor[2]*255),
			180,
		)
		radius3D := float32(t.BrushRadius) * 0.05
		rl.DrawSphereWires(ctx.HitPos, radius3D, 8, 8, col)
		rl.DrawLine3D(ctx.HitPos, rl.Vector3Add(ctx.HitPos, rl.Vector3Scale(ctx.HitNormal, 0.4)), rl.Yellow)
	}
}

func (t *PS1Tool) drawRefPlane(ref RefSheet, center rl.Vector3) {
	if !ref.Loaded || !ref.Visible {
		return
	}
	pos := rl.Vector3Add(center, ref.Offset)
	halfSize := ref.Scale * 0.5
	tint := rl.ColorAlpha(rl.White, ref.Opacity)

	if ref.Name == "Front" || ref.Name == "Back" {
		p0 := rl.Vector3{X: pos.X - halfSize, Y: pos.Y - halfSize, Z: pos.Z}
		p1 := rl.Vector3{X: pos.X + halfSize, Y: pos.Y - halfSize, Z: pos.Z}
		p2 := rl.Vector3{X: pos.X + halfSize, Y: pos.Y + halfSize, Z: pos.Z}
		p3 := rl.Vector3{X: pos.X - halfSize, Y: pos.Y + halfSize, Z: pos.Z}

		rl.DrawCubeWires(pos, ref.Scale, ref.Scale, 0.01, rl.NewColor(100, 200, 255, 120))
		rl.DrawTriangle3D(p0, p1, p2, tint)
		rl.DrawTriangle3D(p0, p2, p3, tint)
	} else if ref.Name == "Side" {
		p0 := rl.Vector3{X: pos.X, Y: pos.Y - halfSize, Z: pos.Z - halfSize}
		p1 := rl.Vector3{X: pos.X, Y: pos.Y - halfSize, Z: pos.Z + halfSize}
		p2 := rl.Vector3{X: pos.X, Y: pos.Y + halfSize, Z: pos.Z + halfSize}
		p3 := rl.Vector3{X: pos.X, Y: pos.Y + halfSize, Z: pos.Z - halfSize}

		rl.DrawCubeWires(pos, 0.01, ref.Scale, ref.Scale, rl.NewColor(100, 200, 255, 120))
		rl.DrawTriangle3D(p0, p1, p2, tint)
		rl.DrawTriangle3D(p0, p2, p3, tint)
	}
}

// -------------------------------------------------------------
// UI Windows & Doll Management
// -------------------------------------------------------------

func (t *PS1Tool) DrawUI(ctx *engine.Context) {
	t.drawDollHierarchyPanel(ctx)
	t.drawModelingToolsPanel(ctx)
	t.drawPaintToolbar(ctx)
	t.drawReferenceSheetsPanel(ctx)
}

func (t *PS1Tool) drawDollHierarchyPanel(ctx *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10, 30), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(250, 320), imgui.CondFirstUseEver)
	if imgui.BeginV("Doll Rig & Limbs", nil, 0) {
		imgui.TextDisabled("Independent Body Part Meshes")

		limbs := []struct {
			name string
			pos  rl.Vector3
			size float32
		}{
			{"Head", rl.Vector3{Y: 2.8}, 0.8},
			{"Torso", rl.Vector3{Y: 1.6}, 1.2},
			{"Pelvis", rl.Vector3{Y: 0.8}, 0.9},
			{"Arm.L", rl.Vector3{X: 1.0, Y: 1.8}, 0.5},
			{"Forearm.L", rl.Vector3{X: 1.6, Y: 1.8}, 0.45},
			{"Hand.L", rl.Vector3{X: 2.1, Y: 1.8}, 0.35},
			{"Arm.R", rl.Vector3{X: -1.0, Y: 1.8}, 0.5},
			{"Forearm.R", rl.Vector3{X: -1.6, Y: 1.8}, 0.45},
			{"Hand.R", rl.Vector3{X: -2.1, Y: 1.8}, 0.35},
			{"Thigh.L", rl.Vector3{X: 0.4, Y: 0.2}, 0.55},
			{"Shin.L", rl.Vector3{X: 0.4, Y: -0.6}, 0.5},
			{"Foot.L", rl.Vector3{X: 0.4, Y: -1.1, Z: 0.2}, 0.4},
			{"Thigh.R", rl.Vector3{X: -0.4, Y: 0.2}, 0.55},
			{"Shin.R", rl.Vector3{X: -0.4, Y: -0.6}, 0.5},
			{"Foot.R", rl.Vector3{X: -0.4, Y: -1.1, Z: 0.2}, 0.4},
		}

		if imgui.Button("Spawn Complete Doll Set") {
			for _, l := range limbs {
				t.addDollPart(ctx, l.name, l.pos, l.size)
			}
			ctx.SetStatus("Spawned complete PS1 doll character base")
		}

		imgui.Separator()
		imgui.Text("Active Objects:")
		for i := range ctx.Objects {
			o := &ctx.Objects[i]
			selected := (o.ID == ctx.SelID)
			if imgui.SelectableBoolV(o.Name, selected, 0, imgui.NewVec2(0, 0)) {
				ctx.SelID = o.ID
			}
		}

		imgui.Separator()
		if o := ctx.ActiveObject(); o != nil {
			imgui.Text("Transform: " + o.Name)
			pos := [3]float32{o.Position.X, o.Position.Y, o.Position.Z}
			if imgui.DragFloat3("Position", &pos) {
				o.Position = rl.Vector3{X: pos[0], Y: pos[1], Z: pos[2]}
			}
			rot := [3]float32{o.Rotation.X, o.Rotation.Y, o.Rotation.Z}
			if imgui.DragFloat3("Rotation", &rot) {
				o.Rotation = rl.Vector3{X: rot[0], Y: rot[1], Z: rot[2]}
			}
		}
	}
	imgui.End()
}

func (t *PS1Tool) addDollPart(ctx *engine.Context, name string, pos rl.Vector3, size float32) {
	cube := mesh.Cube(size)
	cube.AutoGeneratePlanarUVs()
	ctx.NextID++
	obj := mesh.Object{
		ID:       ctx.NextID,
		Name:     name,
		Position: pos,
		Scale:    rl.Vector3{X: 1, Y: 1, Z: 1},
		Visible:  true,
		Mesh:     cube,
		Material: mesh.Material{
			Color:     [4]float32{0.85, 0.85, 0.85, 1.0},
			Roughness: 0.8,
		},
	}
	ctx.Objects = append(ctx.Objects, obj)
	ctx.SelID = obj.ID
}

func (t *PS1Tool) drawModelingToolsPanel(ctx *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(10, 360), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(250, 280), imgui.CondFirstUseEver)
	if imgui.BeginV("PS1 Modeling Ops", nil, 0) {
		obj := ctx.ActiveObject()

		imgui.Text("Contour & Polygonal Ops:")
		imgui.SliderFloat("Extrude Dist (E)", &t.ExtrudeDistance, 0.05, 2.0)
		if imgui.Button("Extrude Selected Face") && obj != nil && ctx.HoveredFace != -1 {
			ctx.History.Save(ctx.Objects, ctx.SelID)
			obj.Mesh.ExtrudeFace(ctx.HoveredFace, t.ExtrudeDistance)
		}

		imgui.SliderFloat("Inset Factor (I)", &t.InsetFactor, 0.05, 0.8)
		if imgui.Button("Inset Selected Face") && obj != nil && ctx.HoveredFace != -1 {
			ctx.History.Save(ctx.Objects, ctx.SelID)
			obj.Mesh.InsetFace(ctx.HoveredFace, t.InsetFactor)
		}

		if imgui.Button("Subdivide Face (Shift+D)") && obj != nil && ctx.HoveredFace != -1 {
			ctx.History.Save(ctx.Objects, ctx.SelID)
			obj.Mesh.SubdivideFace(ctx.HoveredFace)
		}

		imgui.Separator()
		imgui.Text("Shading & Normals:")
		if imgui.RadioButtonBool("Flat Shading (Retro)", t.ShadingMode == 0) {
			t.ShadingMode = 0
			if obj != nil {
				obj.Mesh.RecalculateNormals()
			}
		}
		if imgui.RadioButtonBool("Auto Smooth", t.ShadingMode == 1) {
			t.ShadingMode = 1
			if obj != nil {
				obj.Mesh.CalculateAutoSmoothNormals(t.CreaseAngle)
			}
		}
		if t.ShadingMode == 1 {
			if imgui.SliderFloat("Crease Angle", &t.CreaseAngle, 10.0, 90.0) && obj != nil {
				obj.Mesh.CalculateAutoSmoothNormals(t.CreaseAngle)
			}
		}

		imgui.Separator()
		if imgui.Button("Auto Box Unwrap UVs") && obj != nil {
			obj.Mesh.AutoGeneratePlanarUVs()
			ctx.SetStatus("Auto-generated UVs for " + obj.Name)
		}
	}
	imgui.End()
}

func (t *PS1Tool) drawPaintToolbar(ctx *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(270, 30), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(320, 240), imgui.CondFirstUseEver)
	if imgui.BeginV("256x256 Texture & Smudge", nil, 0) {
		imgui.ColorEdit4("Paint Color", &t.BrushColor)

		imgui.Separator()
		imgui.Text("Paint Tools:")
		if imgui.RadioButtonBool("Brush (Draw)", t.PaintMode == ToolBrush) {
			t.PaintMode = ToolBrush
		}
		imgui.SameLine()
		if imgui.RadioButtonBool("Smudge / Blur", t.PaintMode == ToolSmudge) {
			t.PaintMode = ToolSmudge
		}
		imgui.SameLine()
		if imgui.RadioButtonBool("Eyedropper", t.PaintMode == ToolEyedropper) {
			t.PaintMode = ToolEyedropper
		}

		radiusInt := int32(t.BrushRadius)
		if imgui.SliderInt("Brush Radius", &radiusInt, 1, 16) {
			t.BrushRadius = int(radiusInt)
		}

		if t.PaintMode == ToolSmudge {
			imgui.SliderFloat("Smudge Strength", &t.SmudgePower, 0.05, 1.0)
		}

		imgui.Checkbox("Nearest Neighbor (PS1 Pixel Look)", &t.PixelatedTex)

		imgui.Separator()
		if imgui.Button("Clear / Fill Texture") {
			col := color.RGBA{
				R: uint8(t.BrushColor[0] * 255),
				G: uint8(t.BrushColor[1] * 255),
				B: uint8(t.BrushColor[2] * 255),
				A: 255,
			}
			for y := 0; y < TexResolution; y++ {
				for x := 0; x < TexResolution; x++ {
					t.PaintCanvas.Set(x, y, col)
				}
			}
			t.uploadTexture()
		}
		imgui.SameLine()
		if imgui.Button("Save Texture PNG") {
			_ = t.OnSave(ctx)
			ctx.SetStatus("Texture saved to textures/ps1_texture.png")
		}
	}
	imgui.End()
}

func (t *PS1Tool) drawReferenceSheetsPanel(ctx *engine.Context) {
	imgui.SetNextWindowPosV(imgui.NewVec2(270, 280), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
	imgui.SetNextWindowSizeV(imgui.NewVec2(320, 200), imgui.CondFirstUseEver)
	if imgui.BeginV("Reference Sheet Planes", nil, 0) {
		imgui.TextDisabled("Place front/back/side sheets in references/")

		drawRefRow := func(ref *RefSheet) {
			imgui.PushIDStr(ref.Name)
			imgui.Checkbox(ref.Name+" Visible", &ref.Visible)
			imgui.SameLine()
			imgui.SliderFloat("Opacity", &ref.Opacity, 0.1, 1.0)
			imgui.SliderFloat("Scale", &ref.Scale, 1.0, 20.0)
			imgui.PopID()
		}

		drawRefRow(&t.RefFront)
		drawRefRow(&t.RefBack)
		drawRefRow(&t.RefSide)

		if imgui.Button("Snap Front (Numpad 1)") {
			ctx.Camera.Snap("Front", true)
		}
		imgui.SameLine()
		if imgui.Button("Snap Side (Numpad 3)") {
			ctx.Camera.Snap("Right", true)
		}
	}
	imgui.End()
}

// -------------------------------------------------------------
// Save hook
// -------------------------------------------------------------

func (t *PS1Tool) OnSave(ctx *engine.Context) error {
	texDir := filepath.Join(ctx.Project.Root, "textures")
	_ = os.MkdirAll(texDir, 0755)
	outPath := filepath.Join(texDir, "ps1_texture.png")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, t.PaintCanvas)
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = project.ResolveDir(os.Args[1])
	}
	engine.Launch(dir, &PS1Tool{})
}
