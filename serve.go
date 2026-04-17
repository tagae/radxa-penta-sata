package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"radxa-penta/display"
	"radxa-penta/display/transitions"
)

func serve(cfg *Config, fontDir, socketPath string) error {
	fan := NewFan(cfg)

	// Try to initialize the display (top board). Skipped when disabled
	// in config, or when init fails (no I2C bus, no display attached);
	// in both cases the service runs in headless mode with fan control only.
	var oled *display.Display
	if cfg.Display.Enabled {
		var err error
		oled, err = display.New(display.Config{
			Rotate:     cfg.Display.Rotate,
			Font:       cfg.Display.Font,
			FontMedium: cfg.Display.FontMedium,
			FontLarge:  cfg.Display.FontLarge,
		}, fontDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "display init failed (headless mode): %v\n", err)
		}
	}
	topBoard := oled != nil

	if topBoard {
		eventCh := make(chan string, 10)
		var mu sync.Mutex
		slider := display.NewSlider(oled, display.PageConfig{
			FTemp:      cfg.Display.FTemp,
			SliderTime: cfg.Slider.Time,
			GetDisks:   cfg.GetDisks,
		})
		if out, ok := transitions.Movers[cfg.Slider.TransitionOut]; ok {
			if in, ok := transitions.Movers[cfg.Slider.TransitionIn]; ok {
				slider.WithTransition(out, in,
					cfg.Slider.TransitionSteps,
					time.Duration(cfg.Slider.TransitionFrameMs)*time.Millisecond)
			}
		}

		actions := map[string]func(){
			"none":     func() {},
			"slider":   func() { slider.Next(&mu) },
			"switch":   func() { cfg.FanSwitch() },
			"reboot":   func() { syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART) },
			"poweroff": func() { syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF) },
		}

		// Receive button events and dispatch actions.
		go func() {
			for key := range eventCh {
				fn := cfg.GetFunc(key)
				if action, ok := actions[fn]; ok {
					action()
				}
			}
		}()

		// Watch the hardware button.
		go WatchButton(cfg, eventCh)

		// Control socket for commands.
		go serveSocket(slider, socketPath)

		// Auto-rotate pages.
		go slider.Auto(&mu)

		// Fan thermal control.
		go fan.Run()

		// Wait for termination signal.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		oled.Goodbye()
		oled.Close()
	} else {
		// Headless: only run fan control (blocks).
		fan.Run()
	}
	return nil
}

func serveSocket(slider *display.Slider, socketPath string) {
	os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "control socket: %v\n", err)
		return
	}
	defer os.Remove(socketPath)
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleControl(conn, slider)
	}
}

func handleControl(conn net.Conn, slider *display.Slider) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	commands := map[string]func(){
		"enable":  slider.Enable,
		"disable": slider.Disable,
	}
	if fn, ok := commands[scanner.Text()]; ok {
		fn()
		fmt.Fprintln(conn, "ok")
	} else {
		fmt.Fprintln(conn, "error: unknown command")
	}
}
