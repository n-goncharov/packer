package item

import "fmt"

type Rectangle struct {
	Width, Height uint
}

type PackedRectangle = Packed[*Rectangle]

func (r Rectangle) String() string {
	return fmt.Sprintf(
		"Rect{Width: %d, Height: %d}",
		r.Width,
		r.Height,
	)
}
