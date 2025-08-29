package main

import (
	"fmt"

	"github.com/n-goncharov/packer/internal/packing/container"
	"github.com/n-goncharov/packer/internal/packing/decoder"
	"github.com/n-goncharov/packer/internal/packing/decoder/skyline"
	"github.com/n-goncharov/packer/internal/packing/item"
	"golang.org/x/exp/constraints"
)

func main() {
	items := []*item.Rect{
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
		{Width: 10},
	}

	skylineDecoder, err := skyline.NewDecoder(containers)
	if err != nil {
		onError(err)
	}

	// Any other decoder with generic items can be passed here!
	process(skylineDecoder, items)
}

func process[I, PI any, V constraints.Integer | constraints.Float](decoder decoder.Interface[I, PI, V], items []*I) {
	solution, err := decoder.Decode(items)
	if err != nil {
		onError(err)
	}

	for _, i := range solution.PackedItems {
		fmt.Println(i)
	}

	fmt.Println(solution.ObjVal)
}

func onError(err error) {
	fmt.Println(err)
	panic(err)
}
