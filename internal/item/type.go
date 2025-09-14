package item

import "fmt"

// @todo think about if it is good to have I as Stringer interface here
type Packed[I fmt.Stringer] struct {
	Item I
	X, Y uint
	// @todo get rid of it in the future
	Packed bool
}

func (p Packed[I]) String() string {
	return fmt.Sprintf("%s at (%d, %d)", p.Item.String(), p.X, p.Y)
}
