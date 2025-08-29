package skyline

import (
	"fmt"
	"math"

	"github.com/n-goncharov/packer/internal/packing/container"
	"github.com/n-goncharov/packer/internal/packing/decoder"
	"github.com/n-goncharov/packer/internal/packing/item"
)

// @todo create package level (const) errors through errors.New.
// @todo add tests
// @todo add godoc

// @todo make stateless
type Decoder struct {
	strips      []*container.Strip
	rects       []*item.PackedRect // @todo use doubly linked list implementation
	segmentList *segmentList
}

func NewDecoder(strips []*container.Strip) (*Decoder, error) {
	return &Decoder{
		strips: strips,
	}, nil
}

func (d *Decoder) Decode(rects []*item.Rect) (*decoder.Solution[*item.PackedRect, uint], error) {
	// Init state of Decoder.
	d.init(rects)

	// Validate input.
	err := d.validate(rects)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Create a slice of packed rects to save packing ordering.
	packedRects := make([]*item.PackedRect, 0, len(rects))

	// @todo after making rects doubly linked list the condition will be: while len(d.rects) != 0.
	// Pack rects until the number of packed equals the total.
	for packedCount := 0; packedCount != len(d.rects); {
		// Get the current left-bottom segment to pack a rect on it.
		lbSegmentNode, err := d.segmentList.getLeftBottom()
		if err != nil {
			return nil, err
		}

		// Get the best-fit rect to pack onto left-bottom segment.
		bestFitRect := d.getBestFit(lbSegmentNode.data)

		// If no such rect was found then all of them are wider than the left-bottom segment,
		// then join left-bottom segment with the lowest neighboring segment
		// and start new iteration.
		if bestFitRect == nil {
			if err := d.join(lbSegmentNode); err != nil {
				return nil, fmt.Errorf("error joining LB segment: %w", err)
			}
			continue
		}

		// If the best-fit rect was found, pack it.
		err = d.pack(bestFitRect, lbSegmentNode)
		if err != nil {
			return nil, err
		}

		// Mark the best-fit rect as packed.
		bestFitRect.Packed = true

		// Add packed rect.
		packedRects = append(packedRects, bestFitRect)

		// Increment the packed counter.
		packedCount++
	}

	// Return the solution: packed rects and objective function value.
	return &decoder.Solution[*item.PackedRect, uint]{
		PackedItems: packedRects,
		ObjVal:      d.segmentList.topHeight(),
	}, nil
}

func (d *Decoder) init(rects []*item.Rect) {
	d.rects = toPackedRects(rects)
	d.initSegmentList()
}

