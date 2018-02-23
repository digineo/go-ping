package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	ping "github.com/digineo/go-ping"
)

func main() {
	var attempts, timeout uint
	var proto4, proto6 bool
	var size uint
	var bind string

	flag.UintVar(&attempts, "attempts", 3, "number of attempts")
	flag.UintVar(&timeout, "timeout", 1, "timeout in seconds for a single echo request")
	flag.UintVar(&size, "s", 56, "size of additional payload data")
	flag.BoolVar(&proto4, "4", false, "use IPv4 (mutually exclusive with -6)")
	flag.BoolVar(&proto6, "6", false, "use IPv6 (mutually exclusive with -4)")
	flag.StringVar(&bind, "bind", "", "IPv4 or IPv6 bind address (defaults to 0.0.0.0 for IPv4 and :: for IPv6)")
	flag.Parse()

	if proto4 == proto6 {
		log.Fatalf("need exactly one of -4 and -6 flags")
	}

	if bind == "" {
		if proto4 {
			bind = "0.0.0.0"
		} else if proto6 {
			bind = "::"
		}
	}

	args := flag.Args()

	var pinger *ping.Pinger
	var remote *net.IPAddr

	if proto4 {
		if r, err := net.ResolveIPAddr("ip4", args[0]); err != nil {
			panic(err)
		} else {
			remote = r
		}

		if p, err := ping.New(bind, ""); err != nil {
			panic(err)
		} else {
			pinger = p
		}
	} else if proto6 {
		if r, err := net.ResolveIPAddr("ip6", args[0]); err != nil {
			panic(err)
		} else {
			remote = r
		}

		if p, err := ping.New("", bind); err != nil {
			panic(err)
		} else {
			pinger = p
		}
	}
	defer pinger.Close()

	pinger.Timeout = time.Second * time.Duration(timeout)
	pinger.Attempts = attempts
	if pinger.PayloadSize() != uint16(size) {
		pinger.SetPayloadSize(uint16(size))
	}

	if rtt, err := pinger.PingRTT(remote); err == nil {
		fmt.Printf("ping %s (%s) rtt=%v\n", args[0], remote, rtt)
	} else {
		fmt.Println(err)
		os.Exit(1)
	}
}
