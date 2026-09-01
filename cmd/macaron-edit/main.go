package main

import (
	"fmt"
	"math"
	"os"

	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
	"macaron/pkg/engine"
	"macaron/pkg/mesh"
	"macaron/pkg/project"
)

type SelectMode int

const (
	ModeObject SelectMode = iota
	ModeVertex
	ModeEdge
	ModeFace
)

type ViewMode int

const (
	ViewEdit ViewMode = iota
	ViewWireframe
	ViewObject
)

type ModalOp int

const (
	OpNone ModalOp = iota
	OpGrab
	OpScale
	OpExtrude
	OpInset
	OpCut
)

type AxisLock int

const (
	AxisNone AxisLock = iota
	AxisX
	AxisY
	AxisZ
)

type EditTool struct {
	Mode     SelectMode
	ViewMode ViewMode

	// Window Visibility Flags
	ShowTopBar    bool
	ShowSceneTree bool
	ShowInspector bool

	// Hover states
	HoveredVert int
	HoveredEdge int
	HoveredFace int
	HoveredObj  int

	// Multi-Selection Sets
	SelectedObjects map[int]bool
	SelectedVerts   map[int]bool
	SelectedEdges   map[int]bool
	SelectedFaces   map[int]bool

	// Drag Rectangle Selection State
	IsDraggingBox  bool
	DragStartPos   rl.Vector2
	DragCurrentPos rl.Vector2

	// Modal State
	ActiveOp        ModalOp
	ActiveAxis      AxisLock
	InitialMouse    rl.Vector2
	ModalCenter     rl.Vector3
	OriginalVerts   []mesh.Vertex
	OriginalObjPos  map[int]rl.Vector3
	OriginalObjScal map[int]rl.Vector3

	// Cut Tool State
	CutStep   int
	CutPoint1 rl.Vector3
	CutFace   int

	BannerMsg   string
	BannerTimer float32
}

func (t *EditTool) Name() string        { return "Macaron Edit" }
func (t *EditTool) Description() string { return "Scene hierarchy & multi-selection 3D polygon editor" }

func (t *EditTool) Shortcuts() []engine.ShortcutHelp {
	return []engine.ShortcutHelp{
		{Key: "G", Description: "Grab / Move selected object(s) or sub-elements"},
		{Key: "S", Description: "Scale selected object(s) or sub-elements around pivot"},
		{Key: "E", Description: "Extrude selected faces along normal (Edit mode)"},
		{Key: "I", Description: "Inset selected faces inward (Edit mode)"},
		{Key: "C", Description: "Cut face with precision mouse guide (Edit mode)"},
		{Key: "D / Delete", Description: "Delete selection (objects or mesh elements)"},
		{Key: "1 / 2 / 3", Description: "Lock Grab/Scale to X (1), Y (2), or Z (3) axis"},
		{Key: "LMB Drag", Description: "Drag marquee box selection"},
		{Key: "Shift+LMB", Description: "Add to multi-selection set"},
		{Key: "Left-Click", Description: "Select object / apply modal changes"},
		{Key: "Right-Click", Description: "Cancel modal operation / clear selection"},
	}
}

func (t *EditTool) Init(ctx *engine.Context) error {
	t.Mode = ModeObject
	t.ViewMode = ViewEdit
	t.ShowTopBar = true
	t.ShowSceneTree = true
	t.ShowInspector = true

	t.HoveredVert = -1
	t.HoveredEdge = -1
	t.HoveredFace = -1
	t.HoveredObj = -1

	t.SelectedObjects = make(map[int]bool)
	t.SelectedVerts = make(map[int]bool)
	t.SelectedEdges = make(map[int]bool)
	t.SelectedFaces = make(map[int]bool)

	if len(ctx.Objects) == 0 {
		ctx.AddObject("SceneRoot", mesh.Cube(2))
	}
	if ctx.Objects[0].Name != "SceneRoot" && ctx.Objects[0].ParentID == 0 {
		ctx.Objects[0].Name = "SceneRoot"
	}
	ctx.SelID = ctx.Objects[0].ID
	t.SelectedObjects[ctx.SelID] = true
	return nil
}

func (t *EditTool) OnSave(_ *engine.Context) error { return nil }

func (t *EditTool) showBanner(msg string) {
	t.BannerMsg = msg
	t.BannerTimer = 3.5
}

func (t *EditTool) clearSubSelections() {
	t.SelectedVerts = make(map[int]bool)
	t.SelectedEdges = make(map[int]bool)
	t.SelectedFaces = make(map[int]bool)
}

func (t *EditTool) startModal(ctx *engine.Context, op ModalOp) {
	t.ActiveOp = op
	t.ActiveAxis = AxisNone
	t.InitialMouse = rl.GetMousePosition()
	t.OriginalObjPos = make(map[int]rl.Vector3)
	t.OriginalObjScal = make(map[int]rl.Vector3)

	obj := ctx.ActiveObject()
	if obj != nil {
		t.OriginalVerts = append([]mesh.Vertex(nil), obj.Mesh.Vertices...)
	}

	var centerSum rl.Vector3
	count := 0

	if t.Mode == ModeObject || t.ViewMode == ViewObject {
		for id := range t.SelectedObjects {
			for _, o := range ctx.Objects {
				if o.ID == id {
					t.OriginalObjPos[id] = o.Position
					t.OriginalObjScal[id] = o.Scale
					centerSum = rl.Vector3Add(centerSum, o.Position)
					count++
				}
			}
		}
	} else if obj != nil {
		if t.Mode == ModeVertex {
			for vi := range t.SelectedVerts {
				centerSum = rl.Vector3Add(centerSum, obj.Mesh.Vertices[vi].Position)
				count++
			}
		} else if t.Mode == ModeEdge {
			for ei := range t.SelectedEdges {
				edge := obj.Mesh.Edges[ei]
				centerSum = rl.Vector3Add(centerSum, obj.Mesh.Vertices[edge.V1].Position)
				centerSum = rl.Vector3Add(centerSum, obj.Mesh.Vertices[edge.V2].Position)
				count += 2
			}
		} else if t.Mode == ModeFace {
			for fi := range t.SelectedFaces {
				centerSum = rl.Vector3Add(centerSum, obj.Mesh.GetFaceCenter(fi))
				count++
			}
		}
	}

	if count > 0 {
		t.ModalCenter = rl.Vector3Scale(centerSum, 1.0/float32(count))
	} else if obj != nil {
		t.ModalCenter = obj.Position
	}
}

