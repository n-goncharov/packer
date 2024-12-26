package packing

import (
	"fmt"
	"math"
	"slices"
)

type Skyline struct {
	rectangles []*Rectangle
	segments   []*Segment
}

type Segment struct {
	width            uint
	x, y             uint
	lHeight, rHeight float64
}

type Rectangle struct {
	Width, Height uint
	X, Y          uint
	packed        bool
}

func (r Rectangle) String() string {
	return fmt.Sprintf("Rectangle{Width: %d, Height: %d, X: %d, Y: %d}", r.Width, r.Height, r.X, r.Y)
}

type Strip struct {
	Width uint
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

func (s *Skyline) Decode(rectangles []Rectangle, strips []Strip) (*Solution[Rectangle], error) {
	// Here we fill the rectangles field - a slice with pointers to rectangles.
	// This is necessary so that we can modify rectangles directly in the Skyline structure.
	for _, rectangle := range rectangles {
		s.rectangles = append(s.rectangles, &rectangle)
	}

	// Here we fill the segments field - a slice with pointers to segments.
	// Similarly, pointers are needed to change the original elements in the slice.
	// Initially (in an empty packaging) segments are the bottom of each strip.
	var x uint = 0
	for _, strip := range strips {
		s.segments = append(s.segments, NewSegment(strip.Width, x, 0))
		x += strip.Width
	}

	// We pack rectangles in the loop.
	// The stopping condition is that the number of packed rectangles equals the number of rectangles.
	packedCount := 0
	for packedCount < len(rectangles) {
		// Looking for the most left-bottom segment to pack some rectangle on it.
		lbSegmentIdx := s.findLBSegmentIdx()

		// Now we need to find the best fitting rectangle to pack onto this segment.
		rectangleToPackIdx := s.findBestFitting(lbSegmentIdx)

		// If no such rectangle was found (all of them are wider than this segment), then
		// glue this segment with the lowest neighboring segment and start iteration again.
		if rectangleToPackIdx == -1 {
			err := s.liftLBSegment(lbSegmentIdx)
			if err != nil {
				return nil, fmt.Errorf("liftLBSegment: %w %d", err, packedCount)
			}
			continue
		}

		// If a rectangle was found, we pack it.
		err := s.pack(rectangleToPackIdx, lbSegmentIdx)
		if err != nil {
			return nil, err
		}

		// Mark the rectangle packed and increment the counter of packed rectangles.
		s.rectangles[rectangleToPackIdx].packed = true
		packedCount++
	}

	// Create a variable to fill it with packed rectangles (dereferencing pointers) and return from the method.
	packedItems := make([]Rectangle, len(s.rectangles))
	for i, rectanglePtr := range s.rectangles {
		packedItems[i] = *rectanglePtr
	}

	// Return a solution containing the value of the objective function and a slice of packed items (rectangles).
	return &Solution[Rectangle]{
		ObjFuncVal:  int(s.topHeight()),
		PackedItems: packedItems,
	}, nil
}

func (s *Skyline) findLBSegmentIdx() int {
	var lbSegmentIdx int
	lbSegment := s.segments[lbSegmentIdx]
	for i, segment := range s.segments {
		if segment.y < lbSegment.y {
			lbSegment = segment
			lbSegmentIdx = i
		}
	}
	return lbSegmentIdx
}

func (s *Skyline) findBestFitting(lbSegmentIdx int) int {
	lbSegment := s.segments[lbSegmentIdx]
	bestFittingIdx := -1
	bestFitnessValue := -1
	for i, rectangle := range s.rectangles {
		// If the rectangle has already been packed or does not fit the width on the segment, look next.
		if rectangle.packed || lbSegment.width < rectangle.Width {
			continue
		}

		// Declare (reset) fitnessValue for further increase.
		fitnessValue := 0

		// Increase by 1 if the left segment height and the rectangle height are the same.
		if uint(math.Abs(lbSegment.lHeight)) == rectangle.Height {
			fitnessValue++
		}

		// Increase by 1 if the right segment height and the rectangle height are the same.
		if uint(math.Abs(lbSegment.rHeight)) == rectangle.Height {
			fitnessValue++
		}

		// Increase by 1 if the width of the segment and rectangle are the same.
		if lbSegment.width == rectangle.Width {
			fitnessValue++
		}

		// If the current fitnessValue is greater than the global one, then update the global one with the current value
		// and memorize the index of the rectangle.
		if fitnessValue > bestFitnessValue {
			bestFitnessValue = fitnessValue
			bestFittingIdx = i
		}
	}
	return bestFittingIdx
}

func (s *Skyline) liftLBSegment(lbSegmentIdx int) error {
	lbSegment := s.segments[lbSegmentIdx]

	// Just in case the neighbors are higher up.
	if lbSegment.lHeight >= 0 || lbSegment.rHeight > 0 {
		return fmt.Errorf("segment is not LB")
	}

	// Is the leftmost in the strip.
	leftMost := lbSegment.lHeight == math.Inf(-1)
	// Is the rightmost in the strip.
	rightMost := lbSegment.rHeight == math.Inf(-1)
	// The lowest neighbor is the right one.
	lowestIsRight := lbSegment.lHeight < lbSegment.rHeight
	// The lowest neighbor is the left one.
	lowestIsLeft := lbSegment.lHeight >= lbSegment.rHeight

	// Here we check to see if we are trying to lift a segment with a full strip width.
	// This can occur if the width of the remaining rectangles is greater than even the width of the strip.
	if leftMost && rightMost {
		return fmt.Errorf("trying to lift the whole strip for segment %d", lbSegmentIdx)
	}

	// If the right one is the lowest - glue the segment to it, otherwise if the left one - to the left one.
	// If we raise it to the right, we already assume that there will be no strip edge on the right.
	// Similarly with the left one.
	// In the liftToRight method we pass information whether the segment is from the left edge of the strip.
	// To the liftToLeft method we pass information whether the segment is on the right edge of the strip.
	// Otherwise, there is an error.
	if lowestIsRight {
		err := s.liftToRight(lbSegmentIdx, leftMost)
		if err != nil {
			return err
		}
	} else if lowestIsLeft {
		err := s.liftToLeft(lbSegmentIdx, rightMost)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("error with lifting segment")
	}
	return nil
}

func (s *Skyline) liftToRight(segmentIdx int, leftMost bool) error {
	// Here, just in case, we check if we are working with the last segment,
	// so that we don't go beyond the slice boundaries.
	if segmentIdx != len(s.segments)-1 {
		// If the segment is not on the left edge of the strip, we adjust its left height and, accordingly
		// the right height of the segment on the left.
		if !leftMost {
			// Here, just in case, we check if we are working with the first segment,
			// so that we don't go beyond the slice boundaries.
			if segmentIdx == 0 {
				return fmt.Errorf("adjusting the left neighbor of the first segment")
			}
			s.segments[segmentIdx].lHeight -= s.segments[segmentIdx].rHeight
			s.segments[segmentIdx-1].rHeight = -s.segments[segmentIdx].lHeight
		}

		// Here we adjust the width, y-coordinate and right height of the segment.
		s.segments[segmentIdx].width += s.segments[segmentIdx+1].width
		s.segments[segmentIdx].y = s.segments[segmentIdx+1].y
		s.segments[segmentIdx].rHeight = s.segments[segmentIdx+1].rHeight

		// Delete the segment following it.
		s.segments = slices.Delete(s.segments, segmentIdx+1, segmentIdx+2)
	} else {
		return fmt.Errorf("trying to lift to the right the last segment")
	}
	return nil
}

func (s *Skyline) liftToLeft(segmentIdx int, rightMost bool) error {
	// Here, just in case, we check if we are working with the first segment,
	// so that we don't go beyond the slice boundaries.
	if segmentIdx != 0 {
		// If the segment is not on the right edge of the strip, we adjust its right height and, accordingly
		// the left height of the segment on the right.
		if !rightMost {
			// Here, just in case, we check if we are working with the last segment,
			// so that we don't go beyond the slice boundaries.
			if segmentIdx == len(s.segments)-1 {
				return fmt.Errorf("adjusting the right neighbor of the last segment")
			}
			s.segments[segmentIdx].rHeight -= s.segments[segmentIdx].lHeight
			s.segments[segmentIdx+1].lHeight = -s.segments[segmentIdx].rHeight
		}

		// Here we adjust the width, x-coordinate, y-coordinate and left height of the segment.
		s.segments[segmentIdx].width += s.segments[segmentIdx-1].width
		s.segments[segmentIdx].x = s.segments[segmentIdx-1].x
		s.segments[segmentIdx].y = s.segments[segmentIdx-1].y
		s.segments[segmentIdx].lHeight = s.segments[segmentIdx-1].lHeight

		// Delete the previous segment.
		s.segments = slices.Delete(s.segments, segmentIdx-1, segmentIdx)
	} else {
		return fmt.Errorf("trying to lift to the left the first segment")
	}
	return nil
}

func (s *Skyline) pack(rectangleIdx, segmentIdx int) error {
	rectangle := s.rectangles[rectangleIdx]
	segment := s.segments[segmentIdx]

	// Set the segment's coordinates to the rectangle.
	rectangle.X = segment.x
	rectangle.Y = segment.y

	// If the width of the rectangle is equal to the width of the strip, adjust the y-coordinate of the segment,
	// lHeight, rHeight; and neighboring ones.
	if rectangle.Width == segment.width {
		segment.y += rectangle.Height

		if segment.lHeight != math.Inf(-1) {
			segment.lHeight += float64(rectangle.Height)
			// Here, just in case, we check if we are working with the first segment,
			// so that we don't go beyond the slice boundaries.
			if segmentIdx != 0 {
				s.segments[segmentIdx-1].rHeight = -segment.lHeight
			} else {
				return fmt.Errorf("trying to adjust left neighbor of the first segment")
			}
		}

		if segment.rHeight != math.Inf(-1) {
			segment.rHeight += float64(rectangle.Height)
			// Here, just in case, we check if we are working with the last segment,
			// so that we don't go beyond the slice boundaries.
			if segmentIdx != len(s.segments)-1 {
				s.segments[segmentIdx+1].lHeight = -segment.rHeight
			} else {
				return fmt.Errorf("trying to adjust right neighbor of the last segment")
			}
		}
	} else if rectangle.Width < segment.width {
		// Otherwise add new segment and adjust neighboring (left and current) segments.
		newSegment := NewSegment(rectangle.Width, segment.x, segment.y+rectangle.Height)
		if segment.lHeight != math.Inf(-1) {
			newSegment.lHeight = segment.lHeight + float64(rectangle.Height)
			if segmentIdx != 0 {
				s.segments[segmentIdx-1].rHeight = -newSegment.lHeight
			} else {
				return fmt.Errorf("trying to adjust left neighbor of the first segment")
			}
		}
		newSegment.rHeight = float64(rectangle.Height)

		segment.x += rectangle.Width
		segment.width -= rectangle.Width
		segment.lHeight = -newSegment.rHeight

		s.segments = slices.Insert(s.segments, segmentIdx, newSegment)
	} else if rectangle.Width > segment.width {
		return fmt.Errorf("rectangle wider than segment")
	}
	return nil
}

func (s *Skyline) topHeight() uint {
	var topHeight uint
	for _, segment := range s.segments {
		if segment.y > topHeight {
			topHeight = segment.y
		}
	}
	return topHeight
}
