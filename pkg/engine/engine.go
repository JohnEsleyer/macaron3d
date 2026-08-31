package engine

import (
	"os"
	"path/filepath"

	"macaron/pkg/camera"
	"macaron/pkg/io"
	"macaron/pkg/mesh"
	"macaron/pkg/project"

	"github.com/AllenDang/cimgui-go/backend"
	"github.com/AllenDang/cimgui-go/backend/raylibbackend"
	"github.com/AllenDang/cimgui-go/imgui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Launch — parses target dir (like `code .`), loads project, boots Raylib+ImGui, runs tool loop.
func Launch(targetDir string, tool Tool) {
	// resolve project dir (if file given, use its folder)
	if st, err := os.Stat(targetDir); err == nil && !st.IsDir() {
		targetDir = filepath.Dir(targetDir)
	}
	proj, err := project.Load(targetDir)
	if err != nil {
		// fallback: treat as plain directory — try to work without manifest
		proj = &project.Project{Root: targetDir, Manifest: project.DefaultManifest(filepath.Base(targetDir)), MeshPath: filepath.Join(targetDir, "mesh.obj")}
	}

	be := raylibbackend.NewRaylibBackend()
	backend.CreateBackend[raylibbackend.RaylibBackendFlags](be)
	be.SetConfigFlags(raylibbackend.RaylibBackendFlagsVsyncHint, raylibbackend.RaylibBackendFlagsResizable)
	be.SetBgColor(imgui.NewVec4(0.11, 0.12, 0.14, 1))
	be.CreateWindow(tool.Name()+" — Macaron", 1280, 800)

	vp := camera.New()
	ctx := &Context{Project: proj, Camera: &vp, HoveredFace: -1, HoveredVert: -1}

	// try load canonical mesh
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

		// live raycast for tool use
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
		rl.DrawGrid(32, 1)
		for i := range ctx.Objects {
			o := &ctx.Objects[i]
			if !o.Visible {
				continue
			}
			sel := o.ID == ctx.SelID
			base := rl.NewColor(uint8(o.Material.Color[0]*255), uint8(o.Material.Color[1]*255), uint8(o.Material.Color[2]*255), 255)
			for fi, f := range o.Mesh.Faces {
				if len(f.Indices) < 3 {
					continue
				}
				col := base
				if sel && fi == ctx.HoveredFace {
					col = rl.NewColor(255, 210, 110, 200)
				}
				p0 := o.TransformPoint(o.Mesh.Vertices[f.Indices[0]].Position)
				for j := 1; j < len(f.Indices)-1; j++ {
					p1 := o.TransformPoint(o.Mesh.Vertices[f.Indices[j]].Position)
					p2 := o.TransformPoint(o.Mesh.Vertices[f.Indices[j+1]].Position)
					rl.DrawTriangle3D(p0, p1, p2, col)
				}
			}
			// wireframe if selected
			if sel {
				for _, e := range o.Mesh.Edges {
					rl.DrawLine3D(o.TransformPoint(o.Mesh.Vertices[e.V1].Position), o.TransformPoint(o.Mesh.Vertices[e.V2].Position), rl.NewColor(255, 160, 0, 255))
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
			imgui.EndMainMenuBar()
		}
		tool.DrawUI(ctx)
		// status bar
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
