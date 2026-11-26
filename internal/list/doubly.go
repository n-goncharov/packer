package list

import "fmt"

type DoublyList[T any] struct {
	Head *Node[T]
	Tail *Node[T]
}

type Node[T any] struct {
	Value T
	Prev  *Node[T]
	Next  *Node[T]
}

func (dl *DoublyList[T]) PushBack(val T) {
	newNode := &Node[T]{Value: val}

	if dl.Head == nil {
		dl.Head = newNode
		dl.Tail = newNode
		return
	}

	newNode.Prev = dl.Tail
	dl.Tail.Next = newNode
	dl.Tail = newNode
}

func (dl *DoublyList[T]) InsertBefore(nextNode *Node[T], val T) {
	if nextNode == nil {
		fmt.Println("next node is nil")
		return
	}

	newNode := &Node[T]{
		Value: val,
		Prev:  nextNode.Prev,
		Next:  nextNode,
	}

	nextNode.Prev = newNode

	if newNode.Prev != nil {
		newNode.Prev.Next = newNode
	} else {
		dl.Head = newNode
	}
}

func (dl *DoublyList[T]) Remove(n *Node[T]) {
	if n == nil {
		fmt.Println("segment node is nil")
		return
	}

	if n.Prev == nil {
		dl.Head = n.Next
	} else {
		n.Prev.Next = n.Next
	}

	if n.Next == nil {
		dl.Tail = n.Prev
	} else {
		n.Next.Prev = n.Prev
	}

	n.Prev = nil
	n.Next = nil
}
