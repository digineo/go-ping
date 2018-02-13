package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	ping "github.com/digineo/go-ping"
)

func main() {
	var attempts uint
	var timeout uint
	var bind string
	flag.UintVar(&attempts, "attempts", 3, "number of attempts")
	flag.UintVar(&timeout, "timeout", 1, "timeout in seconds for a single echo request")
	flag.StringVar(&bind, "bind", "0.0.0.0", "bind address")
	flag.Parse()

	args := flag.Args()

	remote, err := net.ResolveIPAddr("ip4", args[0])
	if err != nil {
		panic(err)
	}

	pinger, err := ping.New(bind)
	if err != nil {
		panic(err)
	}
	defer pinger.Close()

	pinger.Timeout = time.Second * time.Duration(timeout)
	pinger.Attempts = attempts

	if rtt, err := pinger.PingRTT(remote); err == nil {
		fmt.Printf("ping %s (%s) rtt=%v\n", args[0], remote, rtt)
	} else {
		fmt.Println(err)
		os.Exit(1)
	}
}
