package skyline

import (
	"fmt"
	"github.com/n-goncharov/packing-backend/internal/packing/container"
	"github.com/n-goncharov/packing-backend/internal/packing/decoder"
	"github.com/n-goncharov/packing-backend/internal/packing/shape"
	"math"
)

// @todo global - rectangle -> rect

type Segment struct {
	width            uint
	x, y             uint
	lHeight, rHeight float64
}

type SegmentNode struct {
	data *Segment
	next *SegmentNode
	prev *SegmentNode
}

type SegmentList struct {
	head *SegmentNode
	tail *SegmentNode
}

func (sl *SegmentList) Append(data *Segment) {
	newNode := &SegmentNode{data: data}

	if sl.head == nil {
		sl.head = newNode
		sl.tail = newNode
		return
	}

	newNode.prev = sl.tail
	sl.tail.next = newNode
	sl.tail = newNode
}

func (sl *SegmentList) InsertBefore(nextNode *SegmentNode, data *Segment) {
	if nextNode == nil {
		fmt.Println("next node is nil")
		return
	}

	newNode := &SegmentNode{
		data: data,
		prev: nextNode.prev,
		next: nextNode,
	}

	nextNode.prev = newNode

	if newNode.prev != nil {
		newNode.prev.next = newNode
	} else {
		sl.head = newNode
	}
}

func (sl *SegmentList) Delete(segmentNode *SegmentNode) {
	if segmentNode == nil {
		fmt.Println("segment node is nil")
		return
	}

	if segmentNode.prev == nil {
		sl.head = segmentNode.next
	} else {
		segmentNode.prev.next = segmentNode.next
	}

	if segmentNode.next == nil {
		sl.tail = segmentNode.prev
	} else {
		segmentNode.next.prev = segmentNode.prev
	}

	segmentNode.prev = nil
	segmentNode.next = nil
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
	segments *SegmentList
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
	segmentList := &SegmentList{}
	for _, strip := range strips {
		segmentList.Append(NewSegment(strip.Width, x, 0))
		x += strip.Width
	}

	return &Decoder{
		rects:    packingRects,
		segments: segmentList,
	}, nil
}

