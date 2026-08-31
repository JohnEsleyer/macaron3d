package undo

import "macaron/pkg/mesh"

type State struct {
	Objects []mesh.Object
	SelID   int
}

type Stack struct {
	States []State
	Index  int
}

func (s *Stack) Save(objs []mesh.Object, sel int) {
	if s.Index < len(s.States)-1 {
		s.States = s.States[:s.Index+1]
	}
	cp := make([]mesh.Object, len(objs))
	for i, o := range objs {
		cp[i] = o
		cp[i].Mesh.Vertices = append([]mesh.Vertex(nil), o.Mesh.Vertices...)
		cp[i].Mesh.Faces = make([]mesh.Face, len(o.Mesh.Faces))
		for fi, f := range o.Mesh.Faces {
			cp[i].Mesh.Faces[fi] = f
			cp[i].Mesh.Faces[fi].Indices = append([]int(nil), f.Indices...)
		}
		cp[i].Mesh.Edges = append([]mesh.Edge(nil), o.Mesh.Edges...)
	}
	s.States = append(s.States, State{Objects: cp, SelID: sel})
	s.Index = len(s.States) - 1
}

func (s *Stack) Undo() (State, bool) {
	if s.Index > 0 {
		s.Index--
		return s.States[s.Index], true
	}
	return State{}, false
}

func (s *Stack) Redo() (State, bool) {
	if s.Index < len(s.States)-1 {
		s.Index++
		return s.States[s.Index], true
	}
	return State{}, false
}
