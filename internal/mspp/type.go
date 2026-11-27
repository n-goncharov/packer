package mspp

import "github.com/n-goncharov/packer/internal/item"

type Strip struct {
	Width int
}

type Solution struct {
	PackedRects []item.PackedRect
	FitVal      int
}