func (t *EditTool) applyModal(ctx *engine.Context) {
	ctx.History.Save(ctx.Objects, ctx.SelID)
	t.ActiveOp = OpNone
	t.ActiveAxis = AxisNone
	t.OriginalVerts = nil
	t.OriginalObjPos = nil
	t.OriginalObjScal = nil
	t.showBanner("Applied changes")
}

func (t *EditTool) cancelModal(ctx *engine.Context) {
	if t.Mode == ModeObject || t.ViewMode == ViewObject {
		for id, pos := range t.OriginalObjPos {
			for i := range ctx.Objects {
				if ctx.Objects[i].ID == id {
					ctx.Objects[i].Position = pos
					ctx.Objects[i].Scale = t.OriginalObjScal[id]
				}
			}
		}
	} else if obj := ctx.ActiveObject(); obj != nil && t.OriginalVerts != nil {
		obj.Mesh.Vertices = append([]mesh.Vertex(nil), t.OriginalVerts...)
		obj.Mesh.RebuildEdges()
		obj.Mesh.RecalculateNormals()
	}
	t.ActiveOp = OpNone
	t.ActiveAxis = AxisNone
	t.OriginalVerts = nil
	t.OriginalObjPos = nil
	t.OriginalObjScal = nil
	t.showBanner("Cancelled operation")
}

