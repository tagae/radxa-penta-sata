package transitions

import (
	"image"

	"radxa-penta-sata/display"
)

// SlideFromTop slides the image down into view from above the display.
// Use as the incoming mover to pair with SlideDown.
var SlideFromTop display.Mover = func(img *image.Gray, step, total int) *image.Gray {
	h := img.Bounds().Dy()
	return shiftY(img, -h+step*h/total)
}
