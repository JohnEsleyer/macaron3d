package mesh

import rl "github.com/gen2brain/raylib-go/raylib"

type Object struct {
	ID        int
	Name      string
	ParentID  int // 0 means root or top-level
	Position  rl.Vector3
	Rotation  rl.Vector3 // Euler degrees
	Scale     rl.Vector3
	Visible   bool
	Wireframe bool
	Mesh      Data
	Material  Material
}

type Material struct {
	Color     [4]float32
	Roughness float32
	Metallic  float32
}

func (o *Object) GetLocalMatrix() rl.Matrix {
	m := rl.MatrixScale(o.Scale.X, o.Scale.Y, o.Scale.Z)
	m = rl.MatrixMultiply(m, rl.MatrixRotateXYZ(rl.Vector3{
		X: o.Rotation.X * rl.Deg2rad,
		Y: o.Rotation.Y * rl.Deg2rad,
		Z: o.Rotation.Z * rl.Deg2rad,
	}))
	m = rl.MatrixMultiply(m, rl.MatrixTranslate(o.Position.X, o.Position.Y, o.Position.Z))
	return m
}

func (o *Object) GetWorldMatrix(objects []Object) rl.Matrix {
	local := o.GetLocalMatrix()
	if o.ParentID <= 0 {
		return local
	}
	for i := range objects {
		if objects[i].ID == o.ParentID {
			parentMat := objects[i].GetWorldMatrix(objects)
			return rl.MatrixMultiply(local, parentMat)
		}
	}
	return local
}

func (o *Object) TransformPoint(p rl.Vector3) rl.Vector3 {
	return rl.Vector3Transform(p, o.GetLocalMatrix())
}

func (o *Object) TransformPointWorld(p rl.Vector3, objects []Object) rl.Vector3 {
	return rl.Vector3Transform(p, o.GetWorldMatrix(objects))
}

func (o *Object) Raycast(ray rl.Ray) (hit bool, dist float32, faceIdx int) {
	best := float32(1e9)
	hitIdx := -1
	for i, f := range o.Mesh.Faces {
		if len(f.Indices) < 3 {
			continue
		}
		p0 := o.TransformPoint(o.Mesh.Vertices[f.Indices[0]].Position)
		for j := 1; j < len(f.Indices)-1; j++ {
			p1 := o.TransformPoint(o.Mesh.Vertices[f.Indices[j]].Position)
			p2 := o.TransformPoint(o.Mesh.Vertices[f.Indices[j+1]].Position)
			c := rl.GetRayCollisionTriangle(ray, p0, p1, p2)
			if c.Hit && c.Distance < best {
				best = c.Distance
				hitIdx = i
			}
		}
	}
	if hitIdx != -1 {
		return true, best, hitIdx
	}
	return false, 0, -1
}