func (t *EditTool) Update(ctx *engine.Context, dt float32) {
	// Sync Engine Context RenderMode with Tool ViewMode
	switch t.ViewMode {
	case ViewEdit:
		ctx.RenderMode = engine.RenderModeDefault
	case ViewWireframe:
		ctx.RenderMode = engine.RenderModeWireframe
	case ViewObject:
		ctx.RenderMode = engine.RenderModeObject
	}

	if t.BannerTimer > 0 {
		t.BannerTimer -= dt
	}

	allowUI := !imgui.CurrentIO().WantCaptureMouse() && !imgui.CurrentIO().WantCaptureKeyboard()
	obj := ctx.ActiveObject()

	// In-Progress Modal Operations (Grab & Scale work seamlessly in Object View)
	if t.ActiveOp != OpNone && t.ActiveOp != OpCut {
		if rl.IsKeyPressed(rl.KeyOne) {
			t.ActiveAxis = AxisX
		} else if rl.IsKeyPressed(rl.KeyTwo) {
			t.ActiveAxis = AxisY
		} else if rl.IsKeyPressed(rl.KeyThree) {
			t.ActiveAxis = AxisZ
		}

		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			t.applyModal(ctx)
			return
		}
		if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
			t.cancelModal(ctx)
			return
		}

		mouseDelta := rl.Vector2Subtract(rl.GetMousePosition(), t.InitialMouse)
		dist := (mouseDelta.X - mouseDelta.Y) * 0.01

		switch t.ActiveOp {
		case OpGrab:
			offset := rl.Vector3{X: mouseDelta.X * 0.015, Y: -mouseDelta.Y * 0.015, Z: 0}
			if t.ActiveAxis == AxisX {
				offset = rl.Vector3{X: mouseDelta.X * 0.015, Y: 0, Z: 0}
			} else if t.ActiveAxis == AxisY {
				offset = rl.Vector3{X: 0, Y: -mouseDelta.Y * 0.015, Z: 0}
			} else if t.ActiveAxis == AxisZ {
				offset = rl.Vector3{X: 0, Y: 0, Z: mouseDelta.Y * 0.015}
			}

			if t.Mode == ModeObject || t.ViewMode == ViewObject {
				for id, origPos := range t.OriginalObjPos {
					for i := range ctx.Objects {
						if ctx.Objects[i].ID == id {
							ctx.Objects[i].Position = rl.Vector3Add(origPos, offset)
						}
					}
				}
			} else if obj != nil {
				for i := range obj.Mesh.Vertices {
					if (t.Mode == ModeVertex && t.SelectedVerts[i]) ||
						(t.Mode == ModeEdge && t.isVertInSelectedEdges(obj, i)) ||
						(t.Mode == ModeFace && t.isVertInSelectedFaces(obj, i)) {
						obj.Mesh.Vertices[i].Position = rl.Vector3Add(t.OriginalVerts[i].Position, offset)
					}
				}
				obj.Mesh.RecalculateNormals()
			}

		case OpScale:
			factor := 1.0 + dist
			if factor < 0.01 {
				factor = 0.01
			}

			if t.Mode == ModeObject || t.ViewMode == ViewObject {
				for id, origScal := range t.OriginalObjScal {
					for i := range ctx.Objects {
						if ctx.Objects[i].ID == id {
							sx, sy, sz := origScal.X*factor, origScal.Y*factor, origScal.Z*factor
							if t.ActiveAxis == AxisX {
								ctx.Objects[i].Scale.X = sx
							} else if t.ActiveAxis == AxisY {
								ctx.Objects[i].Scale.Y = sy
							} else if t.ActiveAxis == AxisZ {
								ctx.Objects[i].Scale.Z = sz
							} else {
								ctx.Objects[i].Scale = rl.Vector3{X: sx, Y: sy, Z: sz}
							}
						}
					}
				}
			} else if obj != nil {
				for i := range obj.Mesh.Vertices {
					if (t.Mode == ModeVertex && t.SelectedVerts[i]) ||
						(t.Mode == ModeEdge && t.isVertInSelectedEdges(obj, i)) ||
						(t.Mode == ModeFace && t.isVertInSelectedFaces(obj, i)) {
						p := t.OriginalVerts[i].Position
						diff := rl.Vector3Subtract(p, t.ModalCenter)
						if t.ActiveAxis == AxisX {
							diff.X *= factor
						} else if t.ActiveAxis == AxisY {
							diff.Y *= factor
						} else if t.ActiveAxis == AxisZ {
							diff.Z *= factor
						} else {
							diff = rl.Vector3Scale(diff, factor)
						}
						obj.Mesh.Vertices[i].Position = rl.Vector3Add(t.ModalCenter, diff)
					}
				}
				obj.Mesh.RecalculateNormals()
			}

		case OpExtrude:
			if obj != nil && t.ViewMode != ViewObject {
				offsetDist := dist * 1.5
				for fi := range t.SelectedFaces {
					norm := obj.Mesh.Faces[fi].Normal
					if rl.Vector3Length(norm) < 0.001 {
						norm = rl.Vector3{Y: 1}
					}
					moveVec := rl.Vector3Scale(norm, offsetDist)
					for _, idx := range obj.Mesh.Faces[fi].Indices {
						obj.Mesh.Vertices[idx].Position = rl.Vector3Add(t.OriginalVerts[idx].Position, moveVec)
					}
				}
				obj.Mesh.RecalculateNormals()
			}

		case OpInset:
			if obj != nil && t.ViewMode != ViewObject {
				factor := float32(math.Abs(float64(dist)))
				if factor > 0.95 {
					factor = 0.95
				}
				for fi := range t.SelectedFaces {
					fc := obj.Mesh.GetFaceCenter(fi)
					for _, idx := range obj.Mesh.Faces[fi].Indices {
						orig := t.OriginalVerts[idx].Position
						dir := rl.Vector3Subtract(fc, orig)
						obj.Mesh.Vertices[idx].Position = rl.Vector3Add(orig, rl.Vector3Scale(dir, factor))
					}
				}
				obj.Mesh.RecalculateNormals()
			}
		}
		return
	}

	// Cut Modal Tool
	if t.ActiveOp == OpCut {
		if rl.IsMouseButtonPressed(rl.MouseButtonRight) {
			t.ActiveOp = OpNone
			t.CutStep = 0
			t.showBanner("Cut cancelled")
			return
		}
		if ctx.RayHit && rl.IsMouseButtonPressed(rl.MouseButtonLeft) && obj != nil {
			if t.CutStep == 0 {
				t.CutPoint1 = ctx.HitPos
				t.CutFace = ctx.HoveredFace
				t.CutStep = 1
				t.showBanner("Cut: Point 1 set. Click second point across face.")
			} else if t.CutStep == 1 {
				if obj.Mesh.CutFace(t.CutFace, t.CutPoint1, ctx.HitPos) {
					ctx.History.Save(ctx.Objects, ctx.SelID)
					t.showBanner("Cut face successfully")
				} else {
					t.showBanner("Cut failed: points must cross face boundaries")
				}
				t.ActiveOp = OpNone
				t.CutStep = 0
			}
		}
		return
	}

	if !allowUI {
		return
	}

	mousePos := rl.GetMousePosition()
	isAlt := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)

	// Drag Rectangle Selection Handling
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && !isAlt && t.ActiveOp == OpNone {
		t.DragStartPos = mousePos
		t.DragCurrentPos = mousePos
		t.IsDraggingBox = false
	}

	if rl.IsMouseButtonDown(rl.MouseButtonLeft) && !isAlt && t.ActiveOp == OpNone {
		t.DragCurrentPos = mousePos
		if rl.Vector2Distance(t.DragStartPos, t.DragCurrentPos) > 6.0 {
			t.IsDraggingBox = true
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && !isAlt {
		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
		if t.IsDraggingBox {
			t.executeBoxSelection(ctx, shift)
			t.IsDraggingBox = false
		} else {
			// Single-Click Selection
			if t.Mode == ModeObject || t.ViewMode == ViewObject {
				if t.HoveredObj != -1 {
					if !shift {
						t.SelectedObjects = make(map[int]bool)
					}
					if t.SelectedObjects[t.HoveredObj] {
						delete(t.SelectedObjects, t.HoveredObj)
						if len(t.SelectedObjects) == 0 {
							t.SelectedObjects[t.HoveredObj] = true
							ctx.SelID = t.HoveredObj
						} else if !t.SelectedObjects[ctx.SelID] {
							for id := range t.SelectedObjects {
								ctx.SelID = id
								break
							}
						}
					} else {
						t.SelectedObjects[t.HoveredObj] = true
						ctx.SelID = t.HoveredObj
					}
				}
			} else {
				if !shift {
					t.clearSubSelections()
				}
				if t.Mode == ModeVertex && t.HoveredVert != -1 {
					if t.SelectedVerts[t.HoveredVert] {
						delete(t.SelectedVerts, t.HoveredVert)
					} else {
						t.SelectedVerts[t.HoveredVert] = true
					}
				} else if t.Mode == ModeEdge && t.HoveredEdge != -1 {
					if t.SelectedEdges[t.HoveredEdge] {
						delete(t.SelectedEdges, t.HoveredEdge)
					} else {
						t.SelectedEdges[t.HoveredEdge] = true
					}
				} else if t.Mode == ModeFace && t.HoveredFace != -1 {
					if t.SelectedFaces[t.HoveredFace] {
						delete(t.SelectedFaces, t.HoveredFace)
					} else {
						t.SelectedFaces[t.HoveredFace] = true
					}
				}
			}
		}
	}

	// Hover Detection (Object hover always active in Object Mode and Object View)
	t.HoveredVert = -1
	t.HoveredEdge = -1
	t.HoveredFace = ctx.HoveredFace
	t.HoveredObj = -1

	if t.Mode == ModeObject || t.ViewMode == ViewObject {
		bestDist := float32(1e9)
		bestID := -1
		for _, o := range ctx.Objects {
			if hit, dist, _ := o.Raycast(rl.GetMouseRay(mousePos, ctx.Camera.Camera)); hit {
				if dist < bestDist {
					bestDist = dist
					bestID = o.ID
				}
			}
		}
		t.HoveredObj = bestID
	} else if obj != nil {
		if t.Mode == ModeVertex {
			bestDist := float32(20.0)
			for i, v := range obj.Mesh.Vertices {
				wp := obj.TransformPointWorld(v.Position, ctx.Objects)
				sp := rl.GetWorldToScreen(wp, ctx.Camera.Camera)
				d := rl.Vector2Distance(mousePos, sp)
				if d < bestDist {
					bestDist = d
					t.HoveredVert = i
				}
			}
		} else if t.Mode == ModeEdge {
			bestDist := float32(14.0)
			for i, e := range obj.Mesh.Edges {
				wp1 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V1].Position, ctx.Objects)
				wp2 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V2].Position, ctx.Objects)
				sp1 := rl.GetWorldToScreen(wp1, ctx.Camera.Camera)
				sp2 := rl.GetWorldToScreen(wp2, ctx.Camera.Camera)

				l2 := rl.Vector2DistanceSqr(sp1, sp2)
				var d float32
				if l2 == 0 {
					d = rl.Vector2Distance(mousePos, sp1)
				} else {
					tVal := ((mousePos.X-sp1.X)*(sp2.X-sp1.X) + (mousePos.Y-sp1.Y)*(sp2.Y-sp1.Y)) / l2
					if tVal < 0 {
						d = rl.Vector2Distance(mousePos, sp1)
					} else if tVal > 1 {
						d = rl.Vector2Distance(mousePos, sp2)
					} else {
						proj := rl.Vector2{X: sp1.X + tVal*(sp2.X-sp1.X), Y: sp1.Y + tVal*(sp2.Y-sp1.Y)}
						d = rl.Vector2Distance(mousePos, proj)
					}
				}
				if d < bestDist {
					bestDist = d
					t.HoveredEdge = i
				}
			}
		}
	}

	// Hotkeys
	if rl.IsKeyPressed(rl.KeyG) {
		if t.hasSelection(ctx) {
			t.startModal(ctx, OpGrab)
			t.showBanner("Grab: [1]=X [2]=Y [3]=Z axis. Left-Click=Apply, Right-Click=Cancel")
		} else {
			t.showBanner("Grab: No selection to move")
		}
	}

	if rl.IsKeyPressed(rl.KeyS) {
		if t.hasSelection(ctx) {
			t.startModal(ctx, OpScale)
			t.showBanner("Scale: [1]=X [2]=Y [3]=Z axis. Left-Click=Apply, Right-Click=Cancel")
		} else {
			t.showBanner("Scale: No selection to scale")
		}
	}

	if rl.IsKeyPressed(rl.KeyE) {
		if t.ViewMode == ViewObject {
			t.showBanner("Switch to Edit View to extrude faces")
		} else if obj != nil && t.Mode == ModeFace && len(t.SelectedFaces) > 0 {
			newFaces := make(map[int]bool)
			for fi := range t.SelectedFaces {
				if nfi := obj.Mesh.ExtrudeFace(fi, 0.0); nfi != -1 {
					newFaces[nfi] = true
				}
			}
			t.SelectedFaces = newFaces
			t.startModal(ctx, OpExtrude)
			t.showBanner("Extrude: Drag mouse to extend. Left-Click=Apply, Right-Click=Cancel")
		} else {
			t.showBanner("Extrude requires selecting face(s) in Face mode")
		}
	}

	if rl.IsKeyPressed(rl.KeyI) {
		if t.ViewMode == ViewObject {
			t.showBanner("Switch to Edit View to inset faces")
		} else if obj != nil && t.Mode == ModeFace && len(t.SelectedFaces) > 0 {
			newFaces := make(map[int]bool)
			for fi := range t.SelectedFaces {
				if nfi := obj.Mesh.InsetFace(fi, 0.0); nfi != -1 {
					newFaces[nfi] = true
				}
			}
			t.SelectedFaces = newFaces
			t.startModal(ctx, OpInset)
			t.showBanner("Inset: Drag mouse to scale inner face. Left-Click=Apply, Right-Click=Cancel")
		} else {
			t.showBanner("Inset requires selecting a face in Face mode")
		}
	}

	if rl.IsKeyPressed(rl.KeyC) {
		if t.ViewMode == ViewObject {
			t.showBanner("Switch to Edit View to cut faces")
		} else if obj != nil && t.Mode == ModeFace {
			t.ActiveOp = OpCut
			t.CutStep = 0
			t.showBanner("Cut Tool: Click first point on face edge/surface")
		} else {
			t.showBanner("Cut Tool: Switch to Face mode to cut faces")
		}
	}

	if rl.IsKeyPressed(rl.KeyD) || rl.IsKeyPressed(rl.KeyDelete) {
		t.deleteSelection(ctx)
	}
}

