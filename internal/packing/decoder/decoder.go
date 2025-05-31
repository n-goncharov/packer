package decoder

type Decoder[I any] interface {
	Decode() (*Solution[I], error)
}

type Solution[I any] struct {
	PackedItems []I
	ObjFnVal    int
}
