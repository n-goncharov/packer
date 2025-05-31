package skyline

import (
	"fmt"
	"github.com/n-goncharov/packing-backend/internal/packing/container"
	"github.com/n-goncharov/packing-backend/internal/packing/decoder"
	"github.com/n-goncharov/packing-backend/internal/packing/shape"
	"math"
)

// @todo move some code to separate files/possibly packages
// @todo possibly create const errors

// @todo consider rects to be doubly linked list too:
// to get rid of 'packed' bool field, just remove rect after packing
// and search best-fit through remaining - not all.

// @todo consider using just one Rect struct with Width, Height, X, Y if the above applies

// @todo abstract linked list code

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
	*shape.Rect
	X, Y   uint
	packed bool
}

func (r Rect) String() string {
	return fmt.Sprintf(
		"Rect{Width: %d, Height: %d, X: %d, Y: %d}",
		r.Width,
		r.Height,
		r.X,
		r.Y,
	)
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
	rects        []*Rect
	segmentsList *SegmentList
}

func NewDecoder(rects []*shape.Rect, strips []*container.Strip) (*Decoder, error) {
	// Validate bad input.
	err := validate(rects, strips)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		rects:        toPackingRects(rects),
		segmentsList: toSegmentsList(strips),
	}, nil
}

func (d *Decoder) Decode() (*decoder.Solution[*Rect], error) {
	// Create a slice for packed rects to save packing ordering.
	packedRects := make([]*Rect, 0, len(d.rects))

	// Pack rects until the number of packed equals the total.
	for packedCount := 0; packedCount != len(d.rects); {
		// Get the current left-bottom segment to pack a rect on it.
		lbSegmentNode, err := d.getLeftBottom()
		if err != nil {
			return nil, err
		}

		// Get the best-fit rect to pack onto left-bottom segment.
		bestFitRect := d.getBestFit(lbSegmentNode.data)

		// If no such rect was found = all of them are wider than the left-bottom segment,
		// then join left-bottom segment with the lowest neighboring segment
		// and start new iteration.
		if bestFitRect == nil {
			err := d.join(lbSegmentNode)
			if err != nil {
				return nil, fmt.Errorf("error joining LB segment: %w", err)
			}
			continue
		}

		// If the best-fit rect was found, pack it.
		err = d.pack(bestFitRect, lbSegmentNode)
		if err != nil {
			return nil, err
		}

		// Mark the best-fit rect as packed and increment the packed counter.
		bestFitRect.packed = true
		packedCount++

		// Add packed rect.
		packedRects = append(packedRects, bestFitRect)
	}

	// Return the objective function value and packed rects.
	return &decoder.Solution[*Rect]{
		ObjFnVal:    int(d.topHeight()),
		PackedItems: packedRects,
	}, nil
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
			return fmt.Errorf(
				"rectangle dimensions (%dx%d) are invalid: width must be ≤ %d",
				rect.Width,
				rect.Height,
				maxStripWidth,
			)
		}
	}

	return nil
}

func toPackingRects(rects []*shape.Rect) []*Rect {
	packingRects := make([]*Rect, len(rects))
	for i, rect := range rects {
		packingRects[i] = &Rect{
			Rect: rect,
		}
	}
	return packingRects
}

func toSegmentsList(strips []*container.Strip) *SegmentList {
	var x uint
	segmentList := &SegmentList{}
	for _, strip := range strips {
		segmentList.Append(NewSegment(strip.Width, x, 0))
		x += strip.Width
	}
	return segmentList
}

