package skyline

type segment struct {
	x, y, width int
	_           struct{}
}

type skyline struct {
	segments []segment
}

func newSkyline(x, width int) skyline {
	return skyline{
		segments: []segment{
			{
				x:     x,
				width: width,
			},
		},
	}
}
