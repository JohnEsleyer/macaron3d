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
	Mode SelectMode

	// Window Visibility Flags
	ShowSelectionBar bool
	ShowSceneTree    bool
	ShowInspector    bool

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
		{Key: "G", Description: "Grab / Move selection in 3D space"},
		{Key: "S", Description: "Scale selection around pivot"},
		{Key: "E", Description: "Extrude selected faces along normal"},
		{Key: "I", Description: "Inset selected faces inward"},
		{Key: "C", Description: "Cut face with precision mouse guide"},
		{Key: "D", Description: "Delete selection (dissolves valence-2 vertices)"},
		{Key: "1 / 2 / 3", Description: "Lock Grab/Scale to X (1), Y (2), or Z (3) axis"},
		{Key: "Shift+Click", Description: "Multi-select / toggle objects, vertices, edges, or faces"},
		{Key: "Left-Click", Description: "Apply modal changes / confirm cut"},
		{Key: "Right-Click", Description: "Cancel modal operation"},
	}
}

func (t *EditTool) Init(ctx *engine.Context) error {
	t.Mode = ModeObject
	t.ShowSelectionBar = true
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

	if t.Mode == ModeObject {
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
	if t.Mode == ModeObject {
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
	if t.BannerTimer > 0 {
		t.BannerTimer -= dt
	}

	allowUI := !imgui.CurrentIO().WantCaptureMouse() && !imgui.CurrentIO().WantCaptureKeyboard()
	obj := ctx.ActiveObject()

	// In-Progress Modal Operations
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

			if t.Mode == ModeObject {
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

			if t.Mode == ModeObject {
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
			if obj != nil {
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
			if obj != nil {
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

	// Cut Modal
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

	// Hover Detection
	t.HoveredVert = -1
	t.HoveredEdge = -1
	t.HoveredFace = ctx.HoveredFace
	t.HoveredObj = -1

	mousePos := rl.GetMousePosition()
	if t.Mode == ModeObject {
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

	// Multi-Selection Click Handlers (Shift+Click to Toggle)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		if t.Mode == ModeObject {
			if t.HoveredObj != -1 {
				if !shift {
					t.SelectedObjects = make(map[int]bool)
				}
				if t.SelectedObjects[t.HoveredObj] {
					delete(t.SelectedObjects, t.HoveredObj)
					// pick another selected as active if we deselected current
					if t.SelectedObjects[ctx.SelID] == false && len(t.SelectedObjects) > 0 {
						for id := range t.SelectedObjects {
							ctx.SelID = id
							break
						}
					}
				} else {
					t.SelectedObjects[t.HoveredObj] = true
					ctx.SelID = t.HoveredObj
				}
				// ensure at least one selection if not shift and deselected all?
				if len(t.SelectedObjects) == 0 && !shift {
					t.SelectedObjects[t.HoveredObj] = true
					ctx.SelID = t.HoveredObj
				}
			} else if !shift {
				// click empty space clears? keep at least root
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
		if obj != nil && t.Mode == ModeFace && len(t.SelectedFaces) > 0 {
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
		if obj != nil && t.Mode == ModeFace && len(t.SelectedFaces) > 0 {
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
		if obj != nil && t.Mode == ModeFace {
			t.ActiveOp = OpCut
			t.CutStep = 0
			t.showBanner("Cut Tool: Click first point on face edge/surface")
		} else {
			t.showBanner("Cut Tool: Switch to Face mode to cut faces")
		}
	}

	if rl.IsKeyPressed(rl.KeyD) {
		if t.Mode == ModeObject && len(t.SelectedObjects) > 0 {
			hasRoot := false
			for id := range t.SelectedObjects {
				for _, o := range ctx.Objects {
					if o.ID == id && o.ParentID == 0 {
						hasRoot = true
						break
					}
				}
			}
			if hasRoot {
				t.showBanner("Cannot delete Root object")
				return
			}
			for id := range t.SelectedObjects {
				t.deleteObjectHierarchy(ctx, id)
			}
			t.SelectedObjects = make(map[int]bool)
			if len(ctx.Objects) > 0 {
				ctx.SelID = ctx.Objects[0].ID
				t.SelectedObjects[ctx.SelID] = true
			}
			ctx.History.Save(ctx.Objects, ctx.SelID)
			t.showBanner("Deleted selected objects")
		} else if obj != nil {
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
}

func (t *EditTool) hasSelection(ctx *engine.Context) bool {
	if t.Mode == ModeObject {
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
}

func (t *EditTool) Draw3D(ctx *engine.Context) {
	// 1. Draw Axis Constraint Lines
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

	// 3. Object Multi-Selection Highlighting
	if t.Mode == ModeObject {
		// Highlight all multi-selected objects
		for id := range t.SelectedObjects {
			for _, o := range ctx.Objects {
				if o.ID == id {
					// Determine if this is active vs secondary selection
					isActive := o.ID == ctx.SelID
					isHovered := o.ID == t.HoveredObj
					// Skip active object's wireframe is drawn by engine; we draw extra highlight
					if isActive && !isHovered {
						continue
					}
					col := rl.NewColor(100, 200, 255, 220)
					if isHovered {
						col = rl.NewColor(255, 230, 80, 200)
					} else if o.ID == ctx.SelID {
						col = rl.NewColor(255, 175, 40, 240)
					}
					// Draw object faces with subtle overlay and wireframe
					for _, f := range o.Mesh.Faces {
						if len(f.Indices) < 3 {
							continue
						}
						// Use world transform for proper hierarchy
						p0 := o.TransformPointWorld(o.Mesh.Vertices[f.Indices[0]].Position, ctx.Objects)
						for j := 1; j < len(f.Indices)-1; j++ {
							p1 := o.TransformPointWorld(o.Mesh.Vertices[f.Indices[j]].Position, ctx.Objects)
							p2 := o.TransformPointWorld(o.Mesh.Vertices[f.Indices[j+1]].Position, ctx.Objects)
							// Subtle face overlay for object selection
							if t.SelectedObjects[o.ID] {
								overlay := rl.NewColor(80, 160, 255, 60)
								if isHovered {
									overlay = rl.NewColor(255, 230, 80, 80)
								}
								rl.DrawTriangle3D(p0, p1, p2, overlay)
							}
							_ = p1
							_ = p2
						}
					}
					for _, e := range o.Mesh.Edges {
						p1 := o.TransformPointWorld(o.Mesh.Vertices[e.V1].Position, ctx.Objects)
						p2 := o.TransformPointWorld(o.Mesh.Vertices[e.V2].Position, ctx.Objects)
						rl.DrawLine3D(p1, p2, col)
					}
					// Draw bounding sphere for hover indication
					if isHovered {
						center := o.TransformPointWorld(o.Mesh.GetFaceCenter(0), ctx.Objects)
						rl.DrawSphereWires(center, 0.1, 6, 6, col)
					}
				}
			}
		}
		// Also handle hovered but not yet selected
		if t.HoveredObj != -1 && !t.SelectedObjects[t.HoveredObj] {
			for _, o := range ctx.Objects {
				if o.ID == t.HoveredObj {
					// Find bounding
					for _, e := range o.Mesh.Edges {
						p1 := o.TransformPointWorld(o.Mesh.Vertices[e.V1].Position, ctx.Objects)
						p2 := o.TransformPointWorld(o.Mesh.Vertices[e.V2].Position, ctx.Objects)
						rl.DrawLine3D(p1, p2, rl.NewColor(255, 255, 120, 180))
					}
				}
			}
		}
		return
	}

	// 4. Sub-Element Highlighting on Active Object
	obj := ctx.ActiveObject()
	if obj == nil {
		return
	}

	if t.Mode == ModeVertex {
		for vi, v := range obj.Mesh.Vertices {
			wp := obj.TransformPointWorld(v.Position, ctx.Objects)
			col := rl.NewColor(80, 180, 255, 255)
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
	} else if t.Mode == ModeEdge {
		for ei, e := range obj.Mesh.Edges {
			wp1 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V1].Position, ctx.Objects)
			wp2 := obj.TransformPointWorld(obj.Mesh.Vertices[e.V2].Position, ctx.Objects)
			// base faint
			rl.DrawLine3D(wp1, wp2, rl.NewColor(80, 90, 110, 60))
			if t.SelectedEdges[ei] {
				rl.DrawLine3D(wp1, wp2, rl.Orange)
			}
			if ei == t.HoveredEdge {
				rl.DrawLine3D(wp1, wp2, rl.Yellow)
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
	// Inject Window menu into main bar via a secondary bar - we use a workaround:
	// Draw a small bar that hosts Window toggles; engine already drew File/View/Help.
	// Instead we overlay a second bar segment for Window.
	// To avoid duplicate MainMenuBar, we just render Window controls as a top bar window.
	// However spec expects Window menu in MainMenuBar; we emulate by adding a floating Window menu.

	// We will create a Window menu by reopening MainMenuBar if not already visible.
	// The idiomatic way is to draw it in a separate window that mimics menu; but we follow spec:
	if imgui.BeginMainMenuBar() {
		if imgui.BeginMenu("Window") {
			// Use MenuItemBoolPtr with 3 args (no shortcut)
			imgui.MenuItemBoolPtr("Selection Mode Bar", "", &t.ShowSelectionBar)
			imgui.MenuItemBoolPtr("Scene Tree Hierarchy", "", &t.ShowSceneTree)
			imgui.MenuItemBoolPtr("Object Inspector", "", &t.ShowInspector)
			imgui.EndMenu()
		}
		imgui.EndMainMenuBar()
	}

	// 1. Selection Mode Bar
	if t.ShowSelectionBar {
		imgui.SetNextWindowPosV(imgui.NewVec2(10, 35), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(440, 70), imgui.CondFirstUseEver)
		if imgui.BeginV("Selection Mode##topbar", &t.ShowSelectionBar, imgui.WindowFlagsNoResize) {
			modes := []string{"Select (Object)", "Vertex", "Edge", "Face"}
			current := int32(t.Mode)
			preview := modes[current]
			if imgui.BeginComboV("Mode", preview, 0) {
				for i, m := range modes {
					selected := i == int(current)
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
			imgui.TextDisabled("(Shift+Click: Multi-select)")
		}
		imgui.End()
	}

	// 2. Scene Tree Hierarchy Panel
	if t.ShowSceneTree {
		imgui.SetNextWindowPosV(imgui.NewVec2(10, 115), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(280, 350), imgui.CondFirstUseEver)
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
		imgui.SetNextWindowPosV(imgui.NewVec2(1060, 35), imgui.CondFirstUseEver, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(290, 240), imgui.CondFirstUseEver)
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

	// 2D Viewport Overlay Status Banner
	modeNames := []string{"SELECT (OBJECT)", "VERTEX", "EDGE", "FACE"}
	h := float32(rl.GetScreenHeight())
	rl.DrawRectangle(10, int32(h)-60, 560, 32, rl.NewColor(20, 22, 28, 220))
	rl.DrawRectangleLines(10, int32(h)-60, 560, 32, rl.NewColor(80, 85, 95, 255))
	statusText := fmt.Sprintf("MODE: [%s] | Multi: Shift+Click | G(Move) S(Scale) E(Extrude) I(Inset) C(Cut) D(Del)", modeNames[t.Mode])
	rl.DrawText(statusText, 20, int32(h)-52, 10, rl.RayWhite)

	if t.BannerTimer > 0 {
		rl.DrawRectangle(10, int32(h)-100, 560, 32, rl.NewColor(220, 70, 50, 240))
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
				// Update SelID to last clicked
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
