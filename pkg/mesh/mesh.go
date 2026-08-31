package mesh

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Vertex — single point with cached normal & UV
type Vertex struct {
	Position rl.Vector3
	Normal   rl.Vector3
	UV       rl.Vector2
	Selected bool
}

type Face struct {
	Indices  []int
	UVs      []rl.Vector2
	Normal   rl.Vector3
	Selected bool
}

type Edge struct {
	V1, V2   int
	Selected bool
}

type Data struct {
	Vertices []Vertex
	Faces    []Face
	Edges    []Edge
}

func (m *Data) RecalculateNormals() {
	for i := range m.Vertices {
		m.Vertices[i].Normal = rl.Vector3{}
	}
	for i := range m.Faces {
		f := &m.Faces[i]
		if len(f.Indices) < 3 {
			continue
		}
		p0 := m.Vertices[f.Indices[0]].Position
		p1 := m.Vertices[f.Indices[1]].Position
		p2 := m.Vertices[f.Indices[2]].Position
		norm := rl.Vector3Normalize(rl.Vector3CrossProduct(rl.Vector3Subtract(p1, p0), rl.Vector3Subtract(p2, p0)))
		f.Normal = norm
		for _, idx := range f.Indices {
			m.Vertices[idx].Normal = rl.Vector3Add(m.Vertices[idx].Normal, norm)
		}
	}
	for i := range m.Vertices {
		if rl.Vector3Length(m.Vertices[i].Normal) > 0.0001 {
			m.Vertices[i].Normal = rl.Vector3Normalize(m.Vertices[i].Normal)
		} else {
			m.Vertices[i].Normal = rl.Vector3{X: 0, Y: 1, Z: 0}
		}
	}
}

func (m *Data) RebuildEdges() {
	edgeMap := make(map[[2]int]bool)
	m.Edges = m.Edges[:0]
	for _, f := range m.Faces {
		n := len(f.Indices)
		for i := 0; i < n; i++ {
			v1, v2 := f.Indices[i], f.Indices[(i+1)%n]
			key := [2]int{v1, v2}
			if v1 > v2 {
				key = [2]int{v2, v1}
			}
			if !edgeMap[key] {
				edgeMap[key] = true
				m.Edges = append(m.Edges, Edge{V1: key[0], V2: key[1]})
			}
		}
	}
}

func (m *Data) GetFaceCenter(faceIdx int) rl.Vector3 {
	f := m.Faces[faceIdx]
	var sum rl.Vector3
	for _, idx := range f.Indices {
		sum = rl.Vector3Add(sum, m.Vertices[idx].Position)
	}
	return rl.Vector3Scale(sum, 1.0/float32(len(f.Indices)))
}

func (m *Data) GetBoundingBox() rl.BoundingBox {
	if len(m.Vertices) == 0 {
		return rl.BoundingBox{}
	}
	min, max := m.Vertices[0].Position, m.Vertices[0].Position
	for _, v := range m.Vertices {
		min.X = float32(math.Min(float64(min.X), float64(v.Position.X)))
		min.Y = float32(math.Min(float64(min.Y), float64(v.Position.Y)))
		min.Z = float32(math.Min(float64(min.Z), float64(v.Position.Z)))
		max.X = float32(math.Max(float64(max.X), float64(v.Position.X)))
		max.Y = float32(math.Max(float64(max.Y), float64(v.Position.Y)))
		max.Z = float32(math.Max(float64(max.Z), float64(v.Position.Z)))
	}
	return rl.BoundingBox{Min: min, Max: max}
}

func (m *Data) SelectAll(selected bool) {
	for i := range m.Vertices {
		m.Vertices[i].Selected = selected
	}
	for i := range m.Edges {
		m.Edges[i].Selected = selected
	}
	for i := range m.Faces {
		m.Faces[i].Selected = selected
	}
}
