package camera

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Viewport struct {
	Camera    rl.Camera3D
	Distance  float32
	Yaw       float32
	Pitch     float32
	Target    rl.Vector3
	IsOrtho   bool
	OrthoSize float32
}

func New() Viewport {
	vp := Viewport{
		Distance:  12, Yaw: 45, Pitch: 30,
		Target:    rl.Vector3{X: 0, Y: 1, Z: 0},
		OrthoSize: 8,
		Camera:    rl.Camera3D{Up: rl.Vector3{X: 0, Y: 1, Z: 0}, Fovy: 45, Projection: rl.CameraPerspective},
	}
	vp.Update()
	return vp
}

func (vp *Viewport) Update() {
	pitch := vp.Pitch * rl.Deg2rad
	yaw := vp.Yaw * rl.Deg2rad
	x := vp.Distance * float32(math.Cos(float64(pitch))*math.Sin(float64(yaw)))
	y := vp.Distance * float32(math.Sin(float64(pitch)))
	z := vp.Distance * float32(math.Cos(float64(pitch))*math.Cos(float64(yaw)))
	vp.Camera.Target = vp.Target
	vp.Camera.Position = rl.Vector3Add(vp.Target, rl.Vector3{X: x, Y: y, Z: z})
	if vp.IsOrtho {
		vp.Camera.Projection = rl.CameraOrthographic
		vp.Camera.Fovy = vp.OrthoSize
	} else {
		vp.Camera.Projection = rl.CameraPerspective
		vp.Camera.Fovy = 45
	}
}

func (vp *Viewport) HandleInput(allow bool) {
	if !allow {
		return
	}
	if w := rl.GetMouseWheelMove(); w != 0 {
		vp.Distance -= w * vp.Distance * 0.12
		if vp.Distance < 0.2 {
			vp.Distance = 0.2
		}
		vp.OrthoSize -= w * vp.OrthoSize * 0.12
		if vp.OrthoSize < 0.5 {
			vp.OrthoSize = 0.5
		}
		vp.Update()
	}
	orbit := rl.IsMouseButtonDown(rl.MouseButtonMiddle) || (rl.IsKeyDown(rl.KeyLeftAlt) && rl.IsMouseButtonDown(rl.MouseButtonLeft))
	pan := rl.IsMouseButtonDown(rl.MouseButtonMiddle) && (rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift))
	if pan {
		d := rl.GetMouseDelta()
		fwd := rl.Vector3Normalize(rl.Vector3Subtract(vp.Camera.Target, vp.Camera.Position))
		right := rl.Vector3Normalize(rl.Vector3CrossProduct(fwd, vp.Camera.Up))
		up := rl.Vector3CrossProduct(right, fwd)
		speed := vp.Distance * 0.0015
		if vp.IsOrtho {
			speed = vp.OrthoSize * 0.0015
		}
		p := rl.Vector3Add(rl.Vector3Scale(right, -d.X*speed), rl.Vector3Scale(up, d.Y*speed))
		vp.Target = rl.Vector3Add(vp.Target, p)
		vp.Update()
	} else if orbit {
		d := rl.GetMouseDelta()
		vp.Yaw -= d.X * 0.35
		vp.Pitch += d.Y * 0.35
		if vp.Pitch > 89.9 {
			vp.Pitch = 89.9
		}
		if vp.Pitch < -89.9 {
			vp.Pitch = -89.9
		}
		vp.Update()
	}
}

func (vp *Viewport) Snap(view string, ortho bool) {
	switch view {
	case "Front":
		vp.Yaw, vp.Pitch = 0, 0
	case "Back":
		vp.Yaw, vp.Pitch = 180, 0
	case "Right":
		vp.Yaw, vp.Pitch = 90, 0
	case "Left":
		vp.Yaw, vp.Pitch = -90, 0
	case "Top":
		vp.Yaw, vp.Pitch = 0, 89.9
	case "Bottom":
		vp.Yaw, vp.Pitch = 0, -89.9
	case "Iso":
		vp.Yaw, vp.Pitch = 45, 30
	}
	if ortho {
		vp.IsOrtho = true
	}
	vp.Update()
}

func (vp *Viewport) ToggleOrtho() { vp.IsOrtho = !vp.IsOrtho; vp.Update() }
