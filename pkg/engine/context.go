package engine

import (
	"macaron/pkg/camera"
	"macaron/pkg/mesh"
	"macaron/pkg/project"
	"macaron/pkg/undo"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type RenderMode int

const (
	RenderModeDefault   RenderMode = iota // Edit View: Solid lit faces + selection wireframes
	RenderModeWireframe                   // Wireframe View: Transparent see-through wireframes (Blender-style)
	RenderModeObject                      // Object View: Clean seamless rendered mesh (no lines, no handles)
	RenderModeCustom                      // Custom: Tool manages 100% of Draw3D
)

type Context struct {
	Project    *project.Project
	Camera     *camera.Viewport
	Objects    []mesh.Object
	NextID     int
	SelID      int
	RenderMode RenderMode

	History undo.Stack

	// Live raycast
	RayHit      bool
	HitPos      rl.Vector3
	HitNormal   rl.Vector3
	HoveredFace int
	HoveredVert int

	StatusMsg   string
	StatusTimer float32
}

func (c *Context) ActiveObject() *mesh.Object {
	for i := range c.Objects {
		if c.Objects[i].ID == c.SelID {
			return &c.Objects[i]
		}
	}
	return nil
}

func (c *Context) AddObject(name string, m mesh.Data) {
	c.NextID++
	o := mesh.Object{
		ID: c.NextID, Name: name, Position: rl.Vector3{Y: 1}, Scale: rl.Vector3{X: 1, Y: 1, Z: 1},
		Visible: true, Mesh: m, Material: mesh.Material{Color: [4]float32{0.82, 0.82, 0.84, 1}, Roughness: 0.4, Metallic: 0.1},
	}
	c.Objects = append(c.Objects, o)
	c.SelID = o.ID
	c.History.Save(c.Objects, c.SelID)
}

func (c *Context) SetStatus(s string) { c.StatusMsg = s; c.StatusTimer = 3 }
