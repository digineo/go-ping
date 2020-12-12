package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digineo/go-ping/monitor"
)

var (
	pingInterval        = 5 * time.Second
	reportInterval      = 60 * time.Second
	size           uint = 56
	targets        []string
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] host [host [...]]")
		flag.PrintDefaults()
	}

	flag.DurationVar(&pingInterval, "pingInterval", pingInterval, "interval for ICMP echo requests")
	flag.DurationVar(&reportInterval, "reportInterval", reportInterval, "interval for reports")
	flag.UintVar(&size, "size", size, "size of additional payload data")
	flag.Parse()

	if n := flag.NArg(); n == 0 {
		// Targets empty?
		flag.Usage()
		os.Exit(1)
	} else if n > int(^byte(0)) {
		// Too many targets?
		fmt.Println("Too many targets")
		os.Exit(1)
	}

	// Create monitor
	monitor := monitor.New(pingInterval)
	monitor.PayloadSize = uint16(size)

	// Add targets
	targets = flag.Args()
	for i, target := range targets {
		ipAddr, err := net.ResolveIPAddr("", target)
		if err != nil {
			fmt.Printf("invalid target '%s': %s", target, err)
			continue
		}
		monitor.AddTarget(string([]byte{byte(i)}), *ipAddr)
	}

	// Start
	err := monitor.Start("0.0.0.0", "::")
	if err != nil {
		fmt.Printf("Unable to bind: %s\nRunning as root?\n", err)
		os.Exit(2)
	}

	defer monitor.Stop()

	// Start report routine
	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			for i, metrics := range monitor.Export() {
				fmt.Printf("%s: %+v\n", targets[[]byte(i)[0]], *metrics)
			}
		}
	}()

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("received", <-ch)
}
