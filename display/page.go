package display

import (
	"fmt"

	"radxa-penta-sata/sys"
)

// PageConfig holds the content and timing configuration for page generation.
type PageConfig struct {
	FTemp      bool
	SliderTime float64
	GetDisks   func() []string
}

// Build returns the current set of display pages.
func Build(cfg PageConfig) []Page {
	return []Page{
		{
			{X: 0, Y: 0, Text: sys.GetUptime()},
			{X: 0, Y: 11, Text: sys.GetCPUTemp(cfg.FTemp)},
			{X: 0, Y: 21, Text: sys.GetIP()},
		},
		{
			{X: 0, Y: 2, Text: sys.GetCPULoad()},
			{X: 0, Y: 18, Text: sys.GetMemory()},
		},
		diskUsage(cfg),
		diskTemps(cfg),
	}
}

func diskUsage(cfg PageConfig) Page {
	keys, vals := sys.GetDiskUsage(cfg.GetDisks())
	if len(keys) == 0 {
		return Page{{X: 0, Y: 10, Text: "Disk: N/A"}}
	}

	text1 := fmt.Sprintf("Disk: %s %s", keys[0], vals[0])

	switch len(keys) {
	case 5:
		text2 := fmt.Sprintf("%s %s %s %s", keys[1], vals[1], keys[2], vals[2])
		text3 := fmt.Sprintf("%s %s %s %s", keys[3], vals[3], keys[4], vals[4])
		return Page{
			{X: 0, Y: 0, Text: text1},
			{X: 0, Y: 11, Text: text2},
			{X: 0, Y: 21, Text: text3},
		}
	case 3:
		text2 := fmt.Sprintf("%s %s %s %s", keys[1], vals[1], keys[2], vals[2])
		return Page{
			{X: 0, Y: 2, Text: text1},
			{X: 0, Y: 18, Text: text2},
		}
	case 2:
		return Page{
			{X: 0, Y: 2, Text: text1},
			{X: 0, Y: 18, Text: fmt.Sprintf("%s %s", keys[1], vals[1])},
		}
	default:
		return Page{{X: 0, Y: 10, Text: text1}}
	}
}

func diskTemps(cfg PageConfig) Page {
	keys, vals := sys.GetDiskTemps(cfg.GetDisks())
	if len(keys) == 0 {
		return Page{{X: 0, Y: 10, Text: "Temp: N/A"}}
	}

	n := len(keys)
	if n > 6 {
		n = 6
	}
	keys, vals = keys[:n], vals[:n]

	pair := func(i int) string {
		if i+1 < n {
			return fmt.Sprintf("%s %s %s %s", keys[i], vals[i], keys[i+1], vals[i+1])
		}
		return fmt.Sprintf("%s %s", keys[i], vals[i])
	}

	switch {
	case n == 1:
		return Page{{X: 0, Y: 10, Text: pair(0)}}
	case n == 2:
		return Page{
			{X: 0, Y: 2, Text: pair(0)},
			{X: 0, Y: 18, Text: pair(1)},
		}
	case n <= 4:
		return Page{
			{X: 0, Y: 2, Text: pair(0)},
			{X: 0, Y: 18, Text: pair(2)},
		}
	default:
		return Page{
			{X: 0, Y: 0, Text: pair(0)},
			{X: 0, Y: 11, Text: pair(2)},
			{X: 0, Y: 21, Text: pair(4)},
		}
	}
}