func (d *Decoder) Decode() (*decoder.Solution[*Rect], error) {
	// Pack rects until the number packed equals the total number.
	for packedCount := 0; packedCount != len(d.rects); {
		// Get the current left-bottom segment to pack a rect on it.
		lbSegmentNode, err := d.getLB()
		if err != nil {
			return nil, err
		}

		// Get the best-fit rect to pack onto left-bottom segment.
		bestFitRect := d.getBestFit(lbSegmentNode.data)

		// If no such rectangle was found = all of them are wider than left-bottom segment,
		// then lift left-bottom segment to the lowest neighboring segment and start new iteration.
		if bestFitRect == nil {
			err := d.liftLB(lbSegmentNode)
			if err != nil {
				return nil, fmt.Errorf("error lifting LB segment: %w", err)
			}
			continue
		}

		// If best-fit rect was found, we pack it.
		err = d.pack(bestFitRect, lbSegmentNode)
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

func (d *Decoder) getLB() (*SegmentNode, error) {
	lbSegment := d.segments.head
	current := d.segments.head.next
	for current != nil {
		if current.data.y < lbSegment.data.y {
			lbSegment = current
		}
		current = current.next
	}
	if lbSegment.data.lHeight >= 0 || lbSegment.data.rHeight >= 0 {
		return nil, fmt.Errorf(
			"corrupted LB segment. lHeight: %f, rHeight: %f",
			lbSegment.data.lHeight,
			lbSegment.data.rHeight,
		)
	}
	return lbSegment, nil
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
func (d *Decoder) liftLB(lbSegmentNode *SegmentNode) error {
	// The following bool vars defines location of LB segment in the strip.
	leftMost := lbSegmentNode.data.lHeight == math.Inf(-1)
	rightMost := lbSegmentNode.data.rHeight == math.Inf(-1)

	// If the right one is the lowest - glue the segment to it, otherwise if the left one - to the left one.
	// If we raise it to the right, we already assume that there will be no strip edge on the right.
	// Similarly with the left one.
	// In the liftToRight method we pass information whether the segment is from the left edge of the strip.
	// To the liftToLeft method we pass information whether the segment is on the right edge of the strip.
	// Otherwise, there is an error.
	if lbSegmentNode.data.lHeight < lbSegmentNode.data.rHeight {
		err := d.liftToRight(lbSegmentNode, leftMost)
		if err != nil {
			return err
		}
	} else {
		err := d.liftToLeft(lbSegmentNode, rightMost)
		if err != nil {
			return err
		}
	}

	return nil
}

// @todo check the logic
func (d *Decoder) liftToRight(segmentNode *SegmentNode, leftMost bool) error {
	// Here, just in case, we check if we are working with the last segment,
	// so that we don't go beyond the slice boundaries.
	if segmentNode != d.segments.tail {
		// If the segment is not on the left edge of the strip, we adjust its left height and, accordingly
		// the right height of the segment on the left.
		if !leftMost {
			// Here, just in case, we check if we are working with the first segment,
			// so that we don't go beyond the slice boundaries.
			if segmentNode == d.segments.head {
				return fmt.Errorf("adjusting the left neighbor of the first segment")
			}
			segmentNode.data.lHeight -= segmentNode.data.rHeight
			segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
		}

		// Here we adjust the width, y-coordinate and right height of the segment.
		segmentNode.data.width += segmentNode.next.data.width
		segmentNode.data.y = segmentNode.next.data.y
		segmentNode.data.rHeight = segmentNode.next.data.rHeight

		// Delete the segment following it.
		d.segments.Delete(segmentNode.next)
	} else {
		return fmt.Errorf("trying to lift to the right the last segment")
	}
	return nil
}

// @todo check the logic
func (d *Decoder) liftToLeft(segmentNode *SegmentNode, rightMost bool) error {
	// Here, just in case, we check if we are working with the first segment,
	// so that we don't go beyond the slice boundaries.
	if segmentNode != d.segments.head {
		// If the segment is not on the right edge of the strip, we adjust its right height and, accordingly
		// the left height of the segment on the right.
		if !rightMost {
			// Here, just in case, we check if we are working with the last segment,
			// so that we don't go beyond the slice boundaries.
			if segmentNode == d.segments.tail {
				return fmt.Errorf("adjusting the right neighbor of the last segment")
			}
			segmentNode.data.rHeight -= segmentNode.data.lHeight
			segmentNode.next.data.lHeight = -segmentNode.data.rHeight
		}

		// Here we adjust the width, x-coordinate, y-coordinate and left height of the segment.
		segmentNode.data.width += segmentNode.prev.data.width
		segmentNode.data.x = segmentNode.prev.data.x
		segmentNode.data.y = segmentNode.prev.data.y
		segmentNode.data.lHeight = segmentNode.prev.data.lHeight

		// Delete the previous segment.
		d.segments.Delete(segmentNode.prev)
	} else {
		return fmt.Errorf("trying to lift to the left the first segment")
	}
	return nil
}

func (d *Decoder) pack(rect *Rect, segmentNode *SegmentNode) error {
	// Set the segment's coordinates to the rect.
	rect.X = segmentNode.data.x
	rect.Y = segmentNode.data.y

	switch {
	case rect.Width == segmentNode.data.width:
		// Adjust the y-coordinate.
		segmentNode.data.y += rect.Height

		// Do if segment is not first in the strip.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segments.head {
				return fmt.Errorf("corrupted lHeight for the very first segment")
			}

			// Adjust its lHeight.
			segmentNode.data.lHeight += float64(rect.Height)

			// Adjust its left neighbour's rHeight.
			segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
		}

		// Do if segment is not last in the strip.
		if segmentNode.data.rHeight != math.Inf(-1) {
			if segmentNode == d.segments.tail {
				return fmt.Errorf("corrupted rHeight for the vary last segment")
			}

			// Adjust its rHeight.
			segmentNode.data.rHeight += float64(rect.Height)

			// Adjust lHeight for the right neighbour.
			segmentNode.next.data.lHeight = -segmentNode.data.rHeight
		}
	case rect.Width < segmentNode.data.width:
		// Add new segment.
		newSegment := NewSegment(rect.Width, segmentNode.data.x, segmentNode.data.y+rect.Height)

		// Do if current segment is not first in the strip.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segments.head {
				return fmt.Errorf("corrupted lHeight for the very first segment")
			}

			// Set lHeight for the new segment.
			newSegment.lHeight = segmentNode.data.lHeight + float64(rect.Height)

			// Adjust rHeight for the left neighbour
			segmentNode.prev.data.rHeight = -newSegment.lHeight
		}

		// Set rHeight for the new segment.
		newSegment.rHeight = float64(rect.Height)

		// Adjust current segment's fields.
		segmentNode.data.x += rect.Width
		segmentNode.data.width -= rect.Width
		segmentNode.data.lHeight = -newSegment.rHeight

		// Insert new segment.
		d.segments.InsertBefore(segmentNode, newSegment)
	case rect.Width > segmentNode.data.width:
		return fmt.Errorf("rect wider than segment")
	}
	return nil
}

func (d *Decoder) topHeight() uint {
	var topHeight uint
	current := d.segments.head
	for current != nil {
		if current.data.y > topHeight {
			topHeight = current.data.y
		}
		current = current.next
	}
	return topHeight
}
