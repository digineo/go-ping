package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	ping "github.com/digineo/go-ping"
)

type unit struct {
	host   string
	remote *net.IPAddr
}

var opts = struct {
	timeout  time.Duration
	interval time.Duration
	bind     string
	remotes  []unit
}{
	timeout:  1000 * time.Millisecond,
	interval: 1000 * time.Millisecond,
	bind:     "0.0.0.0",
}

func main() {
	flag.DurationVar(&opts.timeout, "timeout", opts.timeout, "timeout for a single echo request")
	flag.DurationVar(&opts.interval, "interval", opts.interval, "polling interval")
	flag.StringVar(&opts.bind, "bind", opts.bind, "bind address")
	flag.Parse()

	log.SetFlags(0)

	for _, arg := range flag.Args() {
		if remote, err := net.ResolveIPAddr("ip4", arg); err == nil {
			u := unit{host: arg, remote: remote}
			opts.remotes = append(opts.remotes, u)
		} else {
			log.Printf("host %s: %v", arg, err)
		}
	}

	var pinger *ping.Pinger
	if instance, err := ping.New(opts.bind); err == nil {
		pinger = instance
	} else {
		panic(err)
	}

	pinger.Timeout = opts.timeout
	pinger.Attempts = 1

	for _, u := range opts.remotes {
		go func(u unit) {
			u.ping(opts.interval, pinger)
		}(u)
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("[interrupt received] %s", <-ch)

	pinger.Close()
}

func (u *unit) ping(interval time.Duration, pinger *ping.Pinger) {
	for {
		if rtt, err := pinger.PingRTT(u.remote); err == nil {
			log.Printf("ping %s (%s) rtt=%v", u.host, u.remote, rtt)
		} else {
			log.Printf("ping %s (%s) err=%v", u.host, u.remote, err)
		}
		time.Sleep(interval)
	}
}
