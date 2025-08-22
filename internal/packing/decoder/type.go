package decoder

import "golang.org/x/exp/constraints"

type Solution[I any, V constraints.Integer | constraints.Float] struct {
	Items  []I
	ObjVal V
}