func (t *EditTool) executeBoxSelection(ctx *engine.Context, shift bool) {
	minX := float32(math.Min(float64(t.DragStartPos.X), float64(t.DragCurrentPos.X)))
	maxX := float32(math.Max(float64(t.DragStartPos.X), float64(t.DragCurrentPos.X)))
	minY := float32(math.Min(float64(t.DragStartPos.Y), float64(t.DragCurrentPos.Y)))
	maxY := float32(math.Max(float64(t.DragStartPos.Y), float64(t.DragCurrentPos.Y)))

	inRect := func(p rl.Vector2) bool {
		return p.X >= minX && p.X <= maxX && p.Y >= minY && p.Y <= maxY
	}

	viewDir := rl.Vector3Subtract(ctx.Camera.Camera.Position, ctx.Camera.Camera.Target)

	if t.Mode == ModeObject || t.ViewMode == ViewObject {
		if !shift {
			t.SelectedObjects = make(map[int]bool)
		}
		for _, o := range ctx.Objects {
			center := o.TransformPointWorld(rl.Vector3{}, ctx.Objects)
			sp := rl.GetWorldToScreen(center, ctx.Camera.Camera)
			if inRect(sp) {
				t.SelectedObjects[o.ID] = true
				ctx.SelID = o.ID
			} else {
				for _, v := range o.Mesh.Vertices {
					spV := rl.GetWorldToScreen(o.TransformPointWorld(v.Position, ctx.Objects), ctx.Camera.Camera)
					if inRect(spV) {
						t.SelectedObjects[o.ID] = true
						ctx.SelID = o.ID
						break
					}
				}
			}
		}
	} else if obj := ctx.ActiveObject(); obj != nil {
		if !shift {
			t.clearSubSelections()
		}

		if t.Mode == ModeVertex {
			for i, v := range obj.Mesh.Vertices {
				wp := obj.TransformPointWorld(v.Position, ctx.Objects)
				sp := rl.GetWorldToScreen(wp, ctx.Camera.Camera)
				if inRect(sp) {
					if t.ViewMode == ViewWireframe || rl.Vector3DotProduct(v.Normal, viewDir) > -0.2 {
						t.SelectedVerts[i] = true
					}
				}
			}
		} else if t.Mode == ModeEdge {
			for i, e := range obj.Mesh.Edges {
				wp1 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V1].Position, ctx.Objects)
				wp2 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V2].Position, ctx.Objects)
				sp1 := rl.GetWorldToScreen(wp1, ctx.Camera.Camera)
				sp2 := rl.GetWorldToScreen(wp2, ctx.Camera.Camera)
				mid := rl.Vector2Scale(rl.Vector2Add(sp1, sp2), 0.5)

				if inRect(sp1) || inRect(sp2) || inRect(mid) {
					t.SelectedEdges[i] = true
				}
			}
		} else if t.Mode == ModeFace {
			for i, f := range obj.Mesh.Faces {
				fc := obj.Mesh.GetFaceCenter(i)
				wp := obj.TransformPointWorld(fc, ctx.Objects)
				sp := rl.GetWorldToScreen(wp, ctx.Camera.Camera)
				if inRect(sp) {
					if t.ViewMode == ViewWireframe || rl.Vector3DotProduct(f.Normal, viewDir) > 0.0 {
						t.SelectedFaces[i] = true
					}
				}
			}
		}
	}
}

