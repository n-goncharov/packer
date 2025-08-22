package main

import (
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		if err := run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run() error {
	w := new(app.Window)
	w.Option(app.Title("Packer"))
	w.Option(app.Size(unit.Dp(1920), unit.Dp(1080)))
	theme := material.NewTheme()

	var ops op.Ops

	for {
		switch e := w.Event().(type) {

		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			title := material.H1(theme, "Hello, Packer")

			title.Color = color.NRGBA{R: 127, G: 0, B: 0, A: 255}

			title.Alignment = text.Middle

			title.Layout(gtx)

			e.Frame(gtx.Ops)

		case app.DestroyEvent:
			return e.Err
		}
	}
}
