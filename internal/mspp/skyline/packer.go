package skyline

import (
	"errors"
	"fmt"

	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/mspp"
)

// TODO Add tests (especially fuzzing).
// TODO Add GoDoc for all the written code.

var (
	ErrInvalidInput = errors.New("invalid input")
)

const notFoundIdx = -1

type Packer struct{}

func NewPacker() *Packer {
	return &Packer{}
}

func (p *Packer) Pack(rects []item.Rect, strips []mspp.Strip) (mspp.Solution, error) {
	if err := validate(rects, strips); err != nil {
		return mspp.Solution{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	skylines := make([]skyline, len(strips))
	var x int
	for i, strip := range strips {
		skylines[i] = newSkyline(x, strip.Width)
		x += strip.Width
	}

	packedRects, maxHeight := packRects(skylines, rects)

	return mspp.Solution{
		PackedRects: packedRects,
		FitVal:      maxHeight,
	}, nil
}

func validate(rects []item.Rect, strips []mspp.Strip) error {
	// Find width of the widest strip.
	var maxWidth int
	for _, s := range strips {
		if s.Width > maxWidth {
			maxWidth = s.Width
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

func packRects(skylines []skyline, rects []item.Rect) ([]item.PackedRect, int) {
	packedRects := make([]item.PackedRect, 0, len(rects))
	packedFlags := make([]bool, len(rects))
	unpackedIdxs := make([]int, len(rects))
	for i := range rects {
		unpackedIdxs[i] = i
	}

	var maxHeight int

	for packedCount := 0; packedCount != len(rects); {
		// Choose target segment.
		skylineIdx, segmentIdx := targetSegment(skylines)

		// Find best-fit rectangle to pack onto the target segment.
		segments := skylines[skylineIdx].segments
		unpackedIdx := bestFitUnpackedIdx(rects, unpackedIdxs, segments, segmentIdx)
		if unpackedIdx == notFoundIdx {
			skylines[skylineIdx].fillGap(segmentIdx)
			continue
		}
		rectIdx := unpackedIdxs[unpackedIdx]
		unpackedIdxs[unpackedIdx] = unpackedIdxs[len(unpackedIdxs)-1]
		unpackedIdxs = unpackedIdxs[:len(unpackedIdxs)-1]

		// Pack best-fit rectangle on the target segment.
		packedRect := packRect(rects[rectIdx], &skylines[skylineIdx], segmentIdx)
		packedRects = append(packedRects, packedRect)
		packedFlags[unpackedIdx] = true
		packedCount++

		if topY := packedRect.Y + packedRect.Height; topY > maxHeight {
			maxHeight = topY
		}
	}
	return packedRects, maxHeight
}

func bestFitUnpackedIdx(rects []item.Rect, unpackedIdxs []int, segments []segment, segmentIdx int) int {
	bestFitVal := -1
	bestIdx := notFoundIdx

	targetSegment := segments[segmentIdx]

	var lHeight, rHeight int
	hasLeft := segmentIdx != 0
	if hasLeft {
		lHeight = segments[segmentIdx-1].y - targetSegment.y
	}

	hasRight := segmentIdx != len(segments)-1
	if hasRight {
		rHeight = segments[segmentIdx+1].y - targetSegment.y
	}

	for i, rectIdx := range unpackedIdxs {
		// If the rect has already been packed or
		// wider than the segment's width, check next rectangle.
		if rects[rectIdx].Width > targetSegment.width {
			continue
		}

		// Declare fitness value for the current rectangle.
		fitVal := 0

		// Increment in the following cases:
		// 1. The rectangle height and the left segment height are equal.
		if hasLeft && rects[rectIdx].Height == lHeight {
			fitVal++
		}

		// 2. The rectangle height and the right segment height are equal.
		if hasRight && rects[rectIdx].Height == rHeight {
			fitVal++
		}

		// 3. The rectangle width and the segment width are equal.
		if rects[rectIdx].Width == targetSegment.width {
			fitVal++
		}

		if fitVal == 3 {
			return i
		}

		// Update the best fitness value and rectangle index.
		if fitVal > bestFitVal {
			bestFitVal = fitVal
			bestIdx = i
		}
	}
	return bestIdx
}

func packRect(rect item.Rect, skyline *skyline, segmentIdx int) item.PackedRect {
	segm := skyline.segments[segmentIdx]
	skyline.place(segmentIdx, rect.Width, rect.Height)

	return item.PackedRect{
		Rect: rect,
		X:    segm.x,
		Y:    segm.y,
	}
}

func targetSegment(skylines []skyline) (int, int) {
	lowestY := 0
	skylineIdx := -1
	var segmentIdx int
	for i, s := range skylines {
		for j, segm := range s.segments {
			if skylineIdx == -1 || segm.y < lowestY {
				lowestY = segm.y
				skylineIdx = i
				segmentIdx = j
			}
		}
	}
	return skylineIdx, segmentIdx
}
