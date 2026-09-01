package mesh

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ExtrudeFace extrudes the given face along its normal by the specified distance
// and returns the index of the newly created top face.
func (m *Data) ExtrudeFace(faceIdx int, distance float32) int {
	if faceIdx < 0 || faceIdx >= len(m.Faces) {
		return -1
	}

	origFace := m.Faces[faceIdx]
	origIndices := origFace.Indices
	numVerts := len(origIndices)
	if numVerts < 3 {
		return -1
	}

	norm := origFace.Normal
	if rl.Vector3Length(norm) < 0.0001 {
		norm = rl.Vector3{Y: 1}
	}
	offset := rl.Vector3Scale(norm, distance)

	// Create duplicate vertices for the extruded top face
	newIndices := make([]int, numVerts)
	for i, oldIdx := range origIndices {
		oldV := m.Vertices[oldIdx]
		newPos := rl.Vector3Add(oldV.Position, offset)
		newIndices[i] = len(m.Vertices)
		m.Vertices = append(m.Vertices, Vertex{
			Position: newPos,
			Normal:   oldV.Normal,
			UV:       oldV.UV,
		})
	}

	// Create side quad faces
	for i := 0; i < numVerts; i++ {
		next := (i + 1) % numVerts
		vBottom1 := origIndices[i]
		vBottom2 := origIndices[next]
		vTop2 := newIndices[next]
		vTop1 := newIndices[i]

		m.Faces = append(m.Faces, Face{
			Indices: []int{vBottom1, vBottom2, vTop2, vTop1},
			UVs: []rl.Vector2{
				{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: 0}, {X: 0, Y: 0},
			},
		})
	}

	// Replace the original face with the new top cap face
	m.Faces[faceIdx] = Face{
		Indices: newIndices,
		UVs:     origFace.UVs,
	}

	m.RebuildEdges()
	m.RecalculateNormals()
	return faceIdx
}

// InsetFace creates an inner polygon inside the face scaled down towards its center.
func (m *Data) InsetFace(faceIdx int, factor float32) int {
	if faceIdx < 0 || faceIdx >= len(m.Faces) || factor <= 0 {
		return -1
	}
	if factor >= 0.99 {
		factor = 0.99
	}

	center := m.GetFaceCenter(faceIdx)
	origFace := m.Faces[faceIdx]
	origIndices := origFace.Indices
	numVerts := len(origIndices)
	if numVerts < 3 {
		return -1
	}

	// Create inset vertices
	insetIndices := make([]int, numVerts)
	for i, oldIdx := range origIndices {
		p := m.Vertices[oldIdx].Position
		dir := rl.Vector3Subtract(center, p)
		insetPos := rl.Vector3Add(p, rl.Vector3Scale(dir, factor))

		insetIndices[i] = len(m.Vertices)
		m.Vertices = append(m.Vertices, Vertex{
			Position: insetPos,
			Normal:   origFace.Normal,
			UV:       m.Vertices[oldIdx].UV,
		})
	}

	// Create ring of faces connecting original edge to inset edge
	for i := 0; i < numVerts; i++ {
		next := (i + 1) % numVerts
		m.Faces = append(m.Faces, Face{
			Indices: []int{origIndices[i], origIndices[next], insetIndices[next], insetIndices[i]},
			UVs: []rl.Vector2{
				{X: 0, Y: 1}, {X: 1, Y: 1}, {X: 1, Y: 0}, {X: 0, Y: 0},
			},
		})
	}

	// Inset center face replaces the original
	m.Faces[faceIdx] = Face{
		Indices: insetIndices,
		UVs:     origFace.UVs,
	}

	m.RebuildEdges()
	m.RecalculateNormals()
	return faceIdx
}

// SubdivideFace splits a face (tri or quad) into smaller quads/triangles.
func (m *Data) SubdivideFace(faceIdx int) {
	if faceIdx < 0 || faceIdx >= len(m.Faces) {
		return
	}
	f := m.Faces[faceIdx]
	n := len(f.Indices)
	if n < 3 {
		return
	}

	centerPos := m.GetFaceCenter(faceIdx)
	centerIdx := len(m.Vertices)
	m.Vertices = append(m.Vertices, Vertex{
		Position: centerPos,
		Normal:   f.Normal,
		UV:       rl.Vector2{X: 0.5, Y: 0.5},
	})

	// Midpoint for each edge
	midIndices := make([]int, n)
	for i := 0; i < n; i++ {
		next := (i + 1) % n
		p1 := m.Vertices[f.Indices[i]].Position
		p2 := m.Vertices[f.Indices[next]].Position
		midPos := rl.Vector3Scale(rl.Vector3Add(p1, p2), 0.5)

		midIndices[i] = len(m.Vertices)
		m.Vertices = append(m.Vertices, Vertex{
			Position: midPos,
			Normal:   f.Normal,
		})
	}

	// Build child quads
	newFaces := make([]Face, n)
	for i := 0; i < n; i++ {
		prevMid := (i - 1 + n) % n
		newFaces[i] = Face{
			Indices: []int{f.Indices[i], midIndices[i], centerIdx, midIndices[prevMid]},
			UVs: []rl.Vector2{
				{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1},
			},
		}
	}

	m.Faces[faceIdx] = newFaces[0]
	m.Faces = append(m.Faces, newFaces[1:]...)
	m.RebuildEdges()
	m.RecalculateNormals()
}

