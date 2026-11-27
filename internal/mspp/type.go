package mspp

import "github.com/n-goncharov/packer/internal/item"

type Strip struct {
	Width uint
}

type Solution struct {
	PackedRects []item.PackedRect
	FitVal      uint
}