// @todo also remove each segment on which each rect cannot be packed
func (d *Decoder) validate(rects []*item.Rect) error {
	// Find width of the widest strip.
	var maxWidth uint
	for s := d.segmentList.head; s != nil; s = s.next {
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

func toPackedRects(rects []*item.Rect) []*item.PackedRect {
	packingRects := make([]*item.PackedRect, len(rects))
	for i, rect := range rects {
		packingRects[i] = &item.PackedRect{Item: rect}
	}
	return packingRects
}

func (d *Decoder) initSegmentList() {
	var x uint
	sl := &segmentList{}
	for _, strip := range d.strips {
		sl.append(newSegment(strip.Width, x, 0))
		x += strip.Width
	}
	d.segmentList = sl
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

	// @todo maybe move this check to the tests later and remove error return
	// Here we are not comparing r.Height with zero
	// because there could be join with previous segment but next was the same height.
	if lbSegment.data.lHeight >= 0 || lbSegment.data.rHeight > 0 {
		return nil, fmt.Errorf(
			"corrupted LB segment. lHeight: %f, rHeight: %f",
			lbSegment.data.lHeight,
			lbSegment.data.rHeight,
		)
	}
	return lbSegment, nil
}

func (d *Decoder) getBestFit(lbSegm *segment) *item.PackedRect {
	var bestFitRect *item.PackedRect
	bestFitVal := -1
	for _, r := range d.rects {
		// If the rect has already been packed or
		// wider than the width of the segment, check next.
		if r.Packed || r.Item.Width > lbSegm.width {
			continue
		}

		// Declare fitness value for the current rect.
		fitVal := 0

		// Increment if the rect height and the left segment height are equal.
		if r.Item.Height == uint(math.Abs(lbSegm.lHeight)) {
			fitVal++
		}

		// Increment if the rect height and the right segment height are equal.
		if r.Item.Height == uint(math.Abs(lbSegm.rHeight)) {
			fitVal++
		}

		// Increment if the rect width and the segment width are equal.
		if r.Item.Width == lbSegm.width {
			fitVal++
		}

		// If the current rect's fitness value > best,
		// then update best value and best rect.
		if fitVal > bestFitVal {
			bestFitVal = fitVal
			bestFitRect = r
		}
	}
	return bestFitRect
}

func (d *Decoder) adjustLHeight(segmentNode *segmentNode, height float64) error {
	if segmentNode.data.lHeight == math.Inf(-1) {
		return nil
	}

	// If the segment isn't on the left edge of the strip,
	// adjust its left height and the right height of the prev segment.
	if segmentNode == d.segmentList.head {
		return fmt.Errorf("corrupted left height of the very first segment")
	}
	segmentNode.data.lHeight += height
	segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
	return nil
}

func (d *Decoder) adjustRHeight(segmentNode *segmentNode, height float64) error {
	if segmentNode.data.rHeight == math.Inf(-1) {
		return nil
	}

	// If the segment isn't on the right edge of the strip,
	// adjust its right height and the left height of the next segment.
	if segmentNode == d.segmentList.tail {
		return fmt.Errorf("corrupted right height of the very last segment")
	}
	segmentNode.data.rHeight += height
	segmentNode.next.data.lHeight = -segmentNode.data.rHeight

	return nil
}

func (d *Decoder) joinNext(segmentNode *segmentNode) error {
	if segmentNode == d.segmentList.tail {
		return fmt.Errorf("trying to join last segment with next")
	}

	if err := d.adjustLHeight(segmentNode, -segmentNode.data.rHeight); err != nil {
		return err
	}

	// Adjust the width, y-coordinate, right height of the segment.
	segmentNode.data.width += segmentNode.next.data.width
	segmentNode.data.y = segmentNode.next.data.y
	segmentNode.data.rHeight = segmentNode.next.data.rHeight

	// Delete the next segment.
	d.segmentList.delete(segmentNode.next)

	return nil
}

func (d *Decoder) joinPrev(segmentNode *segmentNode) error {
	if segmentNode == d.segmentList.head {
		return fmt.Errorf("trying to join first segment with prev")
	}

	if err := d.adjustRHeight(segmentNode, -segmentNode.data.lHeight); err != nil {
		return err
	}

	// Adjust the width, x-coordinate, y-coordinate, left height of the segment.
	segmentNode.data.width += segmentNode.prev.data.width
	segmentNode.data.x = segmentNode.prev.data.x
	segmentNode.data.y = segmentNode.prev.data.y
	segmentNode.data.lHeight = segmentNode.prev.data.lHeight

	// Delete the prev segment.
	d.segmentList.delete(segmentNode.prev)

	return nil
}

func (d *Decoder) join(segmentNode *segmentNode) error {
	// In case segment is a strip, and we cannot join it -> just delete this segment.
	if segmentNode.data.lHeight == math.Inf(-1) && segmentNode.data.rHeight == math.Inf(-1) {
		d.segmentList.delete(segmentNode)
		return nil
	}

	// Check with which neighbouring segment to join.
	if segmentNode.data.lHeight < segmentNode.data.rHeight {
		return d.joinNext(segmentNode)
	}
	return d.joinPrev(segmentNode)
}

func (d *Decoder) pack(rect *item.PackedRect, segmentNode *segmentNode) error {
	// Set segment's coordinates to the rect.
	rect.X = segmentNode.data.x
	rect.Y = segmentNode.data.y

	switch {
	case rect.Item.Width == segmentNode.data.width:
		// Adjust the y-coordinate.
		segmentNode.data.y += rect.Item.Height

		if err := d.adjustLHeight(segmentNode, float64(rect.Item.Height)); err != nil {
			return err
		}

		if err := d.adjustRHeight(segmentNode, float64(rect.Item.Height)); err != nil {
			return err
		}

	case rect.Item.Width < segmentNode.data.width:
		// Create a new segment.
		s := newSegment(rect.Item.Width, segmentNode.data.x, segmentNode.data.y+rect.Item.Height)

		// If the segment isn't on the left edge of the strip,
		// adjust new segment's left height and the right height of the prev segment.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segmentList.head {
				return fmt.Errorf("corrupted left height of the very first segment")
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
		d.segmentList.insertBefore(segmentNode, s)
	case rect.Item.Width > segmentNode.data.width:
		return fmt.Errorf("rect is wider than segment")
	}
	return nil
}
