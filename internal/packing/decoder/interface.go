package decoder

import (
	"golang.org/x/exp/constraints"
)

// @todo think about using pointers for generic types
// @todo think about constraints for generic types here
type Interface[I any, PI any, V constraints.Integer | constraints.Float] interface {
	Decode([]*I) (*Solution[PI, V], error)
}
