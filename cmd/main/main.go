package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"cogentcore.org/core/core"
	"cogentcore.org/core/paint"
	_ "cogentcore.org/core/paint/renderers"
	"cogentcore.org/core/styles"
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
		{Width: 120},
		{Width: 120},
		{Width: 120},
	}

	skylineDecoder, err := skyline.NewDecoder(containers)
	if err != nil {
		onError(err)
	}

	// Pack items with the provided Decoder.
	// (any other decoder with generic items can be passed here!)
	packedRects := pack(skylineDecoder, items)

	// Draw package.
	drawSkylinePackage(packedRects, containers)
}

func onError(err error) {
	fmt.Println(err)
	panic(err)
}

func pack[I, PI any, V constraints.Integer | constraints.Float](decoder decoder.Interface[I, PI, V], items []*I) []PI {
	solution, err := decoder.Decode(items)
	if err != nil {
		onError(err)
	}

	for _, i := range solution.PackedItems {
		fmt.Println(i)
	}

	fmt.Println(solution.ObjVal)

	return solution.PackedItems
}

type normalizedRect struct {
	x, y, w, h float32
}

func normalizeRects(packedRects []*item.PackedRect, maxWidth, maxHeight float32) []*normalizedRect {
	normRects := make([]*normalizedRect, 0, len(packedRects))
	for _, r := range packedRects {
		normRects = append(normRects, &normalizedRect{
			x: float32(r.X) / maxWidth,
			y: float32(r.Y) / maxHeight,
			w: float32(r.Item.Width) / maxWidth,
			h: float32(r.Item.Height) / maxHeight,
		})
	}
	return normRects
}

func drawSkylinePackage(packedRects []*item.PackedRect, containers []*container.Strip) {
	var maxWidth, maxHeight float32
	for _, c := range containers {
		maxWidth += float32(c.Width)
	}

	for _, r := range packedRects {
		rectBound := float32(r.Y + r.Item.Height)
		if rectBound > maxHeight {
			maxHeight = rectBound
		}
	}

	b := core.NewBody("Packer")

	rectColors := make([]color.Color, 0, len(packedRects))
	for range packedRects {
		rectColors = append(rectColors, color.RGBA{
			R: uint8(rand.Intn(256)),
			G: uint8(rand.Intn(256)),
			B: uint8(rand.Intn(256)),
			A: 255,
		})
	}

	// Normalize rectangles to draw them in the canvas.
	normalizedRects := normalizeRects(packedRects, maxWidth, maxHeight)

	c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
		// Draw rectangles.
		for i, normRect := range normalizedRects {
			pc.Fill.Color = image.NewUniform(rectColors[i])
			pc.Rectangle(normRect.x, 1-normRect.y-normRect.h, normRect.w, normRect.h)
			pc.Draw()
		}
		pc.Stroke.Color = image.White
		pc.Draw()

		// Draw containers.

		// Draw bottom line.
		pc.MoveTo(0, 1)
		pc.LineTo(1, 1)

		// Draw lines to divide containers.
		var x uint
		var xNorm float32
		pc.MoveTo(xNorm, 1)
		pc.LineTo(xNorm, 0)
		for _, c := range containers {
			x += c.Width
			xNorm = float32(x) / maxWidth
			pc.MoveTo(xNorm, 1)
			pc.LineTo(xNorm, 0)
		}

		pc.Stroke.Color = image.NewUniform(color.RGBA{R: 255, G: 0, B: 0, A: 255})
		pc.Stroke.Width.Dp(2)
		pc.ToDots()
		pc.Draw()
	})

	c.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	b.RunMainWindow()
}
