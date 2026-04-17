package display

import (
	"image"
	"sync"
	"sync/atomic"
	"time"
)

// Slider cycles through display pages with optional animated transitions.
type Slider struct {
	d          *Display
	cfg        PageConfig
	idx        int64
	lastImage  *image.Gray
	outgoing   Mover
	incoming   Mover
	steps      int
	frameDelay time.Duration
	resetTimer chan struct{} // signals Auto to reset its interval timer
	control    chan bool     // true = enable, false = disable
}

// NewSlider creates a Slider for the given display and page configuration.
func NewSlider(d *Display, cfg PageConfig) *Slider {
	return &Slider{
		d:          d,
		cfg:        cfg,
		idx:        -1,
		resetTimer: make(chan struct{}, 1),
		control:    make(chan bool, 1),
	}
}

// WithTransition configures animated transitions between pages. out and in are
// applied to the outgoing and incoming images respectively, over steps frames
// with frameDelay between each. Returns the receiver for chaining.
func (s *Slider) WithTransition(out, in Mover, steps int, frameDelay time.Duration) *Slider {
	s.outgoing = out
	s.incoming = in
	s.steps = steps
	s.frameDelay = frameDelay
	return s
}

// Enable resumes page cycling after a call to Disable.
func (s *Slider) Enable() { s.setControl(true) }

// Disable stops page cycling and clears the display.
func (s *Slider) Disable() { s.setControl(false) }

// setControl sends v to the control channel, replacing any pending value so
// the most recent call always wins.
func (s *Slider) setControl(v bool) {
	select {
	case s.control <- v:
	default:
		<-s.control
		s.control <- v
	}
}

// advance renders and shows the next page. The caller must hold the display mutex.
func (s *Slider) advance() {
	p := Build(s.cfg)
	idx := int(atomic.AddInt64(&s.idx, 1)) % len(p)
	img := s.d.RenderImage(p[idx])
	if s.outgoing != nil && s.lastImage != nil {
		s.d.Animate(s.lastImage, img, s.outgoing, s.incoming, s.steps, s.frameDelay)
	} else {
		s.d.Show(img)
	}
	s.lastImage = img
}

// Next renders the next page and resets the auto-advance timer (thread-safe).
func (s *Slider) Next(mu *sync.Mutex) {
	mu.Lock()
	s.advance()
	// Signal Auto to reset its timer. If a signal is already pending the
	// timer will be reset regardless, so a simple non-blocking send suffices.
	select {
	case s.resetTimer <- struct{}{}:
	default:
	}
	mu.Unlock()
}

// Auto shows the first page immediately, then advances at the configured interval.
// It pauses when Disable is called and resumes when Enable is called.
func (s *Slider) Auto(mu *sync.Mutex) {
	mu.Lock()
	s.advance()
	mu.Unlock()

	d := time.Duration(s.cfg.SliderTime * float64(time.Second))
	timer := time.NewTimer(d)

	for {
		select {
		case <-timer.C:
			mu.Lock()
			// If Next ran while we waited for the lock, skip this advance so
			// the page does not change twice in rapid succession.
			select {
			case <-s.resetTimer:
			default:
				s.advance()
			}
			mu.Unlock()

		case <-s.resetTimer:
			// Next already advanced; just reset the interval.
			stopTimer(timer)

		case v := <-s.control:
			if !v {
				// Disable: stop the timer and clear the display.
				stopTimer(timer)
				mu.Lock()
				s.d.Clear()
				s.lastImage = nil
				mu.Unlock()
				// Block until re-enabled.
				for !<-s.control {
				}
				// Re-enabled: show first page and start fresh.
				mu.Lock()
				s.advance()
				mu.Unlock()
			}
		}

		timer.Reset(d)
	}
}

// stopTimer stops t and drains its channel if it had already fired.
func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
