package item

import "fmt"

type Placed[I fmt.Stringer] struct {
	Item I
	X, Y uint
	// @todo get rid in the future
	Packed bool
}

func (p Placed[I]) String() string {
	return fmt.Sprintf("%s at (%d, %d)", p.Item.String(), p.X, p.Y)
}
