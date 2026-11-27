package skyline

import (
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/list"
)

const notFoundIdx = -1

type segm struct {
	x, y, width uint
	_           struct{}
}

type segmList struct {
	list.DoublyList[*segm]
}

func newSegmList(x, width uint) *segmList {
	sl := &segmList{}
	sl.PushBack(&segm{x: x, width: width})
	return sl
}

func bestFitRectIdx(rects []*item.Rect, packedFlags []bool, targetSegmNode *list.Node[*segm]) int {
	bestFitVal := -1
	bestIdx := notFoundIdx
	targetSegm := targetSegmNode.Value
	for i, r := range rects {
		// If the rect has already been packed or
		// wider than the segment's width, check next rectangle.
		if packedFlags[i] || r.Width > targetSegm.width {
			continue
		}

		// Declare fitness value for the current rectangle.
		fitVal := 0

		// Increment in the following cases:
		// 1. The rectangle height and the left segment height are equal.
		if targetSegmNode.Prev != nil && r.Height == targetSegmNode.Prev.Value.y-targetSegm.y {
			fitVal++
		}

		// 2. The rectangle height and the right segment height are equal.
		if targetSegmNode.Next != nil && r.Height == targetSegmNode.Next.Value.y-targetSegm.y {
			fitVal++
		}

		// 3. The rectangle width and the segment width are equal.
		if r.Width == targetSegm.width {
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

func (sl *segmList) merge(segmNode *list.Node[*segm]) {
	// If segment is a strip, we cannot merge it then delete this segment.
	if segmNode.Prev == nil && segmNode.Next == nil {
		sl.Remove(segmNode)
		return
	}

	// Merge with the lowest neighboring segment.
	segm := segmNode.Value

	var lHeight uint
	if segmNode.Prev != nil {
		lHeight = segmNode.Prev.Value.y - segm.y
	}

	var rHeight uint
	if segmNode.Next != nil {
		rHeight = segmNode.Next.Value.y - segm.y
	}

	if segmNode.Prev != nil && (segmNode.Next == nil || lHeight <= rHeight) {
		segm.width += segmNode.Prev.Value.width
		segm.x = segmNode.Prev.Value.x
		segm.y = segmNode.Prev.Value.y
		sl.Remove(segmNode.Prev)
		return
	}

	segm.width += segmNode.Next.Value.width
	segm.y = segmNode.Next.Value.y
	sl.Remove(segmNode.Next)
}

func (sl *segmList) packRect(rect *item.Rect, segmNode *list.Node[*segm]) *item.PackedRect {
	currSegm := segmNode.Value
	packedRect := &item.PackedRect{
		Rect: *rect,
		X:    currSegm.x,
		Y:    currSegm.y,
	}

	switch {
	case packedRect.Width == currSegm.width:
		currSegm.y += rect.Height
	case rect.Width < currSegm.width:
		newSegm := &segm{x: currSegm.x, y: currSegm.y + rect.Height, width: rect.Width}
		currSegm.x += rect.Width
		currSegm.width -= rect.Width
		sl.InsertBefore(segmNode, newSegm)
	}
	return packedRect
}
