package transitions

import (
	"image"

	"radxa-penta/display"
)

// SlideDown moves the image downward, exiting through the bottom of the display.
// Use as the outgoing mover to pair with SlideFromTop.
var SlideDown display.Mover = func(img *image.Gray, step, total int) *image.Gray {
	return shiftY(img, step*img.Bounds().Dy()/total)
}
