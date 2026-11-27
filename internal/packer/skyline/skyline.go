package skyline

import (
	"errors"
	"fmt"

	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/list"
	"github.com/n-goncharov/packer/internal/packer"
)

// TODO Reconsider architecture of the entire project.
// TODO Add tests (especially fuzzing).
// TODO Add GoDoc for all the written code.

var (
	ErrInvalidInput = errors.New("invalid input")
)

type Packer struct{}

func NewPacker() *Packer {
	return &Packer{}
}

func (d *Packer) Pack(rects []*item.Rect, strips []*container.Strip) (*packer.Solution[item.PackedRect], error) {
	slList := make([]*segmList, len(strips))
	var x uint
	for i, strip := range strips {
		slList[i] = newSegmList(x, strip.Width)
		x += strip.Width
	}

	if err := validate(rects, slList); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	return &packer.Solution[item.PackedRect]{
		PackedItems:  packRects(slList, rects),
		FitnessValue: packingHeight(slList),
	}, nil
}

func validate(rects []*item.Rect, slList []*segmList) error {
	// Find width of the widest strip.
	var maxWidth uint
	for _, sl := range slList {
		if sl.Head.Value.width > maxWidth {
			maxWidth = sl.Head.Value.width
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

func packRects(slList []*segmList, rects []*item.Rect) []*item.PackedRect {
	packedRects := make([]*item.PackedRect, 0, len(rects))
	packedFlags := make([]bool, len(rects))
	for packedCount := 0; packedCount != len(rects); {
		// Choose target segment.
		slIdx, targetSegmNode := targetSegm(slList)

		// Find best-fit rectangle to pack onto the target segment.
		bestFitIdx := bestFitRectIdx(rects, packedFlags, targetSegmNode)
		if bestFitIdx == notFoundIdx {
			slList[slIdx].merge(targetSegmNode)
			continue
		}

		// Pack best-fit rectangle on the target segment.
		packedRect := slList[slIdx].packRect(rects[bestFitIdx], targetSegmNode)
		packedRects = append(packedRects, packedRect)
		packedFlags[bestFitIdx] = true
		packedCount++
	}
	return packedRects
}

func targetSegm(slList []*segmList) (int, *list.Node[*segm]) {
	lbSegmNode := slList[0].Head
	slIdx := 0
	for i, sl := range slList {
		curr := sl.Head
		for curr != nil {
			if curr.Value.y < lbSegmNode.Value.y {
				lbSegmNode = curr
				slIdx = i
			}
			curr = curr.Next
		}
	}
	return slIdx, lbSegmNode
}

func packingHeight(slList []*segmList) uint {
	var height uint
	for _, sl := range slList {
		curr := sl.Head
		for curr != nil {
			if curr.Value.y > height {
				height = curr.Value.y
			}
			curr = curr.Next
		}
	}
	return height
}
