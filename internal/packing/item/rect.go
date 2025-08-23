package item

import "fmt"

type Rect struct {
	Width, Height uint
}

type PackedRect = Packed[*Rect]

func (r Rect) String() string {
	return fmt.Sprintf(
		"Rect{Width: %d, Height: %d}",
		r.Width,
		r.Height,
	)
}
