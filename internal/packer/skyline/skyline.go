package skyline

import (
	"errors"
	"fmt"
	"math"

	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/packer"
)

// todo make all todos in packing folder
// todo add tests for code here (especially fuzzing)
// todo try to deduplicate (if possible) and refine
// todo add GoDoc for all written code

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal error")
)

type Packer struct{}

func NewPacker() *Packer {
	return &Packer{}
}

const notFoundIdx = -1

func (d *Packer) Pack(rects []*item.Rect, strips []*container.Strip) (*packer.Solution[item.PackedRect], error) {
	sl := newSegmList(strips)

	if err := validate(rects, sl); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	packedRects := make([]*item.PackedRect, 0, len(rects))

	packedFlags := make([]bool, len(rects))

	for packedCount := 0; packedCount != len(rects); {
		lbSegmNode, err := sl.lbSegmNode()
		if err != nil {
			return nil, err
		}

		// Get the best-fit rect to pack onto left-bottom segment.
		// If no such rect is found then all rects are wider than that segment.
		// In this case merge LB-segment with the lowest neighboring and start new iteration.
		bestFitIdx := bestFitIdx(rects, packedFlags, lbSegmNode.data)
		if bestFitIdx == notFoundIdx {
			if err := merge(sl, lbSegmNode); err != nil {
				return nil, fmt.Errorf("%w: error merging LB segment: %v", ErrInternal, err)
			}
			continue
		}
		packedFlags[bestFitIdx] = true

		// If the best-fit rect is found, pack it.
		packedRect, err := pack(sl, rects[bestFitIdx], lbSegmNode)
		if err != nil {
			return nil, fmt.Errorf("%w: error packing rect: %v", ErrInternal, err)
		}

		// Add packed rect.
		packedRects = append(packedRects, packedRect)

		// Increment the packed counter.
		packedCount++
	}

	// Return the solution: packed rects and objective function value.
	return &packer.Solution[item.PackedRect]{
		PackedItems:  packedRects,
		FitnessValue: sl.height(),
	}, nil
}

