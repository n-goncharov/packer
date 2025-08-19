package main

import (
	"fmt"

	"github.com/n-goncharov/packing-backend/internal/packing/container"
	"github.com/n-goncharov/packing-backend/internal/packing/decoder/skyline"
	"github.com/n-goncharov/packing-backend/internal/packing/shape"
)

func main() {
	items := []*shape.Rect{
		{Width: 10, Height: 140},
		{Width: 60, Height: 170},
		{Width: 40, Height: 50},
		{Width: 120, Height: 90},
		{Width: 100, Height: 320},
		{Width: 20, Height: 480},
		{Width: 110, Height: 70},
		{Width: 90, Height: 110},
		{Width: 100, Height: 110},
		{Width: 90, Height: 440},
		{Width: 50, Height: 120},
		{Width: 60, Height: 800},
		{Width: 70, Height: 120},
		{Width: 80, Height: 10},
	}

	containers := []*container.Strip{
		{Width: 150},
		{Width: 150},
	}

	decoder, err := skyline.NewDecoder(items, containers)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	solution, err := decoder.Decode()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	for _, rectangle := range solution.PackedItems {
		fmt.Println(rectangle)
	}

	fmt.Println(solution.ObjFnVal)
}
