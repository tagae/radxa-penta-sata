package transitions

import (
	"image"

	"radxa-penta-sata/display"
)

// Movers is the registry of named movers available for use in configuration.
var Movers = map[string]display.Mover{
	"slide-down":     SlideDown,
	"slide-from-top": SlideFromTop,
}

// shiftY returns a copy of img shifted vertically by dy pixels.
// Positive dy moves content downward; negative moves it upward.
// Rows that scroll off either edge are replaced with black.
func shiftY(img *image.Gray, dy int) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		srcY := y - dy
		if srcY >= b.Min.Y && srcY < b.Max.Y {
			for x := b.Min.X; x < b.Max.X; x++ {
				out.SetGray(x, y, img.GrayAt(x, srcY))
			}
		}
	}
	return out
}
