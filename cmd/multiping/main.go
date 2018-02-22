package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	ping "github.com/digineo/go-ping"
)

var opts = struct {
	timeout        time.Duration
	interval       time.Duration
	statBufferSize uint
	bind4          string
	bind6          string
	dests          []*destination
}{
	timeout:        1000 * time.Millisecond,
	interval:       1000 * time.Millisecond,
	bind4:          "0.0.0.0",
	bind6:          "::",
	statBufferSize: 50,
}

var (
	pinger *ping.Pinger
	tui    *userInterface
)

func main() {
	flag.DurationVar(&opts.timeout, "timeout", opts.timeout, "timeout for a single echo request")
	flag.DurationVar(&opts.interval, "interval", opts.interval, "polling interval")
	flag.UintVar(&opts.statBufferSize, "buf", opts.statBufferSize, "buffer size for statistics")
	flag.StringVar(&opts.bind4, "bind4", opts.bind4, "IPv4 bind address")
	flag.StringVar(&opts.bind6, "bind6", opts.bind6, "IPv6 bind address")
	flag.Parse()

	log.SetFlags(0)

	for _, arg := range flag.Args() {
		if remote, err := net.ResolveIPAddr("ip4", arg); err == nil {
			dst := destination{
				host:   arg,
				remote: remote,
				stats: &stats{
					results: make([]time.Duration, opts.statBufferSize),
				},
			}

			if arg == remote.String() {
				dst.display = arg
			} else {
				dst.display = fmt.Sprintf("%s (%s)", arg, remote)
			}

			opts.dests = append(opts.dests, &dst)
		} else {
			log.Printf("host %s: %v", arg, err)
		}
	}

	if instance, err := ping.New(opts.bind4, opts.bind6); err == nil {
		instance.Timeout = opts.timeout
		instance.Attempts = 1
		pinger = instance
		defer pinger.Close()
	} else {
		panic(err)
	}

	go work()

	tui = buildTUI(opts.dests)
	go tui.update(time.Second)

	if err := tui.Run(); err != nil {
		panic(err)
	}
}

func work() {
	for {
		for i, u := range opts.dests {
			go func(u *destination, i int) {
				time.Sleep(time.Duration(i) * time.Millisecond)
				u.ping(pinger)
			}(u, i)
		}
		time.Sleep(opts.interval)
	}
}
