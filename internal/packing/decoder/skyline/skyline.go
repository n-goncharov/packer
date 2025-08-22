package skyline

import (
	"fmt"
	"math"

	"github.com/n-goncharov/packer/internal/packing/container"
	"github.com/n-goncharov/packer/internal/packing/decoder"
	"github.com/n-goncharov/packer/internal/packing/item"
)

// @todo create package level (const) errors through errors.New.

type Decoder struct {
	rects        []*item.PlacedRect // @todo use container/list instead
	segmentsList *segmentList       // @todo use container/list instead
}

func NewDecoder(rects []*item.Rect, strips []*container.Strip) (*Decoder, error) {
	// Validate input.
	err := validate(rects, strips)
	if err != nil {
		return nil, fmt.Errorf("ivalid input: %w", err)
	}

	return &Decoder{
		rects:        toPlacedRects(rects),
		segmentsList: toSegmentsList(strips),
	}, nil
}

func (d *Decoder) Decode() (*decoder.Solution[*item.PlacedRect, uint], error) {
	// Create a slice for packed rects to save packing ordering.
	packedRects := make([]*item.PlacedRect, 0, len(d.rects))

	// @todo after making rects doubly linked list the condition will be: while len(d.rects) != 0.
	// Pack rects until the number of packed equals the total.
	for packedCount := 0; packedCount != len(d.rects); {
		// Get the current left-bottom segment to pack a rect on it.
		lbSegmentNode, err := d.segmentsList.getLeftBottom()
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
	return &decoder.Solution[*item.PlacedRect, uint]{
		Items:  packedRects,
		ObjVal: d.segmentsList.topHeight(),
	}, nil
}

func validate(rects []*item.Rect, strips []*container.Strip) error {
	var maxStripWidth uint
	for _, strip := range strips {
		if strip.Width > maxStripWidth {
			maxStripWidth = strip.Width
		}
	}

	for _, rect := range rects {
		if rect.Width > maxStripWidth {
			return fmt.Errorf("rectangle width mustn't be greater than the widest strip (%d)", maxStripWidth)
		}
	}

	return nil
}

func toPlacedRects(rects []*item.Rect) []*item.PlacedRect {
	packingRects := make([]*item.PlacedRect, len(rects))
	for i, rect := range rects {
		packingRects[i] = &item.PlacedRect{Item: rect}
	}
	return packingRects
}

func toSegmentsList(strips []*container.Strip) *segmentList {
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

func (d *Decoder) getBestFit(segment *segment) *item.PlacedRect {
	var bestFitRect *item.PlacedRect
	bestFitVal := -1
	for _, r := range d.rects {
		// If the rect has already been packed or
		// wider than the width of the segment, check next.
		if r.Packed || r.Item.Width > segment.width {
			continue
		}

		// Declare fitness value for the current rect.
		fitVal := 0

		// Increment if the rect height and the left segment height are equal.
		if r.Item.Height == uint(math.Abs(segment.lHeight)) {
			fitVal++
		}

		// Increment if the rect height and the right segment height are equal.
		if r.Item.Height == uint(math.Abs(segment.rHeight)) {
			fitVal++
		}

		// Increment if the rect width and the segment width are equal.
		if r.Item.Width == segment.width {
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

func (d *Decoder) join(segmentNode *segmentNode) error {
	// In case segment is strip, and we cannot join -> just delete this segment.
	if segmentNode.data.lHeight == math.Inf(-1) && segmentNode.data.rHeight == math.Inf(-1) {
		d.segmentsList.delete(segmentNode)
		return nil
	}

	// Check with which neighbouring segment to join.
	if segmentNode.data.lHeight < segmentNode.data.rHeight {
		return d.joinNext(segmentNode)
	} else {
		return d.joinPrev(segmentNode)
	}
}

// @todo dedup
func (d *Decoder) joinNext(segmentNode *segmentNode) error {
	if segmentNode == d.segmentsList.tail {
		return fmt.Errorf("trying to join last segment with next")
	}

	// @todo dedup
	// If the segment isn't on the left edge of the strip,
	// adjust its left height and the right height of the prev segment.
	if segmentNode.data.lHeight != math.Inf(-1) {
		if segmentNode == d.segmentsList.head {
			return fmt.Errorf("corrupted left height of the very first segment")
		}
		segmentNode.data.lHeight -= segmentNode.data.rHeight
		segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
	}

	// Adjust the width, y-coordinate, right height of the segment.
	segmentNode.data.width += segmentNode.next.data.width
	segmentNode.data.y = segmentNode.next.data.y
	segmentNode.data.rHeight = segmentNode.next.data.rHeight

	// Delete the next segment.
	d.segmentsList.delete(segmentNode.next)

	return nil
}

// @todo dedup
func (d *Decoder) joinPrev(segmentNode *segmentNode) error {
	if segmentNode == d.segmentsList.head {
		return fmt.Errorf("trying to join first segment with prev")
	}

	// @todo dedup
	// If the segment isn't on the right edge of the strip,
	// adjust its right height and the left height of the next segment.
	if segmentNode.data.rHeight != math.Inf(-1) {
		if segmentNode == d.segmentsList.tail {
			return fmt.Errorf("corrupted right height of the very last segment")
		}
		segmentNode.data.rHeight -= segmentNode.data.lHeight
		segmentNode.next.data.lHeight = -segmentNode.data.rHeight
	}

	// Adjust the width, x-coordinate, y-coordinate, left height of the segment.
	segmentNode.data.width += segmentNode.prev.data.width
	segmentNode.data.x = segmentNode.prev.data.x
	segmentNode.data.y = segmentNode.prev.data.y
	segmentNode.data.lHeight = segmentNode.prev.data.lHeight

	// Delete the prev segment.
	d.segmentsList.delete(segmentNode.prev)

	return nil
}

// @todo dedup
func (d *Decoder) pack(rect *item.PlacedRect, segmentNode *segmentNode) error {
	// Set segment's coordinates to the rect.
	rect.X = segmentNode.data.x
	rect.Y = segmentNode.data.y

	switch {
	case rect.Item.Width == segmentNode.data.width:
		// Adjust the y-coordinate.
		segmentNode.data.y += rect.Item.Height

		// @todo dedup
		// If the segment isn't on the left edge of the strip,
		// adjust its left height and the right height of the prev segment.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segmentsList.head {
				return fmt.Errorf("corrupted left height of the very first segment")
			}
			segmentNode.data.lHeight += float64(rect.Item.Height)
			segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
		}

		// @todo dedup
		// If the segment isn't on the right edge of the strip,
		// adjust its right height and the left height of the next segment.
		if segmentNode.data.rHeight != math.Inf(-1) {
			if segmentNode == d.segmentsList.tail {
				return fmt.Errorf("corrupted right height of the very last segment")
			}
			segmentNode.data.rHeight += float64(rect.Item.Height)
			segmentNode.next.data.lHeight = -segmentNode.data.rHeight
		}
	case rect.Item.Width < segmentNode.data.width:
		// Create a new segment.
		s := newSegment(rect.Item.Width, segmentNode.data.x, segmentNode.data.y+rect.Item.Height)

		// @todo dedup if possible
		// If the segment isn't on the left edge of the strip,
		// adjust new segment's left height and the right height of the prev segment.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segmentsList.head {
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
		d.segmentsList.insertBefore(segmentNode, s)
	case rect.Item.Width > segmentNode.data.width:
		return fmt.Errorf("rect is wider than segment")
	}
	return nil
}
