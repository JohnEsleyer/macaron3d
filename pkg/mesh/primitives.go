package mesh

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func Cube(size float32) Data {
	hs := size / 2
	verts := []Vertex{
		{Position: rl.Vector3{X: -hs, Y: -hs, Z: hs}, UV: rl.Vector2{X: 0, Y: 0}},
		{Position: rl.Vector3{X: hs, Y: -hs, Z: hs}, UV: rl.Vector2{X: 1, Y: 0}},
		{Position: rl.Vector3{X: hs, Y: hs, Z: hs}, UV: rl.Vector2{X: 1, Y: 1}},
		{Position: rl.Vector3{X: -hs, Y: hs, Z: hs}, UV: rl.Vector2{X: 0, Y: 1}},
		{Position: rl.Vector3{X: -hs, Y: -hs, Z: -hs}, UV: rl.Vector2{X: 1, Y: 0}},
		{Position: rl.Vector3{X: hs, Y: -hs, Z: -hs}, UV: rl.Vector2{X: 0, Y: 0}},
		{Position: rl.Vector3{X: hs, Y: hs, Z: -hs}, UV: rl.Vector2{X: 0, Y: 1}},
		{Position: rl.Vector3{X: -hs, Y: hs, Z: -hs}, UV: rl.Vector2{X: 1, Y: 1}},
	}
	quadUVs := []rl.Vector2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}
	faces := []Face{
		{Indices: []int{0, 1, 2, 3}, UVs: quadUVs},
		{Indices: []int{5, 4, 7, 6}, UVs: quadUVs},
		{Indices: []int{4, 0, 3, 7}, UVs: quadUVs},
		{Indices: []int{1, 5, 6, 2}, UVs: quadUVs},
		{Indices: []int{3, 2, 6, 7}, UVs: quadUVs},
		{Indices: []int{4, 5, 1, 0}, UVs: quadUVs},
	}
	m := Data{Vertices: verts, Faces: faces}
	m.RebuildEdges()
	m.RecalculateNormals()
	return m
}

func Plane(size float32) Data {
	hs := size / 2
	verts := []Vertex{
		{Position: rl.Vector3{X: -hs, Y: 0, Z: -hs}, UV: rl.Vector2{X: 0, Y: 0}},
		{Position: rl.Vector3{X: hs, Y: 0, Z: -hs}, UV: rl.Vector2{X: 1, Y: 0}},
		{Position: rl.Vector3{X: hs, Y: 0, Z: hs}, UV: rl.Vector2{X: 1, Y: 1}},
		{Position: rl.Vector3{X: -hs, Y: 0, Z: hs}, UV: rl.Vector2{X: 0, Y: 1}},
	}
	m := Data{Vertices: verts, Faces: []Face{{Indices: []int{0, 1, 2, 3}}}}
	m.RebuildEdges()
	m.RecalculateNormals()
	return m
}

func UVSphere(radius float32, rings, segments int) Data {
	if rings < 2 {
		rings = 2
	}
	if segments < 3 {
		segments = 3
	}
	var verts []Vertex
	verts = append(verts, Vertex{Position: rl.Vector3{X: 0, Y: radius, Z: 0}, UV: rl.Vector2{X: 0.5, Y: 1}})
	for r := 1; r < rings; r++ {
		phi := math.Pi * float64(r) / float64(rings)
		v := 1.0 - float32(r)/float32(rings)
		y := radius * float32(math.Cos(phi))
		ringR := radius * float32(math.Sin(phi))
		for s := 0; s < segments; s++ {
			theta := 2 * math.Pi * float64(s) / float64(segments)
			u := float32(s) / float32(segments)
			x := ringR * float32(math.Cos(theta))
			z := ringR * float32(math.Sin(theta))
			verts = append(verts, Vertex{Position: rl.Vector3{X: x, Y: y, Z: z}, UV: rl.Vector2{X: u, Y: v}})
		}
	}
	botIdx := len(verts)
	verts = append(verts, Vertex{Position: rl.Vector3{X: 0, Y: -radius, Z: 0}, UV: rl.Vector2{X: 0.5, Y: 0}})
	var faces []Face
	for s := 0; s < segments; s++ {
		next := (s + 1) % segments
		faces = append(faces, Face{Indices: []int{0, 1 + next, 1 + s}})
	}
	for r := 0; r < rings-2; r++ {
		r1 := 1 + r*segments
		r2 := 1 + (r+1)*segments
		for s := 0; s < segments; s++ {
			next := (s + 1) % segments
			faces = append(faces, Face{Indices: []int{r1 + s, r1 + next, r2 + next, r2 + s}})
		}
	}
	last := 1 + (rings-2)*segments
	for s := 0; s < segments; s++ {
		next := (s + 1) % segments
		faces = append(faces, Face{Indices: []int{botIdx, last + s, last + next}})
	}
	m := Data{Vertices: verts, Faces: faces}
	m.RebuildEdges()
	m.RecalculateNormals()
	return m
}
