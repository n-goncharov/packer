package skyline

import (
	"errors"
	"fmt"

	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/packer"
)

// TODO Consider checking next and prev segm on nil value instead of storing and recalculating lHeight and rHeight.
// TODO Reconsider architecture of the entire project.
// TODO Add tests (especially fuzzing).
// TODO Add GoDoc for all the written code.

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal error")
)

type Packer struct{}

func NewPacker() *Packer {
	return &Packer{}
}

func (d *Packer) Pack(rects []*item.Rect, strips []*container.Strip) (*packer.Solution[item.PackedRect], error) {
	sl := newSegmList(strips)

	if err := validate(rects, sl); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	packedRects, err := sl.pack(rects)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return &packer.Solution[item.PackedRect]{
		PackedItems:  packedRects,
		FitnessValue: sl.height(),
	}, nil
}

func validate(rects []*item.Rect, sl *segmList) error {
	// Find width of the widest strip.
	var maxWidth uint
	for s := sl.Head; s != nil; s = s.Next {
		if s.Value.width > maxWidth {
			maxWidth = s.Value.width
		}
	}

	// Check if all rectangles can fit on it.
	for _, rect := range rects {
		if rect.Width > maxWidth {
			return fmt.Errorf("rectangle width mustn't be greater than the widest strip (%d)", maxWidth)
		}
	}

	return nil
}