func (t *EditTool) deleteSelection(ctx *engine.Context) {
	if (t.Mode == ModeObject || t.ViewMode == ViewObject) && len(t.SelectedObjects) > 0 {
		for id := range t.SelectedObjects {
			for _, o := range ctx.Objects {
				if o.ID == id && o.ParentID == 0 {
					t.showBanner("Cannot delete Root object")
					return
				}
			}
			t.deleteObjectHierarchy(ctx, id)
		}
		t.SelectedObjects = make(map[int]bool)
		if len(ctx.Objects) > 0 {
			ctx.SelID = ctx.Objects[0].ID
			t.SelectedObjects[ctx.SelID] = true
		}
		ctx.History.Save(ctx.Objects, ctx.SelID)
		t.showBanner("Deleted selected objects")
	} else if obj := ctx.ActiveObject(); obj != nil {
		if t.Mode == ModeVertex && len(t.SelectedVerts) > 0 {
			for vi := range t.SelectedVerts {
				obj.Mesh.DissolveOrDeleteVertex(vi)
				break
			}
			t.SelectedVerts = make(map[int]bool)
			ctx.History.Save(ctx.Objects, ctx.SelID)
			t.showBanner("Deleted/dissolved vertex")
		} else if t.Mode == ModeEdge && len(t.SelectedEdges) > 0 {
			for ei := range t.SelectedEdges {
				obj.Mesh.DeleteEdge(ei)
				break
			}
			t.SelectedEdges = make(map[int]bool)
			ctx.History.Save(ctx.Objects, ctx.SelID)
			t.showBanner("Deleted edge")
		} else if t.Mode == ModeFace && len(t.SelectedFaces) > 0 {
			for fi := range t.SelectedFaces {
				obj.Mesh.DeleteFace(fi)
				break
			}
			t.SelectedFaces = make(map[int]bool)
			ctx.History.Save(ctx.Objects, ctx.SelID)
			t.showBanner("Deleted face")
		} else {
			t.showBanner("Nothing selected to delete")
		}
	} else {
		t.showBanner("Nothing selected to delete")
	}
}

func (t *EditTool) hasSelection(ctx *engine.Context) bool {
	if t.Mode == ModeObject || t.ViewMode == ViewObject {
		return len(t.SelectedObjects) > 0
	}
	if t.Mode == ModeVertex {
		return len(t.SelectedVerts) > 0
	}
	if t.Mode == ModeEdge {
		return len(t.SelectedEdges) > 0
	}
	if t.Mode == ModeFace {
		return len(t.SelectedFaces) > 0
	}
	return false
}

func (t *EditTool) isVertInSelectedEdges(obj *mesh.Object, vIdx int) bool {
	for ei := range t.SelectedEdges {
		if ei < len(obj.Mesh.Edges) {
			if obj.Mesh.Edges[ei].V1 == vIdx || obj.Mesh.Edges[ei].V2 == vIdx {
				return true
			}
		}
	}
	return false
}

func (t *EditTool) isVertInSelectedFaces(obj *mesh.Object, vIdx int) bool {
	for fi := range t.SelectedFaces {
		if fi < len(obj.Mesh.Faces) {
			for _, idx := range obj.Mesh.Faces[fi].Indices {
				if idx == vIdx {
					return true
				}
			}
		}
	}
	return false
}

func (t *EditTool) deleteObjectHierarchy(ctx *engine.Context, id int) {
	var remaining []mesh.Object
	for _, o := range ctx.Objects {
		if o.ID != id && o.ParentID != id {
			remaining = append(remaining, o)
		}
	}
	ctx.Objects = remaining
	if ctx.SelID == id && len(ctx.Objects) > 0 {
		ctx.SelID = ctx.Objects[0].ID
		t.SelectedObjects = map[int]bool{ctx.SelID: true}
	}
}

