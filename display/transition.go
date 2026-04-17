package display

import (
	"image"
	"image/color"
)

// Mover animates a page image for one step of a transition.
// step ranges from 1 to total (inclusive); at step==total the image
// must occupy its fully settled, final position on-screen.
type Mover func(img *image.Gray, step, total int) *image.Gray

// overlayImages returns a new image with each pixel set to the brighter of a and b.
func overlayImages(a, b *image.Gray) *image.Gray {
	bounds := a.Bounds()
	out := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			va := a.GrayAt(x, y)
			vb := b.GrayAt(x, y)
			if va.Y > vb.Y {
				out.SetGray(x, y, color.Gray{Y: va.Y})
			} else {
				out.SetGray(x, y, color.Gray{Y: vb.Y})
			}
		}
	}
	return out
}
