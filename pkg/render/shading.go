package render

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func CheckerTexture() rl.Texture2D {
	img := rl.GenImageChecked(256, 256, 32, 32, rl.LightGray, rl.DarkGray)
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

func MatCap(normal, viewDir rl.Vector3) rl.Color {
	nx, ny := normal.X*0.5+0.5, normal.Y*0.5+0.5
	rim := float32(math.Pow(float64(1-rl.Vector3DotProduct(normal, viewDir)), 2.5))
	val := uint8(math.Min(255, float64(120+90*ny+45*rim)))
	_ = nx
	return rl.NewColor(val, val, val, 255)
}
