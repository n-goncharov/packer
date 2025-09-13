package packer

type Interface[I, C, PI any] interface {
	Pack([]*I, []*C) (*Solution[PI], error)
}
