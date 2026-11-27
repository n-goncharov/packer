package skyline

import (
	"slices"
)

func (s *skyline) merge(segmentIdx int) {
	// If segment is a strip, we cannot merge it then delete this segment.
	if len(s.segments) == 1 {
		s.segments = slices.Delete(s.segments, 0, 1)
		return
	}

	// Merge with the lowest neighboring segment.
	segment := &s.segments[segmentIdx]

	var lHeight uint
	if segmentIdx != 0 {
		lHeight = s.segments[segmentIdx-1].y - segment.y
	}

	var rHeight uint
	if segmentIdx != len(s.segments)-1 {
		rHeight = s.segments[segmentIdx+1].y - segment.y
	}

	if segmentIdx != 0 && (segmentIdx == len(s.segments)-1 || lHeight <= rHeight) {
		prevSegment := s.segments[segmentIdx-1]
		segment.width += prevSegment.width
		segment.x = prevSegment.x
		segment.y = prevSegment.y
		s.segments = slices.Delete(s.segments, segmentIdx-1, segmentIdx)
		return
	}

	nextSegment := s.segments[segmentIdx+1]
	segment.width += nextSegment.width
	segment.y = nextSegment.y
	s.segments = slices.Delete(s.segments, segmentIdx+1, segmentIdx+2)
}
