package engine

type ShortcutHelp struct {
	Key         string
	Description string
}

// Tool — lifecycle contract every micro-app implements.
type Tool interface {
	Name() string
	Description() string
	Shortcuts() []ShortcutHelp
	Init(ctx *Context) error
	Update(ctx *Context, dt float32)
	Draw3D(ctx *Context)
	DrawUI(ctx *Context)
	OnSave(ctx *Context) error
}
