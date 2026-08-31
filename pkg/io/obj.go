package io

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"macaron/pkg/mesh"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func ExportOBJ(path string, objects []mesh.Object) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "# macaron OBJ export — %d objects\n", len(objects))
	off := 1
	for _, o := range objects {
		fmt.Fprintf(w, "o %s\n", o.Name)
		for _, v := range o.Mesh.Vertices {
			p := o.TransformPoint(v.Position)
			fmt.Fprintf(w, "v %.6f %.6f %.6f\n", p.X, p.Y, p.Z)
		}
		for _, v := range o.Mesh.Vertices {
			fmt.Fprintf(w, "vt %.6f %.6f\n", v.UV.X, v.UV.Y)
		}
		for _, v := range o.Mesh.Vertices {
			fmt.Fprintf(w, "vn %.6f %.6f %.6f\n", v.Normal.X, v.Normal.Y, v.Normal.Z)
		}
		for _, face := range o.Mesh.Faces {
			fmt.Fprint(w, "f")
			for _, idx := range face.Indices {
				fmt.Fprintf(w, " %d/%d/%d", off+idx, off+idx, off+idx)
			}
			fmt.Fprintln(w)
		}
		off += len(o.Mesh.Vertices)
	}
	return w.Flush()
}

func ImportOBJ(path string) ([]mesh.Object, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var objects []mesh.Object
	var verts []rl.Vector3
	var cur mesh.Data
	name := "Imported"
	flush := func() {
		if len(cur.Vertices) > 0 && len(cur.Faces) > 0 {
			cur.RebuildEdges()
			cur.RecalculateNormals()
			objects = append(objects, mesh.Object{
				Name: name, Position: rl.Vector3{}, Scale: rl.Vector3{X: 1, Y: 1, Z: 1}, Visible: true,
				Mesh: cur, Material: mesh.Material{Color: [4]float32{0.8, 0.8, 0.8, 1}},
			})
			cur = mesh.Data{}
		}
	}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		switch parts[0] {
		case "o", "g":
			if len(cur.Faces) > 0 {
				flush()
			}
			if len(parts) > 1 {
				name = parts[1]
			}
		case "v":
			x, _ := strconv.ParseFloat(parts[1], 32)
			y, _ := strconv.ParseFloat(parts[2], 32)
			z, _ := strconv.ParseFloat(parts[3], 32)
			v := rl.Vector3{X: float32(x), Y: float32(y), Z: float32(z)}
			verts = append(verts, v)
			cur.Vertices = append(cur.Vertices, mesh.Vertex{Position: v})
		case "f":
			var idx []int
			for _, p := range parts[1:] {
				sub := strings.Split(p, "/")
				vi, _ := strconv.Atoi(sub[0])
				if vi > 0 {
					vi--
				} else {
					vi = len(verts) + vi
				}
				if vi >= 0 && vi < len(cur.Vertices) {
					idx = append(idx, vi)
				}
			}
			if len(idx) >= 3 {
				cur.Faces = append(cur.Faces, mesh.Face{Indices: idx})
			}
		}
	}
	flush()
	return objects, nil
}
