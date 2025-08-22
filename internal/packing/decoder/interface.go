package decoder

import (
	"golang.org/x/exp/constraints"
)

type Interface[I any, V constraints.Integer | constraints.Float] interface {
	Decode() (*Solution[I, V], error)
}