// @todo 2. also remove each segm on which each rect cannot be packed (segment's width less than each rect's width)
func validate(rects []*item.Rect, sl *segmList) error {
	// Find width of the widest strip.
	var maxWidth uint
	for s := sl.head; s != nil; s = s.next {
		if s.data.width > maxWidth {
			maxWidth = s.data.width
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

func newSegmList(strips []*container.Strip) *segmList {
	var x uint
	sl := &segmList{}
	for _, strip := range strips {
		sl.pushBack(newSegm(strip.Width, x, 0))
		x += strip.Width
	}
	return sl
}

func (sl *segmList) lbSegmNode() (*segmNode, error) {
	lbSegment := sl.head
	current := sl.head.next
	for current != nil {
		if current.data.y < lbSegment.data.y {
			lbSegment = current
		}
		current = current.next
	}

	// @todo 3. maybe move this check to the tests later and remove error return
	// Here we are not comparing r.Height with zero
	// because there could be merge with previous segment but next was the same height.
	if lbSegment.data.lHeight >= 0 || lbSegment.data.rHeight > 0 {
		return nil, fmt.Errorf(
			"corrupted LB segment\n lHeight: %f, rHeight: %f",
			lbSegment.data.lHeight,
			lbSegment.data.rHeight,
		)
	}
	return lbSegment, nil
}

// @todo 1. change to method of rects list
func bestFitIdx(rects []*item.Rect, packedFlags []bool, targetSegm *segm) int {
	bestFitVal := -1
	bestFitIdx := notFoundIdx
	for i, r := range rects {
		// If the rect has already been packed or
		// wider than the segment's width, check next.
		if packedFlags[i] || r.Width > targetSegm.width {
			continue
		}

		// Declare fitness value for the current rect.
		fitVal := 0

		// @todo should lHeight and rHeight be validated here (error if they are > (=?) 0)?
		// Increment in the following cases:
		// The rect height and the left segment height are equal.
		if r.Height == uint(targetSegm.lHeight) {
			fitVal++
		}

		// The rect height and the right segment height are equal.
		if r.Height == uint(targetSegm.rHeight) {
			fitVal++
		}

		// The rect width and the segment width are equal.
		if r.Width == targetSegm.width {
			fitVal++
		}

		// Update the best fitness value.
		if fitVal > bestFitVal {
			bestFitVal = fitVal
			bestFitIdx = i
		}
	}
	return bestFitIdx
}

// @todo 4. change to method of segments list
func adjustLHeight(sl *segmList, segmentNode *segmNode, height float64) error {
	if math.IsInf(segmentNode.data.lHeight, -1) {
		return nil
	}

	// If the segment isn't on the left edge of the strip,
	// adjust its left height and the right height of the prev segment.
	if segmentNode == sl.head {
		return errors.New("corrupted left height of the very first segment")
	}
	segmentNode.data.lHeight += height
	segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
	return nil
}

// @todo 4. change to method of segments list
func adjustRHeight(sl *segmList, segmentNode *segmNode, height float64) error {
	if math.IsInf(segmentNode.data.rHeight, -1) {
		return nil
	}

	// If the segment isn't on the right edge of the strip,
	// adjust its right height and the left height of the next segment.
	if segmentNode == sl.tail {
		return errors.New("corrupted right height of the very last segment")
	}
	segmentNode.data.rHeight += height
	segmentNode.next.data.lHeight = -segmentNode.data.rHeight

	return nil
}

// @todo 4. change to method of segments list
func mergeNext(sl *segmList, segmentNode *segmNode) error {
	if segmentNode == sl.tail {
		return errors.New("trying to merge last segment with next")
	}

	if err := adjustLHeight(sl, segmentNode, -segmentNode.data.rHeight); err != nil {
		return err
	}

	// Adjust the width, y-coordinate, right height of the segment.
	segmentNode.data.width += segmentNode.next.data.width
	segmentNode.data.y = segmentNode.next.data.y
	segmentNode.data.rHeight = segmentNode.next.data.rHeight

	// Delete the next segment.
	sl.remove(segmentNode.next)

	return nil
}

// @todo 4. change to method of segments list
func mergePrev(sl *segmList, segmentNode *segmNode) error {
	if segmentNode == sl.head {
		return errors.New("trying to merge first segment with prev")
	}

	if err := adjustRHeight(sl, segmentNode, -segmentNode.data.lHeight); err != nil {
		return err
	}

	// Adjust the width, x-coordinate, y-coordinate, left height of the segment.
	segmentNode.data.width += segmentNode.prev.data.width
	segmentNode.data.x = segmentNode.prev.data.x
	segmentNode.data.y = segmentNode.prev.data.y
	segmentNode.data.lHeight = segmentNode.prev.data.lHeight

	// Delete the prev segment.
	sl.remove(segmentNode.prev)

	return nil
}

// @todo 4. change to method of segments list
func merge(sl *segmList, segmentNode *segmNode) error {
	// In case segment is a strip, and we cannot merge it -> just delete this segment.
	if math.IsInf(segmentNode.data.lHeight, -1) && math.IsInf(segmentNode.data.rHeight, -1) {
		sl.remove(segmentNode)
		return nil
	}

	// Check with which neighbouring segment to merge.
	if segmentNode.data.lHeight < segmentNode.data.rHeight {
		return mergeNext(sl, segmentNode)
	}
	return mergePrev(sl, segmentNode)
}

// @todo 4. change to method of segments list
func pack(sl *segmList, rect *item.Rect, segmentNode *segmNode) (*item.PackedRect, error) {
	packedRect := &item.PackedRect{
		Rect: *rect,
		X:    segmentNode.data.x,
		Y:    segmentNode.data.y,
	}

	switch {
	case packedRect.Width == segmentNode.data.width:
		// Adjust the y-coordinate.
		segmentNode.data.y += rect.Height

		if err := adjustLHeight(sl, segmentNode, float64(rect.Height)); err != nil {
			return nil, err
		}

		if err := adjustRHeight(sl, segmentNode, float64(rect.Height)); err != nil {
			return nil, err
		}

	case rect.Width < segmentNode.data.width:
		// Create a new segment.
		s := newSegm(rect.Width, segmentNode.data.x, segmentNode.data.y+rect.Height)

		// If the segment isn't on the left edge of the strip,
		// adjust new segment's left height and the right height of the prev segment.
		if !math.IsInf(segmentNode.data.lHeight, -1) {
			if segmentNode == sl.head {
				return nil, errors.New("corrupted left height of the very first segment")
			}
			s.lHeight = segmentNode.data.lHeight + float64(rect.Height)
			segmentNode.prev.data.rHeight = -s.lHeight
		}

		// Set right height for the new segment.
		s.rHeight = float64(rect.Height)

		// Adjust segment's x-coordinate, width, left height.
		segmentNode.data.x += rect.Width
		segmentNode.data.width -= rect.Width
		segmentNode.data.lHeight = -s.rHeight

		// Insert new segment before current.
		sl.insertBefore(segmentNode, s)
	case rect.Width > segmentNode.data.width:
		return nil, fmt.Errorf("rect is wider than segment")
	}
	return packedRect, nil
}
