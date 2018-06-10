package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	ping "github.com/digineo/go-ping"
)

var (
	attempts       uint = 3
	timeout             = time.Second
	proto4, proto6 bool
	size           uint = 56
	bind           string
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] host [host [...]]")
		flag.PrintDefaults()
	}

	flag.UintVar(&attempts, "attempts", attempts, "number of attempts")
	flag.DurationVar(&timeout, "timeout", timeout, "timeout for a single echo request")
	flag.UintVar(&size, "s", size, "size of additional payload data")
	flag.BoolVar(&proto4, "4", proto4, "use IPv4 (mutually exclusive with -6)")
	flag.BoolVar(&proto6, "6", proto6, "use IPv6 (mutually exclusive with -4)")
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

	if pinger.PayloadSize() != uint16(size) {
		pinger.SetPayloadSize(uint16(size))
	}

	if remote.IP.IsLinkLocalMulticast() {
		fmt.Printf("multicast ping to %s (%s)\n", args[0], remote)

		deadline := time.Now().Add(timeout)
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()

		responses, err := pinger.PingMulticast(ctx, remote)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for response := range responses {
			fmt.Printf("%+v\n", response)
		}

	} else {
		// non-multicast
		rtt, err := pinger.PingAttempts(remote, timeout, int(attempts))

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("ping %s (%s) rtt=%v\n", args[0], remote, rtt)
	}
}
