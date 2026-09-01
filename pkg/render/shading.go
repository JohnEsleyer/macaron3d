package render

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// StudioLighting calculates a high-visibility studio lit color using Key + Fill + Ambient + Rim lighting.
func StudioLighting(baseColor [4]float32, normal rl.Vector3, viewDir rl.Vector3, isHovered, isSelected bool) rl.Color {
	// Normalize vectors
	nLen := rl.Vector3Length(normal)
	if nLen > 0.0001 {
		normal = rl.Vector3Scale(normal, 1.0/nLen)
	} else {
		normal = rl.Vector3{Y: 1}
	}

	vLen := rl.Vector3Length(viewDir)
	if vLen > 0.0001 {
		viewDir = rl.Vector3Scale(viewDir, 1.0/vLen)
	} else {
		viewDir = rl.Vector3{Z: 1}
	}

	// 1. Key Light (Top-Right-Front, warm white)
	keyDir := rl.Vector3Normalize(rl.Vector3{X: 0.45, Y: 0.85, Z: 0.55})
	keyDot := float32(math.Max(0.0, float64(rl.Vector3DotProduct(normal, keyDir))))
	keyIntensity := keyDot * 0.55

	// 2. Fill Light (Bottom-Left-Back, soft cool blue)
	fillDir := rl.Vector3Normalize(rl.Vector3{X: -0.5, Y: -0.3, Z: -0.6})
	fillDot := float32(math.Max(0.0, float64(rl.Vector3DotProduct(normal, fillDir))))
	fillIntensity := fillDot * 0.20

	// 3. Ambient Floor (Ensures rear/underside faces are clearly visible)
	ambient := float32(0.38)

	// 4. Fresnel Rim Light (Enhances silhouette contours against dark viewport)
	fresnel := float32(math.Pow(math.Max(0.0, float64(1.0-rl.Vector3DotProduct(normal, viewDir))), 2.5)) * 0.22

	totalLight := ambient + keyIntensity + fillIntensity + fresnel
	if totalLight > 1.25 {
		totalLight = 1.25
	}

	r := baseColor[0] * totalLight
	g := baseColor[1] * totalLight
	b := baseColor[2] * totalLight

	if isHovered {
		// Amber / Gold hover glow
		r = r*0.4 + 1.0*0.6
		g = g*0.4 + 0.85*0.6
		b = b*0.4 + 0.35*0.6
	} else if isSelected {
		// Subtle warm selection tint
		r = r*0.85 + 1.0*0.15
		g = g*0.85 + 0.65*0.15
		b = b*0.85 + 0.25*0.15
	}

	clampByte := func(val float32) uint8 {
		if val > 1.0 {
			val = 1.0
		}
		if val < 0.0 {
			val = 0.0
		}
		return uint8(val * 255.0)
	}

	return rl.NewColor(clampByte(r), clampByte(g), clampByte(b), 255)
}

// DrawStudioGrid renders a clean viewport grid with X (Red) and Z (Blue) axis accents.
func DrawStudioGrid(slices int, spacing float32) {
	halfSlices := slices / 2
	totalSize := float32(halfSlices) * spacing

	gridColor := rl.NewColor(60, 65, 75, 120)
	gridSubColor := rl.NewColor(45, 48, 56, 80)

	for i := -halfSlices; i <= halfSlices; i++ {
		pos := float32(i) * spacing
		c := gridSubColor
		if i%5 == 0 {
			c = gridColor
		}

		// Skip center lines for axis drawing
		if i != 0 {
			rl.DrawLine3D(rl.Vector3{X: pos, Y: 0, Z: -totalSize}, rl.Vector3{X: pos, Y: 0, Z: totalSize}, c)
			rl.DrawLine3D(rl.Vector3{X: -totalSize, Y: 0, Z: pos}, rl.Vector3{X: totalSize, Y: 0, Z: pos}, c)
		}
	}

	// Colored Main Axes
	rl.DrawLine3D(rl.Vector3{X: -totalSize, Y: 0, Z: 0}, rl.Vector3{X: totalSize, Y: 0, Z: 0}, rl.NewColor(220, 60, 60, 200)) // X Axis
	rl.DrawLine3D(rl.Vector3{X: 0, Y: 0, Z: -totalSize}, rl.Vector3{X: 0, Y: 0, Z: totalSize}, rl.NewColor(60, 120, 240, 200)) // Z Axis
}

func CheckerTexture() rl.Texture2D {
	img := rl.GenImageChecked(256, 256, 32, 32, rl.LightGray, rl.DarkGray)
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

func MatCap(normal, viewDir rl.Vector3) rl.Color {
	ny := normal.Y*0.5 + 0.5
	rim := float32(math.Pow(float64(1-rl.Vector3DotProduct(normal, viewDir)), 2.5))
	val := uint8(math.Min(255, float64(120+90*ny+45*rim)))
	return rl.NewColor(val, val, val, 255)
}
