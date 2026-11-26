package skyline

import (
	"errors"
	"fmt"
	"math"

	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/list"
)

const notFoundIdx = -1

type segm struct {
	width            uint
	x, y             uint
	lHeight, rHeight float64 // todo do we need these values? I don't think so. Consider checking next and prev segm on nil value
}

func newSegm(width, x, y uint) *segm {
	return &segm{
		width:   width,
		x:       x,
		y:       y,
		lHeight: math.Inf(-1),
		rHeight: math.Inf(-1),
	}
}

type segmList struct {
	list.DoublyList[*segm]
}

func newSegmList(strips []*container.Strip) *segmList {
	var x uint
	sl := &segmList{}
	for _, strip := range strips {
		sl.PushBack(newSegm(strip.Width, x, 0))
		x += strip.Width
	}
	return sl
}

func (sl *segmList) height() uint {
	var height uint
	curr := sl.Head
	for curr != nil {
		if curr.Value.y > height {
			height = curr.Value.y
		}
		curr = curr.Next
	}
	return height
}

func (sl *segmList) pack(rects []*item.Rect) ([]*item.PackedRect, error) {
	packedRects := make([]*item.PackedRect, 0, len(rects))
	packedFlags := make([]bool, len(rects))
	for packedCount := 0; packedCount != len(rects); {
		lbSegmNode, err := sl.lbSegmNode()
		if err != nil {
			return nil, err
		}

		// Get the best-fit rect to pack onto left-bottom segment.
		// If no such rect is found then all rects are wider than that segment.
		// In this case merge LB-segment with the lowest neighboring and start new iteration.
		bestFitIdx := bestFitIdx(rects, packedFlags, lbSegmNode.Value)
		if bestFitIdx == notFoundIdx {
			if err := sl.merge(lbSegmNode); err != nil {
				return nil, fmt.Errorf("error merging LB segment: %v", err)
			}
			continue
		}
		packedFlags[bestFitIdx] = true

		// If the best-fit rect is found, pack it.
		packedRect, err := sl.packSingle(rects[bestFitIdx], lbSegmNode)
		if err != nil {
			return nil, fmt.Errorf("error packing rect: %v", err)
		}

		packedRects = append(packedRects, packedRect)
		packedCount++
	}
	return packedRects, nil
}

func (sl *segmList) lbSegmNode() (*list.Node[*segm], error) {
	lbSegmNode := sl.Head
	curr := sl.Head.Next
	for curr != nil {
		if curr.Value.y < lbSegmNode.Value.y {
			lbSegmNode = curr
		}
		curr = curr.Next
	}

	// TODO - Consider moving this check to the tests later and remove error return
	// Here we are not comparing r.Height with zero
	// because there could be merge with previous segment but next was the same height.
	if lbSegmNode.Value.lHeight >= 0 || lbSegmNode.Value.rHeight > 0 {
		return nil, fmt.Errorf(
			"corrupted LB segment\n lHeight: %f, rHeight: %f",
			lbSegmNode.Value.lHeight,
			lbSegmNode.Value.rHeight,
		)
	}
	return lbSegmNode, nil
}

func bestFitIdx(rects []*item.Rect, packedFlags []bool, targetSegm *segm) int {
	bestFitVal := -1
	bestFitIdx := notFoundIdx
	for i, r := range rects {
		// If the rect has already been packed or
		// wider than the segment's width, check next.
		if packedFlags[i] || r.Width > targetSegm.width {
			continue
		}

		// Declare fitness value for the current rect.
		fitVal := 0

		// @todo should lHeight and rHeight be validated here (error if they are > (=?) 0)?
		// Increment in the following cases:
		// The rect height and the left segment height are equal.
		if r.Height == uint(targetSegm.lHeight) {
			fitVal++
		}

		// The rect height and the right segment height are equal.
		if r.Height == uint(targetSegm.rHeight) {
			fitVal++
		}

		// The rect width and the segment width are equal.
		if r.Width == targetSegm.width {
			fitVal++
		}

		// Update the best fitness value.
		if fitVal > bestFitVal {
			bestFitVal = fitVal
			bestFitIdx = i
		}
	}
	return bestFitIdx
}

