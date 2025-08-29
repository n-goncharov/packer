package skyline

import (
	"fmt"
	"math"
)

// @todo add godoc

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

// @todo in a separate package create a separate generic structure for doubly linked list and its implementations
type segmentNode struct {
	data *segment
	next *segmentNode
	prev *segmentNode
}

type segmentList struct {
	head *segmentNode
	tail *segmentNode
}

// @todo call it PushBack
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

// @todo call it Remove
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
