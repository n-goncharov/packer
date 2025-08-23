package decoder

import "golang.org/x/exp/constraints"

type Solution[PI any, V constraints.Integer | constraints.Float] struct {
	PackedItems []PI
	ObjVal      V
}