// drawSilhouetteOutline renders ONLY the camera-relative outer silhouette edges (inner edges hidden)
func (t *EditTool) drawSilhouetteOutline(ctx *engine.Context, o *mesh.Object, col rl.Color) {
	camPos := ctx.Camera.Camera.Position

	type EdgeFaces struct {
		Faces  []int
		V1, V2 int
	}
	edgeMap := make(map[[2]int]*EdgeFaces)

	for fi, f := range o.Mesh.Faces {
		n := len(f.Indices)
		for i := 0; i < n; i++ {
			v1, v2 := f.Indices[i], f.Indices[(i+1)%n]
			k := [2]int{v1, v2}
			if v1 > v2 {
				k = [2]int{v2, v1}
			}
			if ef, ok := edgeMap[k]; ok {
				ef.Faces = append(ef.Faces, fi)
			} else {
				edgeMap[k] = &EdgeFaces{Faces: []int{fi}, V1: v1, V2: v2}
			}
		}
	}

	originWorld := o.TransformPointWorld(rl.Vector3{}, ctx.Objects)

	for _, ef := range edgeMap {
		p1 := o.TransformPointWorld(o.Mesh.Vertices[ef.V1].Position, ctx.Objects)
		p2 := o.TransformPointWorld(o.Mesh.Vertices[ef.V2].Position, ctx.Objects)
		mid := rl.Vector3Scale(rl.Vector3Add(p1, p2), 0.5)
		viewVec := rl.Vector3Subtract(camPos, mid)

		isSilhouette := false
		if len(ef.Faces) == 1 {
			// Open boundary edge: visible if face is front-facing
			f1 := o.Mesh.Faces[ef.Faces[0]]
			norm1World := rl.Vector3Normalize(rl.Vector3Subtract(o.TransformPointWorld(f1.Normal, ctx.Objects), originWorld))
			if rl.Vector3DotProduct(norm1World, viewVec) > 0 {
				isSilhouette = true
			}
		} else if len(ef.Faces) >= 2 {
			f1 := o.Mesh.Faces[ef.Faces[0]]
			f2 := o.Mesh.Faces[ef.Faces[1]]

			norm1World := rl.Vector3Normalize(rl.Vector3Subtract(o.TransformPointWorld(f1.Normal, ctx.Objects), originWorld))
			norm2World := rl.Vector3Normalize(rl.Vector3Subtract(o.TransformPointWorld(f2.Normal, ctx.Objects), originWorld))

			dot1 := rl.Vector3DotProduct(norm1World, viewVec)
			dot2 := rl.Vector3DotProduct(norm2World, viewVec)

			// Silhouette edge: exactly one adjacent face is front-facing and the other is back-facing
			if (dot1 > 0 && dot2 <= 0) || (dot1 <= 0 && dot2 > 0) {
				isSilhouette = true
			}
		}

		if isSilhouette {
			rl.DrawLine3D(p1, p2, col)
		}
	}
}

func (t *EditTool) Draw3D(ctx *engine.Context) {
	// 1. Draw Axis Constraint Lines (Grab & Scale)
	if (t.ActiveOp == OpGrab || t.ActiveOp == OpScale) && ctx.ActiveObject() != nil {
		origin := ctx.ActiveObject().TransformPointWorld(t.ModalCenter, ctx.Objects)
		if t.ActiveAxis == AxisX {
			rl.DrawLine3D(rl.Vector3Add(origin, rl.Vector3{X: -100}), rl.Vector3Add(origin, rl.Vector3{X: 100}), rl.Red)
		} else if t.ActiveAxis == AxisY {
			rl.DrawLine3D(rl.Vector3Add(origin, rl.Vector3{Y: -100}), rl.Vector3Add(origin, rl.Vector3{Y: 100}), rl.Green)
		} else if t.ActiveAxis == AxisZ {
			rl.DrawLine3D(rl.Vector3Add(origin, rl.Vector3{Z: -100}), rl.Vector3Add(origin, rl.Vector3{Z: 100}), rl.Blue)
		}
	}

	// 2. Cut Tool Guide
	if t.ActiveOp == OpCut && t.CutStep == 1 {
		rl.DrawSphere(t.CutPoint1, 0.08, rl.Yellow)
		if ctx.RayHit {
			rl.DrawLine3D(t.CutPoint1, ctx.HitPos, rl.Orange)
			rl.DrawSphere(ctx.HitPos, 0.06, rl.Orange)
		}
	}

	// In Object View: Render ONLY silhouette/outline edges (no inner mesh lines or vertex handles)
	if t.ViewMode == ViewObject {
		// Highlight Selected Objects with clean silhouette contour
		for id := range t.SelectedObjects {
			for _, o := range ctx.Objects {
				if o.ID == id {
					col := rl.NewColor(255, 165, 30, 245) // Primary Gold/Orange Silhouette
					if o.ID != ctx.SelID {
						col = rl.NewColor(100, 200, 255, 220) // Multi-Select Cyan Silhouette
					}
					t.drawSilhouetteOutline(ctx, &o, col)
				}
			}
		}

		// Highlight Hovered Object Outline
		if t.HoveredObj != -1 && !t.SelectedObjects[t.HoveredObj] {
			for _, o := range ctx.Objects {
				if o.ID == t.HoveredObj {
					t.drawSilhouetteOutline(ctx, &o, rl.NewColor(255, 235, 90, 180)) // Soft yellow outline
				}
			}
		}
		return
	}

	// 3. Object Multi-Selection Highlighting in Edit & Wireframe Views
	if t.Mode == ModeObject {
		for id := range t.SelectedObjects {
			for _, o := range ctx.Objects {
				if o.ID == id && o.ID != ctx.SelID {
					for _, e := range o.Mesh.Edges {
						p1 := o.TransformPointWorld(o.Mesh.Vertices[e.V1].Position, ctx.Objects)
						p2 := o.TransformPointWorld(o.Mesh.Vertices[e.V2].Position, ctx.Objects)
						rl.DrawLine3D(p1, p2, rl.NewColor(100, 200, 255, 220))
					}
				}
			}
		}
	}

	// 4. Sub-Element Highlighting on Active Object
	obj := ctx.ActiveObject()
	if obj == nil {
		return
	}

	// In Wireframe View: draw all vertices as glowing points for through-selection
	if t.Mode == ModeVertex || t.ViewMode == ViewWireframe {
		for vi, v := range obj.Mesh.Vertices {
			wp := obj.TransformPointWorld(v.Position, ctx.Objects)
			col := rl.NewColor(90, 190, 255, 255)
			radius := float32(0.04)
			if t.SelectedVerts[vi] {
				col = rl.Orange
				radius = 0.07
			}
			if vi == t.HoveredVert {
				col = rl.Yellow
				radius = 0.08
			}
			rl.DrawSphere(wp, radius, col)
		}
	}

	if t.Mode == ModeEdge {
		for ei, e := range obj.Mesh.Edges {
			wp1 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V1].Position, ctx.Objects)
			wp2 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V2].Position, ctx.Objects)
			col := rl.NewColor(100, 200, 255, 180)
			if t.SelectedEdges[ei] {
				col = rl.Orange
				rl.DrawLine3D(wp1, wp2, col)
			}
			if ei == t.HoveredEdge {
				col = rl.Yellow
				rl.DrawLine3D(wp1, wp2, col)
			}
		}
	} else if t.Mode == ModeFace {
		for fi, f := range obj.Mesh.Faces {
			if len(f.Indices) < 3 {
				continue
			}
			isSel := t.SelectedFaces[fi]
			isHov := fi == t.HoveredFace
			if isSel || isHov {
				col := rl.NewColor(255, 160, 40, 140)
				if isHov {
					col = rl.NewColor(255, 230, 80, 180)
				}
				p0 := obj.TransformPointWorld(obj.Mesh.Vertices[f.Indices[0]].Position, ctx.Objects)
				for j := 1; j < len(f.Indices)-1; j++ {
					p1 := obj.TransformPointWorld(obj.Mesh.Vertices[f.Indices[j]].Position, ctx.Objects)
					p2 := obj.TransformPointWorld(obj.Mesh.Vertices[f.Indices[j+1]].Position, ctx.Objects)
					rl.DrawTriangle3D(p0, p1, p2, col)
				}
			}
		}
	}
}

