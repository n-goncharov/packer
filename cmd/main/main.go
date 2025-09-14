package main

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/core"
	"cogentcore.org/core/paint"
	_ "cogentcore.org/core/paint/renderers"
	"cogentcore.org/core/styles"
	"github.com/n-goncharov/packer/internal/container"
	"github.com/n-goncharov/packer/internal/item"
	"github.com/n-goncharov/packer/internal/packer"
	"github.com/n-goncharov/packer/internal/packer/skyline"
)

func main() {
	// Define rectangles.
	rectangles := []*item.Rectangle{
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

	// Define strips.
	strips := []*container.Strip{
		{Width: 13},
		{Width: 14},
		{Width: 15},
	}

	// Pack rectangles into strips using skyline algo.
	packedRectangles, err := packMSPP(skyline.NewPacker(), rectangles, strips)
	if err != nil {
		onError(err)
	}

	// Draw package.
	drawMSPP(packedRectangles, strips)
}

func onError(err error) {
	fmt.Println(err)
	panic(err)
}

func packMSPP(
	packer packer.Interface[item.Rectangle, container.Strip, item.PackedRectangle],
	rectangles []*item.Rectangle,
	strips []*container.Strip,
) ([]*item.PackedRectangle, error) {
	solution, err := packer.Pack(rectangles, strips)
	if err != nil {
		return nil, err
	}

	for _, i := range solution.PackedItems {
		fmt.Println(i)
	}

	fmt.Println(solution.FitnessValue)

	return solution.PackedItems, nil
}

type normalizedRect struct {
	x, y, w, h float32
}

func normalizeRectangles(packedRects []*item.PackedRectangle, maxRightBound, maxUpperBound float32) []*normalizedRect {
	normRects := make([]*normalizedRect, 0, len(packedRects))
	for _, r := range packedRects {
		normRects = append(normRects, &normalizedRect{
			x: float32(r.X) / maxRightBound,
			y: float32(r.Y) / maxUpperBound,
			w: float32(r.Item.Width) / maxRightBound,
			h: float32(r.Item.Height) / maxUpperBound,
		})
	}
	return normRects
}

func drawMSPP(packedRectangles []*item.PackedRectangle, strips []*container.Strip) {
	// Find right and upper bound of packing.
	var maxRightBound, maxUpperBound float32
	for _, c := range strips {
		maxRightBound += float32(c.Width)
	}

	for _, r := range packedRectangles {
		upperBound := float32(r.Y + r.Item.Height)
		if upperBound > maxUpperBound {
			maxUpperBound = upperBound
		}
	}
	maxUpperBound += maxRightBound / 20

	b := core.NewBody("Packer")

	b.Styler(func(s *styles.Style) {
		s.Background = image.NewUniform(color.White)
	})

	// Normalize rectangles to draw them in the canvas.
	normalizedRectangles := normalizeRectangles(packedRectangles, maxRightBound, maxUpperBound)

	c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
		// Draw rectangles.
		pc.Fill.Color = image.NewUniform(color.RGBA{R: 128, G: 128, B: 128, A: 255})
		pc.Stroke.Color = image.NewUniform(color.RGBA{R: 0, G: 0, B: 0, A: 255})

		for _, normRect := range normalizedRectangles {
			pc.Rectangle(normRect.x, 1-normRect.y-normRect.h, normRect.w, normRect.h)
			pc.Draw()
		}

		// Draw strips.

		// Draw lines to divide strips.
		var x uint
		var xNorm float32
		pc.MoveTo(xNorm, 1)
		pc.LineTo(xNorm, 0)
		for _, c := range strips {
			x += c.Width
			xNorm = float32(x) / maxRightBound
			pc.MoveTo(xNorm, 1)
			pc.LineTo(xNorm, 0)
		}

		// Draw bottom line.
		pc.MoveTo(0, 1)
		pc.LineTo(xNorm, 1)

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
