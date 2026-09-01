package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"macaron/pkg/camera"
	"macaron/pkg/io"
	"macaron/pkg/mesh"
	"macaron/pkg/project"
	"macaron/pkg/render"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/raylibbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var showAboutDialog bool = false

func Launch(targetDir string, tool Tool) {
	if st, err := os.Stat(targetDir); err == nil && !st.IsDir() {
		targetDir = filepath.Dir(targetDir)
	}
	proj, err := project.Load(targetDir)
	if err != nil {
		proj = &project.Project{
			Root:     targetDir,
			Manifest: project.DefaultManifest(filepath.Base(targetDir)),
			MeshPath: filepath.Join(targetDir, "mesh.obj"),
		}
	}

	be := raylibbackend.NewRaylibBackend()
	backend.CreateBackend[raylibbackend.RaylibBackendFlags](be)

	be.SetConfigFlags(
		raylibbackend.RaylibBackendFlagsMSAA4X,
		raylibbackend.RaylibBackendFlagsVsyncHint,
		raylibbackend.RaylibBackendFlagsResizable,
		raylibbackend.RaylibBackendFlagsHighDPI,
	)
	be.SetBgColor(imgui.NewVec4(0.13, 0.14, 0.17, 1.0))
	be.CreateWindow(tool.Name()+" — Macaron", 1360, 840)

	vp := camera.New()
	ctx := &Context{Project: proj, Camera: &vp, HoveredFace: -1, HoveredVert: -1}

	if _, err := os.Stat(proj.MeshPath); err == nil {
		if objs, err := io.ImportOBJ(proj.MeshPath); err == nil {
			for _, o := range objs {
				ctx.NextID++
				o.ID = ctx.NextID
				ctx.Objects = append(ctx.Objects, o)
			}
			if len(ctx.Objects) > 0 {
				ctx.SelID = ctx.Objects[0].ID
			}
		}
	}
	if len(ctx.Objects) == 0 {
		ctx.AddObject("Cube", mesh.Cube(2))
	}

	_ = tool.Init(ctx)

	be.SetBeforeImGuiRenderHook(func() {
		dt := rl.GetFrameTime()
		if ctx.StatusTimer > 0 {
			ctx.StatusTimer -= dt
		}
		allow := !imgui.CurrentIO().WantCaptureMouse() && !imgui.CurrentIO().WantCaptureKeyboard()
		vp.HandleInput(allow)
		tool.Update(ctx, dt)

		ctx.RayHit = false
		ctx.HoveredFace = -1
		if obj := ctx.ActiveObject(); obj != nil && allow {
			ray := rl.GetMouseRay(rl.GetMousePosition(), vp.Camera)
			if hit, dist, fi := obj.Raycast(ray); hit {
				ctx.RayHit = true
				ctx.HitPos = rl.Vector3Add(ray.Position, rl.Vector3Scale(ray.Direction, dist))
				ctx.HitNormal = obj.Mesh.Faces[fi].Normal
				ctx.HoveredFace = fi
			}
		}

		rl.BeginMode3D(vp.Camera)
		render.DrawStudioGrid(32, 1.0)

		viewDir := rl.Vector3Subtract(vp.Camera.Position, vp.Camera.Target)
		for i := range ctx.Objects {
			o := &ctx.Objects[i]
			if !o.Visible {
				continue
			}
			sel := o.ID == ctx.SelID

			for fi, f := range o.Mesh.Faces {
				if len(f.Indices) < 3 {
					continue
				}
				isHov := sel && fi == ctx.HoveredFace
				col := render.StudioLighting(o.Material.Color, f.Normal, viewDir, isHov, sel)
				p0 := o.TransformPointWorld(o.Mesh.Vertices[f.Indices[0]].Position, ctx.Objects)
				for j := 1; j < len(f.Indices)-1; j++ {
					p1 := o.TransformPointWorld(o.Mesh.Vertices[f.Indices[j]].Position, ctx.Objects)
					p2 := o.TransformPointWorld(o.Mesh.Vertices[f.Indices[j+1]].Position, ctx.Objects)
					rl.DrawTriangle3D(p0, p1, p2, col)
				}
			}

			if sel {
				for _, e := range o.Mesh.Edges {
					p1 := o.TransformPointWorld(o.Mesh.Vertices[e.V1].Position, ctx.Objects)
					p2 := o.TransformPointWorld(o.Mesh.Vertices[e.V2].Position, ctx.Objects)
					rl.DrawLine3D(p1, p2, rl.NewColor(255, 175, 40, 240))
				}
			} else {
				for _, e := range o.Mesh.Edges {
					p1 := o.TransformPointWorld(o.Mesh.Vertices[e.V1].Position, ctx.Objects)
					p2 := o.TransformPointWorld(o.Mesh.Vertices[e.V2].Position, ctx.Objects)
					rl.DrawLine3D(p1, p2, rl.NewColor(30, 32, 38, 90))
				}
			}
		}

		tool.Draw3D(ctx)
		rl.EndMode3D()
	})

	be.Run(func() {
		imgui.ClearSizeCallbackPool()
		if imgui.BeginMainMenuBar() {
			if imgui.BeginMenu("File") {
				if imgui.MenuItemBool("Save (mesh.obj)") {
					_ = io.ExportOBJ(proj.MeshPath, ctx.Objects)
					_ = tool.OnSave(ctx)
					ctx.SetStatus("Saved " + proj.MeshPath)
				}
				imgui.EndMenu()
			}
			if imgui.BeginMenu("View") {
				for _, v := range []string{"Front", "Back", "Right", "Left", "Top", "Bottom", "Iso"} {
					vv := v
					if imgui.MenuItemBool(vv) {
						vp.Snap(vv, true)
					}
				}
				if imgui.MenuItemBool("Toggle Ortho (Numpad 5)") {
					vp.ToggleOrtho()
				}
				imgui.EndMenu()
			}
			if imgui.BeginMenu("Help") {
				if imgui.MenuItemBool("About " + tool.Name()) {
					showAboutDialog = true
				}
				imgui.EndMenu()
			}
			imgui.EndMainMenuBar()
		}

		// Render Standardized About Dialog
		if showAboutDialog {
			imgui.SetNextWindowSizeV(imgui.NewVec2(520, 420), imgui.CondFirstUseEver)
			if imgui.BeginV("About — "+tool.Name(), &showAboutDialog, imgui.WindowFlagsNoCollapse) {
				imgui.Text(tool.Name())
				imgui.TextDisabled(tool.Description())
				imgui.Separator()

				imgui.TextWrapped("Note: Macaron micro-tools are specialized. Key bindings (such as 1, 2, 3, G, S, E) are context-dependent and optimized for this tool's workflow.")
				imgui.Spacing()

				imgui.Text("Tool Shortcuts & Operations:")
				imgui.Separator()
				for _, sc := range tool.Shortcuts() {
					imgui.BulletText(fmt.Sprintf("%-12s : %s", sc.Key, sc.Description))
				}

				imgui.Spacing()
				imgui.Text("Standard Viewport Controls:")
				imgui.Separator()
				imgui.BulletText("MMB          : Orbit Camera")
				imgui.BulletText("Shift + MMB  : Pan Camera")
				imgui.BulletText("Wheel        : Zoom")
				imgui.BulletText("Numpad 1/3/7 : Front / Right / Top View")
				imgui.BulletText("Numpad 5     : Toggle Orthographic / Perspective")
			}
			imgui.End()
		}

		tool.DrawUI(ctx)

		// Status Bar
		io2 := imgui.CurrentIO()
		msg := tool.Name() + " — " + tool.Description()
		if ctx.StatusTimer > 0 {
			msg = ctx.StatusMsg + " | " + msg
		}
		imgui.SetNextWindowPosV(imgui.NewVec2(0, io2.DisplaySize().Y-20), imgui.CondAlways, imgui.NewVec2(0, 0))
		imgui.SetNextWindowSizeV(imgui.NewVec2(io2.DisplaySize().X, 20), imgui.CondAlways)
		if imgui.BeginV("##status", nil, imgui.WindowFlagsNoTitleBar|imgui.WindowFlagsNoResize|imgui.WindowFlagsNoMove) {
			imgui.Text(msg)
		}
		imgui.End()
	})
}
