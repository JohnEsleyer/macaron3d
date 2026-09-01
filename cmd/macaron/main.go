package main

import (
	"fmt"
	"os"
	"path/filepath"

	"macaron/pkg/project"
)

func usage() {
	fmt.Print(`macaron — project manager

Usage:
  macaron init <name>        Create new project directory
  macaron status [path|dev]  Show project info (dev = ./playground)
  macaron doctor [path|dev]  Validate project
  macaron dev                Shortcut to status for playground

Tools:
  macaron-edit               Scene hierarchy & sub-element 3D polygon editor
  macaron-ps1                PS1 low-poly doll modeler & 256x256 smudge painter
  macaron-model              Low-poly blockout
  macaron-sculpt             Digital clay & voxel sculpting
  macaron-uv                 Palette & UV unwrapping
  macaron-rig                Bone & poser
  macaron-pixel              3D → 2D sprite exporter

Examples:
  macaron init hero-character
  macaron init ./knight-shield
  macaron status dev         # uses playground/
  macaron-ps1 dev            # launch PS1 modeler
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "init":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "missing project name: macaron init <name>")
			os.Exit(1)
		}
		name := os.Args[2]
		dir := name
		if filepath.Ext(name) != "" || filepath.IsAbs(name) || name == "." {
			dir = name
			name = filepath.Base(dir)
		}
		p, err := project.Init(dir, name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "init failed:", err)
			os.Exit(1)
		}
		fmt.Printf("Created project at %s\n", p.Root)
		fmt.Println("  macaron.json")
		fmt.Println("  mesh.obj (canonical geometry — commit me)")
		fmt.Println("  textures/  references/  .macaron/ (gitignored)")
		fmt.Printf("\nNext:\n  cd %s && macaron-model .\n", dir)
	case "status":
		dir := "."
		if len(os.Args) >= 3 {
			dir = project.ResolveDir(os.Args[2])
		}
		p, err := project.Load(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "not a macaron project:", err)
			os.Exit(1)
		}
		fmt.Printf("Project: %s (%s)\n", p.Manifest.Name, p.Root)
		fmt.Printf("Mesh:    %s\n", p.Manifest.Mesh)
		if _, err := os.Stat(p.MeshPath); err == nil {
			fmt.Println("Status:  mesh.obj present")
		} else {
			fmt.Println("Status:  mesh.obj missing (empty project)")
		}
	case "doctor":
		dir := "."
		if len(os.Args) >= 3 {
			dir = project.ResolveDir(os.Args[2])
		}
		p, err := project.Load(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "doctor: not a macaron project")
			os.Exit(1)
		}
		ok := true
		if _, err := os.Stat(filepath.Join(p.Root, "macaron.json")); err != nil {
			fmt.Println("✗ macaron.json missing")
			ok = false
		} else {
			fmt.Println("✓ macaron.json")
		}
		if _, err := os.Stat(p.MeshPath); err != nil {
			fmt.Println("· mesh.obj missing (ok for new project)")
		} else {
			fmt.Println("✓", p.Manifest.Mesh)
		}
		if ok {
			fmt.Println("doctor: ok")
		}
	case "dev":
		// shortcut: macaron dev -> status for playground/
		dir := project.DevDir()
		p, err := project.Load(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "dev: playground not found at", dir, "-", err)
			fmt.Fprintln(os.Stderr, "run: macaron init playground")
			os.Exit(1)
		}
		fmt.Printf("Project: %s (%s) [dev]\n", p.Manifest.Name, p.Root)
		fmt.Printf("Mesh:    %s\n", p.Manifest.Mesh)
		if _, err := os.Stat(p.MeshPath); err == nil {
			fmt.Println("Status:  mesh.obj present")
		} else {
			fmt.Println("Status:  mesh.obj missing (empty project)")
		}
	default:
		usage()
		os.Exit(1)
	}
}
