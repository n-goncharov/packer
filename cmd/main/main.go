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

	// Define strips.
	strips := []*container.Strip{
		{Width: 120},
		{Width: 100},
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

	b := core.NewBody("Packer")

	// Normalize rectangles to draw them in the canvas.
	normalizedRectangles := normalizeRectangles(packedRectangles, maxRightBound, maxUpperBound)

	c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
		// Draw rectangles.
		pc.Fill.Color = image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255})
		pc.Stroke.Color = image.NewUniform(color.RGBA{R: 0, G: 180, B: 230, A: 255})

		for _, normRect := range normalizedRectangles {
			pc.Rectangle(normRect.x, 1-normRect.y-normRect.h, normRect.w, normRect.h)
			pc.Draw()
		}

		// Draw strips.

		// Draw bottom line.
		pc.MoveTo(0, 1)
		pc.LineTo(1, 1)

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

		pc.Stroke.Color = image.NewUniform(color.RGBA{R: 200, G: 50, B: 200, A: 255})
		pc.Stroke.Width.Dp(2)
		pc.ToDots()
		pc.Draw()
	})

	c.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	b.RunMainWindow()
}
