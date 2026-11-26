package item

import "fmt"

type Rect struct {
	Width, Height uint
}

func (r *Rect) String() string {
	return fmt.Sprintf(
		"Rect{Width: %d, Height: %d}",
		r.Width,
		r.Height,
	)
}

type PackedRect struct {
	Rect
	Placement
}

func (p *PackedRect) String() string {
	return fmt.Sprintf("%s at (%d, %d)", p.Rect.String(), p.X, p.Y)
}
