package skyline

import (
	"fmt"
	"github.com/n-goncharov/packing-backend/internal/packing/container"
	"github.com/n-goncharov/packing-backend/internal/packing/decoder"
	"github.com/n-goncharov/packing-backend/internal/packing/shape"
	"math"
	"slices"
)

// @todo global - rectangle -> rect

type Segment struct {
	width            uint
	x, y             uint
	lHeight, rHeight float64
}

type Rect struct {
	shape.Rect
	X, Y   uint
	packed bool
}

func (r Rect) String() string {
	return fmt.Sprintf("Rect{Width: %d, Height: %d, X: %d, Y: %d}", r.Width, r.Height, r.X, r.Y)
}

func NewSegment(width, x, y uint) *Segment {
	return &Segment{
		width:   width,
		x:       x,
		y:       y,
		lHeight: math.Inf(-1),
		rHeight: math.Inf(-1),
	}
}

type Decoder struct {
	rects    []*Rect
	segments []*Segment
}

func validate(rects []*shape.Rect, strips []*container.Strip) error {
	var maxStripWidth uint
	for _, strip := range strips {
		if strip.Width > maxStripWidth {
			maxStripWidth = strip.Width
		}
	}

	for _, rect := range rects {
		if rect.Width > maxStripWidth {
			return fmt.Errorf("invalid input: width of rect is higher than the max strip width")
		}
	}

	return nil
}

// @todo add comments
func NewDecoder(rects []*shape.Rect, strips []*container.Strip) (*Decoder, error) {
	err := validate(rects, strips)
	if err != nil {
		return nil, err
	}

	packingRects := make([]*Rect, len(rects))
	for i, rect := range rects {
		packingRects[i] = &Rect{
			Rect: *rect,
		}
	}

	var x uint
	segments := make([]*Segment, len(strips))
	for i, strip := range strips {
		segments[i] = NewSegment(strip.Width, x, 0)
		x += strip.Width
	}

	return &Decoder{
		rects:    packingRects,
		segments: segments,
	}, nil
}

func (d *Decoder) Decode() (*decoder.Solution[*Rect], error) {
	// Pack rects until the number packed equals the total number.
	for packedCount := 0; packedCount != len(d.rects); {
		// Get the current left-bottom segment to pack a rect on it.
		lbSegmentIdx, err := d.getLBIdx()
		if err != nil {
			return nil, err
		}
		lbSegment := d.segments[lbSegmentIdx]

		// Get the best-fit rect to pack onto left-bottom segment.
		bestFitRect := d.getBestFit(lbSegment)

		// If no such rectangle was found = all of them are wider than left-bottom segment,
		// then lift left-bottom segment to the lowest neighboring segment and start new iteration.
		if bestFitRect == nil {
			err := d.liftLB(lbSegmentIdx)
			if err != nil {
				return nil, fmt.Errorf("error lifting LB segment: %w", err)
			}
			continue
		}

		// If best-fit rect was found, we pack it.
		err = d.pack(bestFitRect, lbSegmentIdx)
		if err != nil {
			return nil, err
		}

		// Mark the best-fit rect as packed and increment the counter of packed.
		bestFitRect.packed = true
		packedCount++
	}

	// Create a variable to fill it with packed rects (dereferencing pointers) and return from the method.
	packedItems := make([]*Rect, len(d.rects))
	for i, r := range d.rects {
		packedItems[i] = r
	}

	// Return a solution containing the value of the objective function and a slice of packed shape (rects).
	return &decoder.Solution[*Rect]{
		ObjFnVal:    int(d.topHeight()),
		PackedItems: packedItems,
	}, nil
}

func (d *Decoder) getLBIdx() (int, error) {
	idx := 0
	lbSegment := d.segments[idx]
	for i, segm := range d.segments {
		if segm.y < lbSegment.y {
			lbSegment = segm
			idx = i
		}
	}
	if lbSegment.lHeight >= 0 || lbSegment.rHeight >= 0 {
		return 0, fmt.Errorf("corrupted LB segment. lHeight: %f, rHeight: %f", lbSegment.lHeight, lbSegment.rHeight)
	}
	return idx, nil
}

// @todo check the logic
func (d *Decoder) getBestFit(lbSegment *Segment) *Rect {
	var bestFitRect *Rect
	bestFitnessValue := -1
	for _, r := range d.rects {
		// If the r has already been packed or does not fit the width on the segment, look next.
		if r.packed || lbSegment.width < r.Width {
			continue
		}

		// Declare (reset) fitnessValue for further increase.
		fitnessValue := 0

		// Increase by 1 if the left segment height and the r height are the same.
		if uint(math.Abs(lbSegment.lHeight)) == r.Height {
			fitnessValue++
		}

		// Increase by 1 if the right segment height and the r height are the same.
		if uint(math.Abs(lbSegment.rHeight)) == r.Height {
			fitnessValue++
		}

		// Increase by 1 if the width of the segment and r are the same.
		if lbSegment.width == r.Width {
			fitnessValue++
		}

		// If the current fitnessValue is greater than the global one, then update the global one with the current value
		// and memorize the index of the r.
		if fitnessValue > bestFitnessValue {
			bestFitnessValue = fitnessValue
			bestFitRect = r
		}
	}
	return bestFitRect
}

