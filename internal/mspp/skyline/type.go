package skyline

type segment struct {
	x, y, width uint
	_           struct{}
}

type skyline struct {
	segments []segment
}

func newSkyline(x, width uint) skyline {
	return skyline{
		segments: []segment{
			{
				x:     x,
				width: width,
			},
		},
	}
}
