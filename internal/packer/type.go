package packer

import (
	"github.com/n-goncharov/packer/internal/decoder"
)

type Solution[PI any] struct {
	PackedItems []*PI
	decoder.FitnessValue
}
