package skyline

import (
	"fmt"
	"math"
)

type segm struct {
	width            uint
	x, y             uint
	lHeight, rHeight float64 // todo do we need these values? consider checking next and prev segm on nil value
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

// @todo in a separate package create a separate generic structure for doubly linked list and its implementations
type segmNode struct {
	data *segm
	next *segmNode
	prev *segmNode
}

type segmList struct {
	head *segmNode
	tail *segmNode
}

func (sl *segmList) pushBack(data *segm) {
	newNode := &segmNode{data: data}

	if sl.head == nil {
		sl.head = newNode
		sl.tail = newNode
		return
	}

	newNode.prev = sl.tail
	sl.tail.next = newNode
	sl.tail = newNode
}

func (sl *segmList) insertBefore(nextNode *segmNode, data *segm) {
	if nextNode == nil {
		fmt.Println("next node is nil")
		return
	}

	newNode := &segmNode{
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

func (sl *segmList) remove(segmentNode *segmNode) {
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

func (sl *segmList) height() uint {
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