func (d *Decoder) getLeftBottom() (*SegmentNode, error) {
	lbSegment := d.segmentsList.head
	current := d.segmentsList.head.next
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

func (d *Decoder) getBestFit(segment *Segment) *Rect {
	var bestFitRect *Rect
	bestFitVal := -1
	for _, r := range d.rects {
		// If the rect has already been packed or
		// wider than the width of the segment, check next.
		if r.packed || r.Width > segment.width {
			continue
		}

		// Declare fitness value for the current rect.
		fitVal := 0

		// Increment if the rect height and the left segment height are equal.
		if r.Height == uint(math.Abs(segment.lHeight)) {
			fitVal++
		}

		// Increment if the rect height and the right segment height are equal.
		if r.Height == uint(math.Abs(segment.rHeight)) {
			fitVal++
		}

		// Increment if the rect width and the segment width are equal.
		if r.Width == segment.width {
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

func (d *Decoder) join(segmentNode *SegmentNode) error {
	// Check with which neighbouring segment to join.
	if segmentNode.data.lHeight < segmentNode.data.rHeight {
		return d.joinNext(segmentNode)
	} else {
		return d.joinPrev(segmentNode)
	}
}

func (d *Decoder) joinNext(segmentNode *SegmentNode) error {
	if segmentNode == d.segmentsList.tail {
		return fmt.Errorf("trying to join last segment with next")
	}

	// @todo remove duplication
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
	d.segmentsList.Delete(segmentNode.next)

	return nil
}

func (d *Decoder) joinPrev(segmentNode *SegmentNode) error {
	if segmentNode == d.segmentsList.head {
		return fmt.Errorf("trying to join first segment with prev")
	}

	// @todo remove duplication
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
	d.segmentsList.Delete(segmentNode.prev)

	return nil
}

func (d *Decoder) pack(rect *Rect, segmentNode *SegmentNode) error {
	// Set segment's coordinates to the rect.
	rect.X = segmentNode.data.x
	rect.Y = segmentNode.data.y

	switch {
	case rect.Width == segmentNode.data.width:
		// Adjust the y-coordinate.
		segmentNode.data.y += rect.Height

		// @todo remove duplication
		// If the segment isn't on the left edge of the strip,
		// adjust its left height and the right height of the prev segment.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segmentsList.head {
				return fmt.Errorf("corrupted left height of the very first segment")
			}
			segmentNode.data.lHeight += float64(rect.Height)
			segmentNode.prev.data.rHeight = -segmentNode.data.lHeight
		}

		// @todo remove duplication
		// If the segment isn't on the right edge of the strip,
		// adjust its right height and the left height of the next segment.
		if segmentNode.data.rHeight != math.Inf(-1) {
			if segmentNode == d.segmentsList.tail {
				return fmt.Errorf("corrupted right height of the very last segment")
			}
			segmentNode.data.rHeight += float64(rect.Height)
			segmentNode.next.data.lHeight = -segmentNode.data.rHeight
		}
	case rect.Width < segmentNode.data.width:
		// Create a new segment.
		newSegment := NewSegment(rect.Width, segmentNode.data.x, segmentNode.data.y+rect.Height)

		// @todo remove duplication if possible
		// If the segment isn't on the left edge of the strip,
		// adjust new segment's left height and the right height of the prev segment.
		if segmentNode.data.lHeight != math.Inf(-1) {
			if segmentNode == d.segmentsList.head {
				return fmt.Errorf("corrupted left height of the very first segment")
			}
			newSegment.lHeight = segmentNode.data.lHeight + float64(rect.Height)
			segmentNode.prev.data.rHeight = -newSegment.lHeight
		}

		// Set right height for the new segment.
		newSegment.rHeight = float64(rect.Height)

		// Adjust segment's x-coordinate, width, left height.
		segmentNode.data.x += rect.Width
		segmentNode.data.width -= rect.Width
		segmentNode.data.lHeight = -newSegment.rHeight

		// Insert new segment before current.
		d.segmentsList.InsertBefore(segmentNode, newSegment)
	case rect.Width > segmentNode.data.width:
		return fmt.Errorf("rect is wider than segment")
	}
	return nil
}

func (d *Decoder) topHeight() uint {
	var topHeight uint
	current := d.segmentsList.head
	for current != nil {
		if current.data.y > topHeight {
			topHeight = current.data.y
		}
		current = current.next
	}
	return topHeight
}
