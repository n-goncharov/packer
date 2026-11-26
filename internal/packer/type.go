package packer

import (
	"github.com/n-goncharov/packer/internal/decoder"
)

// todo is it better to use generics here or interface. e.g., Packable or something like this?
type Solution[PI any] struct {
	PackedItems []*PI
	decoder.FitnessValue
}