func (t *EditTool) DrawUI(ctx *engine.Context) {
	if imgui.BeginMainMenuBar() {
		if imgui.BeginMenu("Window") {
			imgui.MenuItemBoolPtr("Top Bar Controls", "", &t.ShowTopBar)
			imgui.MenuItemBoolPtr("Scene Tree Hierarchy", "", &t.ShowSceneTree)
			imgui.MenuItemBoolPtr("Object Inspector", "", &t.ShowInspector)
			imgui.EndMenu()
		}
		imgui.EndMainMenuBar()
	}

	// 1. Top Bar Controls
	if t.ShowTopBar {
		imgui.SetNextWindowPosV(imgui.NewVec2(10, 35), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(560, 70), imgui.CondFirstUseEver)
		if imgui.BeginV("Editor Controls##topbar", &t.ShowTopBar, imgui.WindowFlagsNoResize) {
			modes := []string{"Select (Object)", "Vertex", "Edge", "Face"}
			curMode := int32(t.Mode)
			imgui.SetNextItemWidth(140)
			previewMode := modes[curMode]
			if imgui.BeginComboV("Mode", previewMode, 0) {
				for i, m := range modes {
					selected := i == int(curMode)
					if imgui.SelectableBoolV(m, selected, 0, imgui.NewVec2(0, 0)) {
						t.Mode = SelectMode(i)
						t.clearSubSelections()
					}
					if selected {
						imgui.SetItemDefaultFocus()
					}
				}
				imgui.EndCombo()
			}

			imgui.SameLine()
			imgui.Spacing()
			imgui.SameLine()

			views := []string{"Edit View", "Wireframe View", "Object View"}
			curView := int32(t.ViewMode)
			imgui.SetNextItemWidth(140)
			previewView := views[curView]
			if imgui.BeginComboV("View", previewView, 0) {
				for i, v := range views {
					selected := i == int(curView)
					if imgui.SelectableBoolV(v, selected, 0, imgui.NewVec2(0, 0)) {
						t.ViewMode = ViewMode(i)
					}
					if selected {
						imgui.SetItemDefaultFocus()
					}
				}
				imgui.EndCombo()
			}

			imgui.SameLine()
			imgui.TextDisabled("(Marquee: Drag LMB)")
		}
		imgui.End()
	}

	// 2. Scene Tree Hierarchy Panel with Delete Buttons
	if t.ShowSceneTree {
		imgui.SetNextWindowPosV(imgui.NewVec2(10, 115), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(300, 380), imgui.CondFirstUseEver)
		if imgui.BeginV("Scene Tree", &t.ShowSceneTree, imgui.WindowFlagsNone) {
			imgui.Text("Hierarchy (Godot-style):")
			t.drawSceneNode(ctx, 0)
			imgui.Separator()
			if o := ctx.ActiveObject(); o != nil {
				if imgui.Button("+ Child Cube") {
					t.addChildObject(ctx, o.ID, "ChildCube", mesh.Cube(1.5))
				}
				imgui.SameLine()
				if imgui.Button("+ Child Sphere") {
					t.addChildObject(ctx, o.ID, "ChildSphere", mesh.UVSphere(1.0, 10, 14))
				}
			}
		}
		imgui.End()
	}

	// 3. Object Inspector Panel
	if t.ShowInspector {
		imgui.SetNextWindowPosV(imgui.NewVec2(1050, 35), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(300, 240), imgui.CondFirstUseEver)
		if imgui.BeginV("Inspector", &t.ShowInspector, imgui.WindowFlagsNone) {
			if o := ctx.ActiveObject(); o != nil {
				nameStr := o.Name
				if imgui.InputTextWithHint("Name", "", &nameStr, 0, nil) {
					o.Name = nameStr
				}
				pos := [3]float32{o.Position.X, o.Position.Y, o.Position.Z}
				if imgui.DragFloat3("Position", &pos) {
					o.Position = rl.Vector3{X: pos[0], Y: pos[1], Z: pos[2]}
				}
				rot := [3]float32{o.Rotation.X, o.Rotation.Y, o.Rotation.Z}
				if imgui.DragFloat3("Rotation", &rot) {
					o.Rotation = rl.Vector3{X: rot[0], Y: rot[1], Z: rot[2]}
				}
				scal := [3]float32{o.Scale.X, o.Scale.Y, o.Scale.Z}
				if imgui.DragFloat3("Scale", &scal) {
					o.Scale = rl.Vector3{X: scal[0], Y: scal[1], Z: scal[2]}
				}
				col := o.Material.Color
				if imgui.ColorEdit4("Color", &col) {
					o.Material.Color = col
				}
			} else {
				imgui.TextDisabled("No active object selected")
			}
		}
		imgui.End()
	}

	// 4. Render Drag Selection Rectangle
	if t.IsDraggingBox {
		minX := int32(math.Min(float64(t.DragStartPos.X), float64(t.DragCurrentPos.X)))
		maxX := int32(math.Max(float64(t.DragStartPos.X), float64(t.DragCurrentPos.X)))
		minY := int32(math.Min(float64(t.DragStartPos.Y), float64(t.DragCurrentPos.Y)))
		maxY := int32(math.Max(float64(t.DragStartPos.Y), float64(t.DragCurrentPos.Y)))
		w, h := maxX-minX, maxY-minY

		rl.DrawRectangle(minX, minY, w, h, rl.NewColor(0, 140, 255, 45))
		rl.DrawRectangleLines(minX, minY, w, h, rl.NewColor(0, 180, 255, 220))
	}

	// 5. Viewport Status Banner
	modeNames := []string{"OBJECT", "VERTEX", "EDGE", "FACE"}
	viewNames := []string{"EDIT", "WIREFRAME (SEE-THROUGH)", "OBJECT (OUTLINE PREVIEW)"}
	h := float32(rl.GetScreenHeight())
	rl.DrawRectangle(10, int32(h)-60, 640, 32, rl.NewColor(20, 22, 28, 220))
	rl.DrawRectangleLines(10, int32(h)-60, 640, 32, rl.NewColor(80, 85, 95, 255))
	statusText := fmt.Sprintf("[%s] | MODE: [%s] | Drag: Box Select | G(Move) S(Scale) D(Del)", viewNames[t.ViewMode], modeNames[t.Mode])
	rl.DrawText(statusText, 20, int32(h)-52, 10, rl.RayWhite)

	if t.BannerTimer > 0 {
		rl.DrawRectangle(10, int32(h)-100, 640, 32, rl.NewColor(220, 70, 50, 240))
		rl.DrawText(t.BannerMsg, 20, int32(h)-92, 10, rl.White)
	}
}

