package engine

// Tool — lifecycle contract every micro-app implements.
type Tool interface {
	Name() string
	Description() string
	Init(ctx *Context) error
	Update(ctx *Context, dt float32)
	Draw3D(ctx *Context)
	DrawUI(ctx *Context)
	OnSave(ctx *Context) error
}