// @todo check the logic
func (d *Decoder) liftLB(lbSegmentIdx int) error {
	lbSegment := d.segments[lbSegmentIdx]

	// The following bool vars defines location of LB segment in the strip.
	leftMost := lbSegment.lHeight == math.Inf(-1)
	rightMost := lbSegment.rHeight == math.Inf(-1)

	// If the right one is the lowest - glue the segment to it, otherwise if the left one - to the left one.
	// If we raise it to the right, we already assume that there will be no strip edge on the right.
	// Similarly with the left one.
	// In the liftToRight method we pass information whether the segment is from the left edge of the strip.
	// To the liftToLeft method we pass information whether the segment is on the right edge of the strip.
	// Otherwise, there is an error.
	if lbSegment.lHeight < lbSegment.rHeight {
		err := d.liftToRight(lbSegmentIdx, leftMost)
		if err != nil {
			return err
		}
	} else {
		err := d.liftToLeft(lbSegmentIdx, rightMost)
		if err != nil {
			return err
		}
	}

	return nil
}

// @todo check the logic
func (d *Decoder) liftToRight(segmentIdx int, leftMost bool) error {
	// Here, just in case, we check if we are working with the last segment,
	// so that we don't go beyond the slice boundaries.
	if segmentIdx != len(d.segments)-1 {
		// If the segment is not on the left edge of the strip, we adjust its left height and, accordingly
		// the right height of the segment on the left.
		if !leftMost {
			// Here, just in case, we check if we are working with the first segment,
			// so that we don't go beyond the slice boundaries.
			if segmentIdx == 0 {
				return fmt.Errorf("adjusting the left neighbor of the first segment")
			}
			d.segments[segmentIdx].lHeight -= d.segments[segmentIdx].rHeight
			d.segments[segmentIdx-1].rHeight = -d.segments[segmentIdx].lHeight
		}

		// Here we adjust the width, y-coordinate and right height of the segment.
		d.segments[segmentIdx].width += d.segments[segmentIdx+1].width
		d.segments[segmentIdx].y = d.segments[segmentIdx+1].y
		d.segments[segmentIdx].rHeight = d.segments[segmentIdx+1].rHeight

		// Delete the segment following it.
		d.segments = slices.Delete(d.segments, segmentIdx+1, segmentIdx+2)
	} else {
		return fmt.Errorf("trying to lift to the right the last segment")
	}
	return nil
}

// @todo check the logic
func (d *Decoder) liftToLeft(segmentIdx int, rightMost bool) error {
	// Here, just in case, we check if we are working with the first segment,
	// so that we don't go beyond the slice boundaries.
	if segmentIdx != 0 {
		// If the segment is not on the right edge of the strip, we adjust its right height and, accordingly
		// the left height of the segment on the right.
		if !rightMost {
			// Here, just in case, we check if we are working with the last segment,
			// so that we don't go beyond the slice boundaries.
			if segmentIdx == len(d.segments)-1 {
				return fmt.Errorf("adjusting the right neighbor of the last segment")
			}
			d.segments[segmentIdx].rHeight -= d.segments[segmentIdx].lHeight
			d.segments[segmentIdx+1].lHeight = -d.segments[segmentIdx].rHeight
		}

		// Here we adjust the width, x-coordinate, y-coordinate and left height of the segment.
		d.segments[segmentIdx].width += d.segments[segmentIdx-1].width
		d.segments[segmentIdx].x = d.segments[segmentIdx-1].x
		d.segments[segmentIdx].y = d.segments[segmentIdx-1].y
		d.segments[segmentIdx].lHeight = d.segments[segmentIdx-1].lHeight

		// Delete the previous segment.
		d.segments = slices.Delete(d.segments, segmentIdx-1, segmentIdx)
	} else {
		return fmt.Errorf("trying to lift to the left the first segment")
	}
	return nil
}

func (d *Decoder) pack(rect *Rect, segmentIdx int) error {
	segment := d.segments[segmentIdx]

	// Set the segment's coordinates to the rect.
	rect.X = segment.x
	rect.Y = segment.y

	switch {
	case rect.Width == segment.width:
		// Adjust the y-coordinate.
		segment.y += rect.Height

		// Do if segment is not first in the strip.
		if segment.lHeight != math.Inf(-1) {
			if segmentIdx == 0 {
				return fmt.Errorf("corrupted lHeight for the very first segment")
			}

			// Adjust its lHeight.
			segment.lHeight += float64(rect.Height)

			// Adjust its left neighbour's rHeight.
			d.segments[segmentIdx-1].rHeight = -segment.lHeight
		}

		// Do if segment is not last in the strip.
		if segment.rHeight != math.Inf(-1) {
			if segmentIdx == len(d.segments)-1 {
				return fmt.Errorf("corrupted rHeight for the vary last segment")
			}

			// Adjust its rHeight.
			segment.rHeight += float64(rect.Height)

			// Adjust lHeight for the right neighbour.
			d.segments[segmentIdx+1].lHeight = -segment.rHeight
		}
	case rect.Width < segment.width:
		// Add new segment.
		newSegment := NewSegment(rect.Width, segment.x, segment.y+rect.Height)

		// Do if current segment is not first in the strip.
		if segment.lHeight != math.Inf(-1) {
			if segmentIdx == 0 {
				return fmt.Errorf("corrupted lHeight for the very first segment")
			}

			// Set lHeight for the new segment.
			newSegment.lHeight = segment.lHeight + float64(rect.Height)

			// Adjust rHeight for the left neighbour
			d.segments[segmentIdx-1].rHeight = -newSegment.lHeight
		}

		// Set rHeight for the new segment.
		newSegment.rHeight = float64(rect.Height)

		// Adjust current segment's fields.
		segment.x += rect.Width
		segment.width -= rect.Width
		segment.lHeight = -newSegment.rHeight

		// Insert new segment.
		d.segments = slices.Insert(d.segments, segmentIdx, newSegment)
	case rect.Width > segment.width:
		return fmt.Errorf("rect wider than segment")
	}
	return nil
}

func (d *Decoder) topHeight() uint {
	var topHeight uint
	for _, segment := range d.segments {
		if segment.y > topHeight {
			topHeight = segment.y
		}
	}
	return topHeight
}
