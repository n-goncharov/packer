package skyline

import (
	"errors"
	"fmt"
	"slices"

	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/mspp"
)

// TODO Reconsider architecture of the entire project.
// TODO Add tests (especially fuzzing).
// TODO Add GoDoc for all the written code.

var (
	ErrInvalidInput = errors.New("invalid input")
)

const notFoundIdx = -1

type Packer struct {
	skylines []skyline
}

func NewPacker() *Packer {
	return &Packer{}
}

func (p *Packer) Pack(rects []item.Rect, strips []mspp.Strip) (mspp.Solution, error) {
	if err := validate(rects, strips); err != nil {
		return mspp.Solution{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	p.skylines = make([]skyline, len(strips))
	var x uint
	for i, strip := range strips {
		p.skylines[i] = newSkyline(x, strip.Width)
		x += strip.Width
	}

	return mspp.Solution{
		PackedRects: p.packRects(rects),
		FitVal:      p.packingHeight(),
	}, nil
}

func validate(rects []item.Rect, strips []mspp.Strip) error {
	// Find width of the widest strip.
	var maxWidth uint
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

func (p *Packer) packRects(rects []item.Rect) []item.PackedRect {
	packedRects := make([]item.PackedRect, 0, len(rects))
	packedFlags := make([]bool, len(rects))
	for packedCount := 0; packedCount != len(rects); {
		// Choose target segment.
		skylineIdx, segmentIdx := p.targetSegment()

		// Find best-fit rectangle to pack onto the target segment.
		bestFitIdx := p.bestFitRectIdx(rects, packedFlags, skylineIdx, segmentIdx)
		if bestFitIdx == notFoundIdx {
			p.skylines[skylineIdx].merge(segmentIdx)
			continue
		}

		// Pack best-fit rectangle on the target segment.
		packedRect := p.packRect(rects[bestFitIdx], skylineIdx, segmentIdx)
		packedRects = append(packedRects, packedRect)
		packedFlags[bestFitIdx] = true
		packedCount++
	}
	return packedRects
}

func (p *Packer) bestFitRectIdx(rects []item.Rect, packedFlags []bool, skylineIdx, segmentIdx int) int {
	bestFitVal := -1
	bestIdx := notFoundIdx
	skyline := p.skylines[skylineIdx]
	targetSegment := skyline.segments[segmentIdx]
	for i, r := range rects {
		// If the rect has already been packed or
		// wider than the segment's width, check next rectangle.
		if packedFlags[i] || r.Width > targetSegment.width {
			continue
		}

		// Declare fitness value for the current rectangle.
		fitVal := 0

		// Increment in the following cases:
		// 1. The rectangle height and the left segment height are equal.
		if segmentIdx != 0 && r.Height == skyline.segments[segmentIdx-1].y-targetSegment.y {
			fitVal++
		}

		// 2. The rectangle height and the right segment height are equal.
		if segmentIdx != len(skyline.segments)-1 && r.Height == skyline.segments[segmentIdx+1].y-targetSegment.y {
			fitVal++
		}

		// 3. The rectangle width and the segment width are equal.
		if r.Width == targetSegment.width {
			fitVal++
		}

		// Update the best fitness value and rectangle index.
		if fitVal > bestFitVal {
			bestFitVal = fitVal
			bestIdx = i
		}
	}
	return bestIdx
}

func (p *Packer) packRect(rect item.Rect, skylineIdx, segmentIdx int) item.PackedRect {
	skyline := &p.skylines[skylineIdx]
	currSegm := &skyline.segments[segmentIdx]
	packedRect := item.PackedRect{
		Rect: rect,
		X:    currSegm.x,
		Y:    currSegm.y,
	}

	switch {
	case packedRect.Width == currSegm.width:
		currSegm.y += rect.Height
	case rect.Width < currSegm.width:
		currSegm.x += rect.Width
		currSegm.width -= rect.Width
		skyline.segments = slices.Insert(
			skyline.segments,
			segmentIdx,
			segment{
				x:     currSegm.x,
				y:     currSegm.y + rect.Height,
				width: rect.Width,
			},
		)
	}
	return packedRect
}

func (p *Packer) targetSegment() (int, int) {
	lowestY := p.skylines[0].segments[0].y
	var skylineIdx, segmentIdx int
	for i, s := range p.skylines {
		for j, segm := range s.segments {
			if segm.y < lowestY {
				lowestY = segm.y
				skylineIdx = i
				segmentIdx = j
			}
		}
	}
	return skylineIdx, segmentIdx
}

func (p *Packer) packingHeight() uint {
	var height uint
	for _, s := range p.skylines {
		for _, seg := range s.segments {
			if seg.y > height {
				height = seg.y
			}
		}
	}
	return height
}
