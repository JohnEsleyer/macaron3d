package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Created string `json:"created"`
	Mesh    string `json:"mesh"` // canonical geometry file, default mesh.obj
}

func DefaultManifest(name string) Manifest {
	return Manifest{Name: name, Version: "0.1.0", Mesh: "mesh.obj"}
}

// Project — loaded working directory
type Project struct {
	Root     string
	Manifest Manifest
	MeshPath string
}

func Init(dir, name string) (*Project, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	m := DefaultManifest(name)
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "macaron.json"), data, 0644); err != nil {
		return nil, err
	}
	for _, sub := range []string{"textures", "references", ".macaron"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0755)
	}
	_ = os.WriteFile(filepath.Join(dir, ".macaron", ".gitignore"), []byte("*\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".macaron/\n"), 0644)
	return Load(dir)
}

func Load(dir string) (*Project, error) {
	abs, _ := filepath.Abs(dir)
	data, err := os.ReadFile(filepath.Join(abs, "macaron.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &Project{Root: abs, Manifest: m, MeshPath: filepath.Join(abs, m.Mesh)}, nil
}

func (p *Project) Save() error {
	data, _ := json.MarshalIndent(p.Manifest, "", "  ")
	return os.WriteFile(filepath.Join(p.Root, "macaron.json"), data, 0644)
}
