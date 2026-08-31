package main

import (
	"fmt"
	"os"
	"path/filepath"

	"macaron/pkg/project"
)

func usage() {
	fmt.Println(`macaron — project manager

Usage:
  macaron init <name>        Create new project directory
  macaron status [path]      Show project info
  macaron doctor [path]      Validate project

Examples:
  macaron init hero-character
  macaron init ./knight-shield
  macaron status .
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
			dir = os.Args[2]
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
			dir = os.Args[2]
		}
		p, err := project.Load(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "doctor: not a macaron project")
			os.Exit(1)
		}
		ok := true
		if _, err := os.Stat(filepath.Join(p.Root, "macaron.json")); err != nil {
			fmt.Println("✗ macaron.json missing"); ok = false
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
	default:
		usage()
		os.Exit(1)
	}
}