// DeleteFace removes a face by index
func (m *Data) DeleteFace(faceIdx int) {
	if faceIdx < 0 || faceIdx >= len(m.Faces) {
		return
	}
	m.Faces = append(m.Faces[:faceIdx], m.Faces[faceIdx+1:]...)
	m.RebuildEdges()
	m.RecalculateNormals()
}

// DeleteEdge removes an edge and all adjacent faces
func (m *Data) DeleteEdge(edgeIdx int) {
	if edgeIdx < 0 || edgeIdx >= len(m.Edges) {
		return
	}
	edge := m.Edges[edgeIdx]
	var remainingFaces []Face
	for _, f := range m.Faces {
		hasEdge := false
		n := len(f.Indices)
		for i := 0; i < n; i++ {
			v1, v2 := f.Indices[i], f.Indices[(i+1)%n]
			if (v1 == edge.V1 && v2 == edge.V2) || (v1 == edge.V2 && v2 == edge.V1) {
				hasEdge = true
				break
			}
		}
		if !hasEdge {
			remainingFaces = append(remainingFaces, f)
		}
	}
	m.Faces = remainingFaces
	m.RebuildEdges()
	m.RecalculateNormals()
}

// DissolveOrDeleteVertex dissolves valence-2 vertices (joining connecting edges) or deletes the vertex and its incident faces
func (m *Data) DissolveOrDeleteVertex(vertIdx int) {
	if vertIdx < 0 || vertIdx >= len(m.Vertices) {
		return
	}

	// Find all neighboring vertices connected by edges
	neighbors := make([]int, 0)
	for _, e := range m.Edges {
		if e.V1 == vertIdx {
			neighbors = append(neighbors, e.V2)
		} else if e.V2 == vertIdx {
			neighbors = append(neighbors, e.V1)
		}
	}

	// If exactly 2 connecting edges, dissolve vertex and join edges
	if len(neighbors) == 2 {
		for fi := range m.Faces {
			f := &m.Faces[fi]
			newIndices := make([]int, 0, len(f.Indices))
			for _, idx := range f.Indices {
				if idx != vertIdx {
					newIndices = append(newIndices, idx)
				}
			}
			f.Indices = newIndices
		}
		// Clean up degenerate faces with < 3 vertices
		var validFaces []Face
		for _, f := range m.Faces {
			if len(f.Indices) >= 3 {
				validFaces = append(validFaces, f)
			}
		}
		m.Faces = validFaces
	} else {
		// General deletion: remove any faces referencing this vertex
		var validFaces []Face
		for _, f := range m.Faces {
			contains := false
			for _, idx := range f.Indices {
				if idx == vertIdx {
					contains = true
					break
				}
			}
			if !contains {
				validFaces = append(validFaces, f)
			}
		}
		m.Faces = validFaces
	}

	// Remove vertex from vertex array
	m.Vertices = append(m.Vertices[:vertIdx], m.Vertices[vertIdx+1:]...)

	// Decrement indices in faces and edges for shifted vertices
	for fi := range m.Faces {
		for ii := range m.Faces[fi].Indices {
			if m.Faces[fi].Indices[ii] > vertIdx {
				m.Faces[fi].Indices[ii]--
			}
		}
	}
	for ei := range m.Edges {
		if m.Edges[ei].V1 > vertIdx {
			m.Edges[ei].V1--
		}
		if m.Edges[ei].V2 > vertIdx {
			m.Edges[ei].V2--
		}
	}

	m.RebuildEdges()
	m.RecalculateNormals()
}

