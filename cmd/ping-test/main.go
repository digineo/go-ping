package main

import (
	"net"
	"time"

	"fmt"

	"os"

	ping "github.com/digineo/go-ping"
)

func main() {

	var remote net.IPAddr

	if addr, err := net.ResolveIPAddr("ip4", os.Args[2]); err == nil {
		remote = *addr
	} else {
		panic(err)
	}

	p := &ping.Pinger{
		Local:    net.ParseIP(os.Args[1]),
		Remote:   remote,
		Attempts: 3,
		Timeout:  2 * time.Second,
	}

	fmt.Println(p.Ping())
}
