package main

import (
	"fmt"

	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/packer/skyline"
)

func main() {
	// Init rectangles.
	rectangles := []*item.Rect{
		{Width: 14, Height: 1},
		{Width: 13, Height: 2},
		{Width: 12, Height: 3},
		{Width: 11, Height: 4},
		{Width: 10, Height: 5},
		{Width: 9, Height: 6},
		{Width: 8, Height: 7},
		{Width: 7, Height: 8},
		{Width: 6, Height: 8},
		{Width: 5, Height: 10},
		{Width: 4, Height: 11},
		{Width: 3, Height: 12},
		{Width: 2, Height: 13},
		{Width: 1, Height: 14},
	}

	// Init strips.
	strips := []*container.Strip{
		{Width: 13},
		{Width: 14},
		{Width: 15},
	}

	// Pack rectangles into strips using skyline algo.
	solution, err := skyline.NewPacker().Pack(rectangles, strips)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	// Print results.
	for _, i := range solution.PackedItems {
		fmt.Println(i)
	}

	fmt.Println(solution.FitnessValue)
}
