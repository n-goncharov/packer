package skyline

import (
	"slices"
)

type skyline struct {
	segments []segment
}

func newSkyline(x, width int) skyline {
	return skyline{
		segments: []segment{
			{
				x:     x,
				width: width,
			},
		},
	}
}

func (s *skyline) fillGap(segmentIdx int) {
	if len(s.segments) == 1 {
		s.segments = slices.Delete(s.segments, 0, 1)
		return
	}

	curr := s.segments[segmentIdx]
	hasPrev := segmentIdx != 0
	hasNext := segmentIdx != len(s.segments)-1

	var targetY int
	if hasPrev && (!hasNext || s.segments[segmentIdx-1].y <= s.segments[segmentIdx+1].y) {
		targetY = s.segments[segmentIdx-1].y
	} else {
		targetY = s.segments[segmentIdx+1].y
	}

	s.place(segmentIdx, curr.width, targetY-curr.y)
}

func (s *skyline) place(segmentIdx int, width, height int) {
	currSegm := &s.segments[segmentIdx]
	switch {
	case width == currSegm.width:
		currSegm.y += height
	case width < currSegm.width:
		origX, origY := currSegm.x, currSegm.y
		currSegm.x += width
		currSegm.width -= width
		s.segments = slices.Insert(
			s.segments,
			segmentIdx,
			segment{
				x:     origX,
				y:     origY + height,
				width: width,
			},
		)
	}

	curr := &s.segments[segmentIdx]
	if segmentIdx != len(s.segments)-1 && s.segments[segmentIdx+1].y == curr.y {
		curr.width += s.segments[segmentIdx+1].width
		s.segments = slices.Delete(s.segments, segmentIdx+1, segmentIdx+2)
	}

	if segmentIdx != 0 && s.segments[segmentIdx-1].y == curr.y {
		curr.x = s.segments[segmentIdx-1].x
		curr.width += s.segments[segmentIdx-1].width
		s.segments = slices.Delete(s.segments, segmentIdx-1, segmentIdx)
	}
}
