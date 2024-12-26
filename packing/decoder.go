package packing

type Decoder[I any, C any] interface {
	Decode(items []I, containers []C) (*Solution[I], error)
}

type Solution[I any] struct {
	PackedItems []I
	ObjFuncVal  int
}
