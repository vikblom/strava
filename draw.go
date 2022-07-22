package strava

import (
	"image/color"
	"math"
	"time"

	"github.com/nikolaydubina/calendarheatmap/charts"
)

var (
	GRAY          = color.RGBA{240, 240, 240, 255}
	STRAVA_ORANGE = color.RGBA{252, 76, 2, 255}
)

var DefaultConfig = charts.HeatmapConfig{
	Format:             "png",
	DrawMonthSeparator: false,
	DrawLabels:         false,
	BoxSize:            30,
	Margin:             10,
	TextColor:          color.RGBA{100, 100, 100, 255},
	BorderColor:        color.RGBA{200, 200, 200, 255},
	Locale:             "sv_SE",
	ShowWeekdays:       map[time.Weekday]bool{},

	// Bias the colorscale so that a high max (very long activity) does not shift
	// everything else to gray.
	ColorScale: BiasedColorscale(GRAY, STRAVA_ORANGE, 100),
}

func LinearColorscale(from, to color.RGBA, n int) charts.BasicColorScale {
	// TODO
	if n < 2 {
		return nil
	}
	dr := float64(int(to.R)-int(from.R)) / float64(n)
	dg := float64(int(to.G)-int(from.G)) / float64(n)
	db := float64(int(to.B)-int(from.B)) / float64(n)
	cs := make(charts.BasicColorScale, n)
	for i := 0; i < n; i++ {
		cs[i] = color.RGBA{
			R: from.R + uint8(math.Round(dr*float64(i))),
			G: from.G + uint8(math.Round(dg*float64(i))),
			B: from.B + uint8(math.Round(db*float64(i))),
			A: 255,
		}
	}
	return cs
}

// BiasedColorscale leans towards the "to" color.
func BiasedColorscale(from, to color.RGBA, n int) charts.BasicColorScale {
	// TODO
	if n < 2 {
		return nil
	}
	dr := float64(int(to.R) - int(from.R))
	dg := float64(int(to.G) - int(from.G))
	db := float64(int(to.B) - int(from.B))
	cs := make(charts.BasicColorScale, n)
	for i := 0; i < n; i++ {
		cs[i] = color.RGBA{
			R: from.R + round(dr*square(float64(i)/float64(n))),
			G: from.G + round(dg*square(float64(i)/float64(n))),
			B: from.B + round(db*square(float64(i)/float64(n))),
			A: 255,
		}
	}
	return cs
}

// round to nearest uint8, no bounds checking.
func round(x float64) uint8 {
	return uint8(math.Round(x))
}

// square f in magnitude, preserving sign.
func square(f float64) float64 {
	return math.Copysign(math.Sqrt(math.Abs(f)), f)
}
