package display

import (
	"fmt"
	"image"
	"os"
	"time"
	"unicode/utf8"

	"radxa-penta/display/font"
	"radxa-penta/pin"

	"github.com/warthog618/go-gpiocdev"
)

// Item is a text element to render at a given position.
type Item struct {
	X, Y int
	Text string
}

// Page is a collection of items rendered together as one screen.
type Page []Item

// Display manages the SSD1306 display.
type Display struct {
	dev        *ssd1306
	cfg        Config
	smallFont  *font.Font // 3-row pages
	mediumFont *font.Font // 2-row pages
	largeFont  *font.Font // 1-row pages
	resetPin   *gpiocdev.Line
	width      int
	height     int
}

// New resets the display hardware, initializes the SSD1306 over
// I2C, and loads the PSF fonts from fontDir.
func New(cfg Config, fontDir string) (*Display, error) {
	// Reset the display via GPIO.
	resetChip, resetLine := pin.ParsePin(os.Getenv("OLED_RESET"))
	rst, err := gpiocdev.RequestLine(resetChip, resetLine, gpiocdev.AsOutput(1))
	if err != nil {
		return nil, fmt.Errorf("reset GPIO: %w", err)
	}
	rst.SetValue(1)
	time.Sleep(time.Millisecond)
	rst.SetValue(0)
	time.Sleep(10 * time.Millisecond)
	rst.SetValue(1)
	time.Sleep(10 * time.Millisecond)

	// Open I2C and init SSD1306.
	i2cDev := pin.I2CBus()
	dev, err := newSSD1306(i2cDev, 128, 32)
	if err != nil {
		rst.Close()
		return nil, fmt.Errorf("SSD1306: %w", err)
	}
	dev.clear() //nolint:errcheck

	// Load PSF fonts; medium and large fall back to small if unset or missing.
	small, err := font.Load(fontDir + "/" + cfg.Font)
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	loadOpt := func(name string) *font.Font {
		if name == "" {
			return small
		}
		if f, err := font.Load(fontDir + "/" + name); err == nil {
			return f
		}
		return small
	}
	medium := loadOpt(cfg.FontMedium)
	large := loadOpt(cfg.FontLarge)

	return &Display{
		dev:        dev,
		cfg:        cfg,
		smallFont:  small,
		mediumFont: medium,
		largeFont:  large,
		resetPin:   rst,
		width:      128,
		height:     32,
	}, nil
}

// newImage returns a blank 128x32 grayscale image.
func (d *Display) newImage() *image.Gray {
	return image.NewGray(image.Rect(0, 0, d.width, d.height))
}

// Show sends the image to the display, optionally rotating 180°.
func (d *Display) Show(img *image.Gray) {
	out := img
	if d.cfg.Rotate {
		out = rotate180(img)
	}
	d.dev.display(imageToSSD1306(out, d.width, d.height)) //nolint:errcheck
}

// RenderImage renders a page to a grayscale image without displaying it.
// The font is chosen by row count, then stepped down if any item overflows.
func (d *Display) RenderImage(page Page) *image.Gray {
	fnt := d.smallFont
	for _, f := range d.fontsForPage(len(page)) {
		fnt = f
		if d.pageFits(page, f) {
			break
		}
	}
	img := d.newImage()
	for _, item := range page {
		fnt.DrawText(img, item.X, item.Y, item.Text)
	}
	return img
}

// fontsForPage returns fonts in descending size order starting from the
// preferred size for the given number of rows.
func (d *Display) fontsForPage(rows int) []*font.Font {
	switch {
	case rows == 1:
		return []*font.Font{d.largeFont, d.mediumFont, d.smallFont}
	case rows == 2:
		return []*font.Font{d.mediumFont, d.smallFont}
	default:
		return []*font.Font{d.smallFont}
	}
}

// pageFits reports whether every item in the page fits within the display width
// when rendered with fnt.
func (d *Display) pageFits(page Page, fnt *font.Font) bool {
	for _, item := range page {
		if item.X+utf8.RuneCountInString(item.Text)*fnt.Width > d.width {
			return false
		}
	}
	return true
}

// Render draws a page on the display, choosing the font by row count.
func (d *Display) Render(page Page) {
	d.Show(d.RenderImage(page))
}

// Animate displays a step-by-step transition from old to new. At each of the
// steps frames, out and in are applied to the respective images and the result
// is composited and shown. frameDelay is inserted between frames.
func (d *Display) Animate(old, new *image.Gray, out, in Mover, steps int, frameDelay time.Duration) {
	for step := 1; step <= steps; step++ {
		d.Show(overlayImages(out(old, step, steps), in(new, step, steps)))
		if step < steps {
			time.Sleep(frameDelay)
		}
	}
}

// Clear blanks the display.
func (d *Display) Clear() {
	d.dev.clear()
}

// Goodbye shows the shutdown screen for 2 seconds, then clears.
func (d *Display) Goodbye() {
	img := d.newImage()
	y := (d.height - d.largeFont.Height) / 2
	d.largeFont.DrawText(img, 0, y, "Goodbye ~")
	d.Show(img)
	time.Sleep(2 * time.Second)
	d.dev.clear() //nolint:errcheck
}

// Close releases hardware resources.
func (d *Display) Close() {
	d.dev.close() //nolint:errcheck
	d.resetPin.Close()
}
