package skyline

import (
	"fmt"
	"math"
)

// @todo start to add godoc

// @todo make rects to be doubly linked list too:
// to get rid of 'Packed' bool field (just remove rect after packing)
// and search best-fit through remaining list - not all rects.

type segment struct {
	width            uint
	x, y             uint
	lHeight, rHeight float64
}

func newSegment(width, x, y uint) *segment {
	return &segment{
		width:   width,
		x:       x,
		y:       y,
		lHeight: math.Inf(-1),
		rHeight: math.Inf(-1),
	}
}

// @todo use container/list instead
type segmentNode struct {
	data *segment
	next *segmentNode
	prev *segmentNode
}

// @todo use container/list instead
type segmentList struct {
	head *segmentNode
	tail *segmentNode
}

// @todo use container/list instead
func (sl *segmentList) append(data *segment) {
	newNode := &segmentNode{data: data}

	if sl.head == nil {
		sl.head = newNode
		sl.tail = newNode
		return
	}

	newNode.prev = sl.tail
	sl.tail.next = newNode
	sl.tail = newNode
}

// @todo use container/list instead
func (sl *segmentList) insertBefore(nextNode *segmentNode, data *segment) {
	if nextNode == nil {
		fmt.Println("next node is nil")
		return
	}

	newNode := &segmentNode{
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

// @todo use container/list instead
func (sl *segmentList) delete(segmentNode *segmentNode) {
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

// @todo use container/list instead
func (sl *segmentList) topHeight() uint {
	var topHeight uint
	current := sl.head
	for current != nil {
		if current.data.y > topHeight {
			topHeight = current.data.y
		}
		current = current.next
	}
	return topHeight
}
