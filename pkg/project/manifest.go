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

// Dev helpers — `dev` shortcut resolves to playground/

func IsDevArg(s string) bool { return s == "dev" || s == "--dev" }

// DevDir returns the playground path for `dev`.
// Walks up from cwd to find playground/ sibling to the repo root; falls back to "playground".
func DevDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "playground"
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "playground")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "playground"
}

// ResolveDir maps a CLI arg to a real directory: "dev" -> playground, else passthrough.
func ResolveDir(arg string) string {
	if IsDevArg(arg) {
		return DevDir()
	}
	return arg
}
