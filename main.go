package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"radxa-penta/sys"

	"github.com/spf13/cobra"
)

func main() {
	var configPath string
	var fontDir string
	var socketPath string

	root := &cobra.Command{
		Use:           "radxa-penta",
		Short:         "Fan control and OLED display for the Radxa Penta SATA HAT",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVarP(&configPath, "config", "c", "radxa-penta.conf", "path to config file")
	root.PersistentFlags().StringVar(&fontDir, "fonts", "/usr/share/consolefonts", "directory containing PSF fonts")
	root.PersistentFlags().StringVar(&socketPath, "socket", "radxa-penta.sock", "path to control socket")

	root.AddCommand(
		serveCmd(&configPath, &fontDir, &socketPath),
		statusCmd(&configPath),
		fanCmd(&configPath),
		displayCmd(&socketPath),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveCmd(configPath, fontDir, socketPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the fan and display service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig(*configPath)
			cfg.UpdateDisks()
			return serve(cfg, *fontDir, *socketPath)
		},
	}
}

func displayCmd(socketPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "Control the display",
	}
	send := func(action string) error {
		conn, err := net.DialTimeout("unix", *socketPath, 2*time.Second)
		if err != nil {
			return fmt.Errorf("connect to service: %w", err)
		}
		defer conn.Close()
		if _, err := fmt.Fprintln(conn, action); err != nil {
			return err
		}
		resp, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return err
		}
		if strings.TrimSpace(resp) != "ok" {
			return fmt.Errorf("%s", strings.TrimSpace(resp))
		}
		return nil
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "enable",
			Short: "Enable the display",
			RunE:  func(*cobra.Command, []string) error { return send("enable") },
		},
		&cobra.Command{
			Use:   "disable",
			Short: "Disable the display",
			RunE:  func(*cobra.Command, []string) error { return send("disable") },
		},
	)
	return cmd
}

func fanCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "fan <speed>",
		Short: "Set fan speed (0–100)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			speed, err := strconv.Atoi(args[0])
			if err != nil || speed < 0 || speed > 100 {
				return fmt.Errorf("speed must be an integer between 0 and 100")
			}
			cfg := LoadConfig(*configPath)
			fan := NewFan(cfg)
			if err := fan.initPin(); err != nil {
				return err
			}
			dc := 1.0 - float64(speed)/100.0
			fan.pin.Write(dc)
			fmt.Printf("Fan speed set to %d%%\n", speed)
			// Software PWM requires the process to stay alive to keep
			// generating pulses; block until the user interrupts.
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			<-sig
			return nil
		},
	}
}

func statusCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print current board status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig(*configPath)
			cfg.UpdateDisks()

			fmt.Println(sys.GetUptime())
			fmt.Println(sys.GetIP())
			fmt.Println(sys.GetMemory())
			fmt.Println(sys.GetCPUTemp(cfg.Display.FTemp))

			disks := cfg.GetDisks()
			temps := sys.DriveTemps(disks)
			for _, disk := range disks {
				mp := sys.MountpointOf("/dev/" + disk)
				usage := sys.DiskUsagePct(mp)
				if t, ok := temps[disk]; ok {
					fmt.Printf("Drive %s: %.0f°C  Disk: %s\n", disk, t, usage)
				} else {
					fmt.Printf("Drive %s: N/A  Disk: %s\n", disk, usage)
				}
			}

			dc := cfg.FanTemp2DC(sys.DriveTemp(disks))
			fmt.Printf("Fan: %d%%\n", int((1-dc)*100))
			return nil
		},
	}
}
