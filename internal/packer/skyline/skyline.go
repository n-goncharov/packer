package skyline

import (
	"errors"
	"fmt"
	"math"

	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/packer"
)

// Order of todos:
// @todo make all todos in packing folder
// @todo add tests for code here (especially fuzzing)
// @todo try to deduplicate (if possible) and refine
// @todo add godoc for all written code

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal error")
)

type Packer struct{}

// @todo 1. make rects to be doubly linked list too (create generic list impl first)
// to get rid of 'Packed' bool field (just remove rect after packing)
// and search best-fit through remaining list - not all rects.

func NewPacker() *Packer {
	return &Packer{}
}

func (d *Packer) Pack(rects []*item.Rectangle, strips []*container.Strip) (*packer.Solution[item.PackedRectangle], error) {

	rectsToPack := toPackedRects(rects)

	sl := createSegmentList(strips)

	// Validate input.
	if err := validate(rects, sl); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// @todo 1. create a doubly linked list of rects

	// Create a slice of packed rects to save packing ordering.
	packedRects := make([]*item.PackedRectangle, 0, len(rects))

	// Pack rects until the number of packed equals the total.
	for packedCount := 0; packedCount != len(rects); {
		// Get the current left-bottom segment to pack a rect on it.
		lbSegmentNode, err := sl.getLeftBottom()
		if err != nil {
			return nil, err
		}

		// Get the best-fit rect to pack onto left-bottom segment.
		bestFitRect := getBestFit(rectsToPack, lbSegmentNode.data)

		// If no such rect is found it means that all rects are wider than the left-bottom segment.
		// In this case merge left-bottom segment with the lowest neighboring segment and start new iteration.
		if bestFitRect == nil {
			if err := merge(sl, lbSegmentNode); err != nil {
				return nil, fmt.Errorf("%w: error merging LB segment: %v", ErrInternal, err)
			}
			continue
		}

		// If the best-fit rect is found, pack it.
		err = pack(sl, bestFitRect, lbSegmentNode)
		if err != nil {
			return nil, fmt.Errorf("%w: error packing rect: %v", ErrInternal, err)
		}

		// @todo 1. remove this
		// Mark the best-fit rect as packed.
		bestFitRect.Packed = true

		// Add packed rect.
		packedRects = append(packedRects, bestFitRect)

		// Increment the packed counter.
		packedCount++

		// @todo 1. packed rect should be removed from rects list
	}

	// Return the solution: packed rects and objective function value.
	return &packer.Solution[item.PackedRectangle]{
		PackedItems:  packedRects,
		FitnessValue: sl.topHeight(),
	}, nil
}

// @todo 2. also remove each segment on which each rect cannot be packed (segment's width less than each rect's width)
func validate(rects []*item.Rectangle, sl *segmentList) error {
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

func toPackedRects(rects []*item.Rectangle) []*item.PackedRectangle {
	packingRects := make([]*item.PackedRectangle, len(rects))
	for i, rect := range rects {
		packingRects[i] = &item.PackedRectangle{Item: rect}
	}
	return packingRects
}

func createSegmentList(strips []*container.Strip) *segmentList {
	var x uint
	sl := &segmentList{}
	for _, strip := range strips {
		sl.append(newSegment(strip.Width, x, 0))
		x += strip.Width
	}
	return sl
}

func (sl *segmentList) getLeftBottom() (*segmentNode, error) {
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
func getBestFit(rects []*item.PackedRectangle, lbSegm *segment) *item.PackedRectangle {
	var bestFitRect *item.PackedRectangle
	bestFitVal := -1
	for _, r := range rects {
		// If the rect has already been packed or
		// wider than the width of the segment, check next.
		if r.Packed || r.Item.Width > lbSegm.width {
			continue
		}

		// Declare fitness value for the current rect.
		fitVal := 0

		// @todo should lHeight and rHeight be validated here (error if they are > (=?) 0)?
		// Increment in the following cases:
		// The rect height and the left segment height are equal.
		if r.Item.Height == uint(lbSegm.lHeight) {
			fitVal++
		}

		// The rect height and the right segment height are equal.
		if r.Item.Height == uint(lbSegm.rHeight) {
			fitVal++
		}

		// The rect width and the segment width are equal.
		if r.Item.Width == lbSegm.width {
			fitVal++
		}

		// Update the best fitness value and the best rect.
		if fitVal > bestFitVal {
			bestFitVal = fitVal
			bestFitRect = r
		}
	}
	return bestFitRect
}

// @todo 4. change to method of segments list
func adjustLHeight(sl *segmentList, segmentNode *segmentNode, height float64) error {
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
func adjustRHeight(sl *segmentList, segmentNode *segmentNode, height float64) error {
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
func mergeNext(sl *segmentList, segmentNode *segmentNode) error {
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
	sl.delete(segmentNode.next)

	return nil
}

// @todo 4. change to method of segments list
func mergePrev(sl *segmentList, segmentNode *segmentNode) error {
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
	sl.delete(segmentNode.prev)

	return nil
}

// @todo 4. change to method of segments list
func merge(sl *segmentList, segmentNode *segmentNode) error {
	// In case segment is a strip, and we cannot merge it -> just delete this segment.
	if math.IsInf(segmentNode.data.lHeight, -1) && math.IsInf(segmentNode.data.rHeight, -1) {
		sl.delete(segmentNode)
		return nil
	}

	// Check with which neighbouring segment to merge.
	if segmentNode.data.lHeight < segmentNode.data.rHeight {
		return mergeNext(sl, segmentNode)
	}
	return mergePrev(sl, segmentNode)
}

// @todo 4. change to method of segments list
func pack(sl *segmentList, rect *item.PackedRectangle, segmentNode *segmentNode) error {
	// Set segment's coordinates to the rect.
	rect.X = segmentNode.data.x
	rect.Y = segmentNode.data.y

	switch {
	case rect.Item.Width == segmentNode.data.width:
		// Adjust the y-coordinate.
		segmentNode.data.y += rect.Item.Height

		if err := adjustLHeight(sl, segmentNode, float64(rect.Item.Height)); err != nil {
			return err
		}

		if err := adjustRHeight(sl, segmentNode, float64(rect.Item.Height)); err != nil {
			return err
		}

	case rect.Item.Width < segmentNode.data.width:
		// Create a new segment.
		s := newSegment(rect.Item.Width, segmentNode.data.x, segmentNode.data.y+rect.Item.Height)

		// If the segment isn't on the left edge of the strip,
		// adjust new segment's left height and the right height of the prev segment.
		if !math.IsInf(segmentNode.data.lHeight, -1) {
			if segmentNode == sl.head {
				return errors.New("corrupted left height of the very first segment")
			}
			s.lHeight = segmentNode.data.lHeight + float64(rect.Item.Height)
			segmentNode.prev.data.rHeight = -s.lHeight
		}

		// Set right height for the new segment.
		s.rHeight = float64(rect.Item.Height)

		// Adjust segment's x-coordinate, width, left height.
		segmentNode.data.x += rect.Item.Width
		segmentNode.data.width -= rect.Item.Width
		segmentNode.data.lHeight = -s.rHeight

		// Insert new segment before current.
		sl.insertBefore(segmentNode, s)
	case rect.Item.Width > segmentNode.data.width:
		return fmt.Errorf("rect is wider than segment")
	}
	return nil
}
