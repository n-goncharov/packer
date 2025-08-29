package decoder

import "golang.org/x/exp/constraints"

// @todo think about constraints for generic types here
// @todo is it better to use generics here or interface. e.g., Packable or something like this?
type Solution[PI any, V constraints.Integer | constraints.Float] struct {
	PackedItems []PI
	ObjVal      V
}