func (t *EditTool) drawSceneNode(ctx *engine.Context, parentID int) {
	for i := range ctx.Objects {
		o := &ctx.Objects[i]
		if o.ParentID == parentID {
			flags := imgui.TreeNodeFlagsOpenOnArrow | imgui.TreeNodeFlagsOpenOnDoubleClick
			if t.SelectedObjects[o.ID] {
				flags |= imgui.TreeNodeFlagsSelected
			}
			if !t.hasChildren(ctx, o.ID) {
				flags |= imgui.TreeNodeFlagsLeaf
			}

			isOpen := imgui.TreeNodeExStrV(fmt.Sprintf("%s##%d", o.Name, o.ID), flags)
			if imgui.IsItemClicked() {
				shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
				if !shift {
					t.SelectedObjects = make(map[int]bool)
				}
				if t.SelectedObjects[o.ID] {
					delete(t.SelectedObjects, o.ID)
				} else {
					t.SelectedObjects[o.ID] = true
				}
				if t.SelectedObjects[o.ID] {
					ctx.SelID = o.ID
				} else if len(t.SelectedObjects) > 0 {
					for id := range t.SelectedObjects {
						ctx.SelID = id
						break
					}
				}
				t.clearSubSelections()
			}

			if o.ParentID != 0 {
				imgui.SameLine()
				if imgui.SmallButton(fmt.Sprintf("Del##%d", o.ID)) {
					t.deleteObjectHierarchy(ctx, o.ID)
					ctx.History.Save(ctx.Objects, ctx.SelID)
					t.showBanner("Deleted object from hierarchy")
					if isOpen {
						imgui.TreePop()
					}
					return
				}
			}

			if isOpen {
				t.drawSceneNode(ctx, o.ID)
				imgui.TreePop()
			}
		}
	}
}

func (t *EditTool) hasChildren(ctx *engine.Context, id int) bool {
	for _, o := range ctx.Objects {
		if o.ParentID == id {
			return true
		}
	}
	return false
}

func (t *EditTool) addChildObject(ctx *engine.Context, parentID int, name string, m mesh.Data) {
	ctx.NextID++
	newObj := mesh.Object{
		ID:       ctx.NextID,
		Name:     name,
		ParentID: parentID,
		Position: rl.Vector3{Y: 1.5},
		Scale:    rl.Vector3{X: 1, Y: 1, Z: 1},
		Visible:  true,
		Mesh:     m,
		Material: mesh.Material{Color: [4]float32{0.75, 0.78, 0.82, 1.0}},
	}
	ctx.Objects = append(ctx.Objects, newObj)
	ctx.SelID = newObj.ID
	t.SelectedObjects = map[int]bool{newObj.ID: true}
	t.clearSubSelections()
	ctx.History.Save(ctx.Objects, ctx.SelID)
	t.showBanner("Added child node to scene")
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = project.ResolveDir(os.Args[1])
	}
	engine.Launch(dir, &EditTool{})
}