// CutFace splits a face along two 3D surface points P1 and P2
func (m *Data) CutFace(faceIdx int, p1, p2 rl.Vector3) bool {
	if faceIdx < 0 || faceIdx >= len(m.Faces) {
		return false
	}
	f := m.Faces[faceIdx]
	if len(f.Indices) < 3 {
		return false
	}

	// Insert 2 new cut vertices
	idx1 := len(m.Vertices)
	m.Vertices = append(m.Vertices, Vertex{Position: p1, Normal: f.Normal})
	idx2 := len(m.Vertices)
	m.Vertices = append(m.Vertices, Vertex{Position: p2, Normal: f.Normal})

	// Find the closest edge segments on the face to p1 and p2
	findClosestEdge := func(p rl.Vector3) int {
		bestEdge := 0
		bestDist := float32(1e9)
		for i := 0; i < len(f.Indices); i++ {
			next := (i + 1) % len(f.Indices)
			a := m.Vertices[f.Indices[i]].Position
			b := m.Vertices[f.Indices[next]].Position
			mid := rl.Vector3Scale(rl.Vector3Add(a, b), 0.5)
			d := rl.Vector3Distance(p, mid)
			if d < bestDist {
				bestDist = d
				bestEdge = i
			}
		}
		return bestEdge
	}

	e1 := findClosestEdge(p1)
	e2 := findClosestEdge(p2)
	if e1 == e2 {
		e2 = (e1 + len(f.Indices)/2) % len(f.Indices)
	}
	if e1 > e2 {
		e1, e2 = e2, e1
		idx1, idx2 = idx2, idx1
	}

	// Poly 1
	var poly1 []int
	for i := 0; i <= e1; i++ {
		poly1 = append(poly1, f.Indices[i])
	}
	poly1 = append(poly1, idx1, idx2)
	for i := e2 + 1; i < len(f.Indices); i++ {
		poly1 = append(poly1, f.Indices[i])
	}

	// Poly 2
	var poly2 []int
	poly2 = append(poly2, idx1)
	for i := e1 + 1; i <= e2; i++ {
		poly2 = append(poly2, f.Indices[i])
	}
	poly2 = append(poly2, idx2)

	if len(poly1) >= 3 && len(poly2) >= 3 {
		m.Faces[faceIdx] = Face{Indices: poly1}
		m.Faces = append(m.Faces, Face{Indices: poly2})
		m.RebuildEdges()
		m.RecalculateNormals()
		return true
	}
	return false
}

// CalculateAutoSmoothNormals groups adjacent face normals if angle <= thresholdAngleDeg.
func (m *Data) CalculateAutoSmoothNormals(thresholdAngleDeg float32) {
	cosThresh := float32(math.Cos(float64(thresholdAngleDeg * rl.Deg2rad)))
	for i := range m.Faces {
		f := &m.Faces[i]
		if len(f.Indices) < 3 {
			continue
		}
		p0 := m.Vertices[f.Indices[0]].Position
		p1 := m.Vertices[f.Indices[1]].Position
		p2 := m.Vertices[f.Indices[2]].Position
		f.Normal = rl.Vector3Normalize(rl.Vector3CrossProduct(rl.Vector3Subtract(p1, p0), rl.Vector3Subtract(p2, p0)))
	}
	for i := range m.Vertices {
		m.Vertices[i].Normal = rl.Vector3{}
	}
	for _, f := range m.Faces {
		for _, idx := range f.Indices {
			vertNorm := m.Vertices[idx].Normal
			if rl.Vector3Length(vertNorm) < 0.001 {
				m.Vertices[idx].Normal = f.Normal
			} else {
				dot := rl.Vector3DotProduct(rl.Vector3Normalize(vertNorm), f.Normal)
				if dot >= cosThresh {
					m.Vertices[idx].Normal = rl.Vector3Add(m.Vertices[idx].Normal, f.Normal)
				}
			}
		}
	}
	for i := range m.Vertices {
		if rl.Vector3Length(m.Vertices[i].Normal) > 0.0001 {
			m.Vertices[i].Normal = rl.Vector3Normalize(m.Vertices[i].Normal)
		} else {
			m.Vertices[i].Normal = rl.Vector3{Y: 1}
		}
	}
}

// AutoGeneratePlanarUVs projects UVs based on face orientation
func (m *Data) AutoGeneratePlanarUVs() {
	for fi := range m.Faces {
		f := &m.Faces[fi]
		norm := f.Normal
		f.UVs = make([]rl.Vector2, len(f.Indices))

		absX := float32(math.Abs(float64(norm.X)))
		absY := float32(math.Abs(float64(norm.Y)))
		absZ := float32(math.Abs(float64(norm.Z)))

		for vi, idx := range f.Indices {
			p := m.Vertices[idx].Position
			var u, v float32
			if absX > absY && absX > absZ {
				u = p.Z
				v = p.Y
			} else if absY > absX && absY > absZ {
				u = p.X
				v = p.Z
			} else {
				u = p.X
				v = p.Y
			}
			u = u*0.5 + 0.5
			v = 1.0 - (v*0.5 + 0.5)
			f.UVs[vi] = rl.Vector2{X: u, Y: v}
			m.Vertices[idx].UV = f.UVs[vi]
		}
	}
}