func (sl *segmList) adjustLHeight(node *list.Node[*segm], height float64) error {
	segm := node.Value
	if math.IsInf(segm.lHeight, -1) {
		return nil
	}

	// If the segment isn't on the left edge of the strip,
	// adjust its left height and the right height of the prev segment.
	if node == sl.Head {
		return errors.New("corrupted left height of the very first segment")
	}
	segm.lHeight += height
	node.Prev.Value.rHeight = -segm.lHeight
	return nil
}

func (sl *segmList) adjustRHeight(node *list.Node[*segm], height float64) error {
	segm := node.Value
	if math.IsInf(segm.rHeight, -1) {
		return nil
	}

	// If the segment isn't on the right edge of the strip,
	// adjust its right height and the left height of the next segment.
	if node == sl.Tail {
		return errors.New("corrupted right height of the very last segment")
	}
	segm.rHeight += height
	node.Next.Value.lHeight = -segm.rHeight

	return nil
}

func (sl *segmList) mergeNext(node *list.Node[*segm]) error {
	segm := node.Value
	if node == sl.Tail {
		return errors.New("trying to merge last segment with next")
	}

	if err := sl.adjustLHeight(node, -segm.rHeight); err != nil {
		return err
	}

	// Adjust the width, y-coordinate, right height of the segment.
	segm.width += node.Next.Value.width
	segm.y = node.Next.Value.y
	segm.rHeight = node.Next.Value.rHeight

	// Delete the next segment.
	sl.Remove(node.Next)

	return nil
}

func (sl *segmList) mergePrev(node *list.Node[*segm]) error {
	segm := node.Value
	if node == sl.Head {
		return errors.New("trying to merge first segment with prev")
	}

	if err := sl.adjustRHeight(node, -segm.lHeight); err != nil {
		return err
	}

	// Adjust the width, x-coordinate, y-coordinate, left height of the segment.
	segm.width += node.Prev.Value.width
	segm.x = node.Prev.Value.x
	segm.y = node.Prev.Value.y
	segm.lHeight = node.Prev.Value.lHeight

	// Delete the prev segment.
	sl.Remove(node.Prev)

	return nil
}

func (sl *segmList) merge(node *list.Node[*segm]) error {
	segm := node.Value
	// In case segment is a strip, and we cannot merge it -> just delete this segment.
	if math.IsInf(segm.lHeight, -1) && math.IsInf(node.Value.rHeight, -1) {
		sl.Remove(node)
		return nil
	}

	// Check with which neighboring segment to merge.
	if segm.lHeight < segm.rHeight {
		return sl.mergeNext(node)
	}
	return sl.mergePrev(node)
}

func (sl *segmList) packSingle(rect *item.Rect, node *list.Node[*segm]) (*item.PackedRect, error) {
	segm := node.Value
	packedRect := &item.PackedRect{
		Rect: *rect,
		X:    segm.x,
		Y:    segm.y,
	}

	switch {
	case packedRect.Width == segm.width:
		// Adjust the y-coordinate.
		segm.y += rect.Height

		if err := sl.adjustLHeight(node, float64(rect.Height)); err != nil {
			return nil, err
		}

		if err := sl.adjustRHeight(node, float64(rect.Height)); err != nil {
			return nil, err
		}

	case rect.Width < segm.width:
		// Create a new segment.
		s := newSegm(rect.Width, segm.x, segm.y+rect.Height)

		// If the segment isn't on the left edge of the strip,
		// adjust new segment's left height and the right height of the prev segment.
		if !math.IsInf(segm.lHeight, -1) {
			if segm == sl.Head.Value {
				return nil, errors.New("corrupted left height of the very first segment")
			}
			s.lHeight = segm.lHeight + float64(rect.Height)
			node.Prev.Value.rHeight = -s.lHeight
		}

		// Set right height for the new segment.
		s.rHeight = float64(rect.Height)

		// Adjust segment's x-coordinate, width, left height.
		segm.x += rect.Width
		segm.width -= rect.Width
		segm.lHeight = -s.rHeight

		// Insert new segment before current.
		sl.InsertBefore(node, s)
	case rect.Width > segm.width:
		return nil, fmt.Errorf("rect is wider than segment")
	}
	return packedRect, nil
}
