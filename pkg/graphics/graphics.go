package graphics

const (
	GraphicsWidth  = 64
	GraphicsHeight = 32
)

type Graphics struct {
	Pixels [GraphicsWidth * GraphicsHeight]byte
}
