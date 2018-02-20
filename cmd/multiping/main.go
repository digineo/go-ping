package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	ping "github.com/digineo/go-ping"
)

type stats struct {
	received int
	lost     int
	results  []time.Duration // ring buffer, index = .received
	mtx      sync.RWMutex
}

type unit struct {
	host    string
	remote  *net.IPAddr
	display string
	*stats
}

var opts = struct {
	timeout        time.Duration
	interval       time.Duration
	statBufferSize uint
	bind           string
	remotes        []unit

	pinger *ping.Pinger
}{
	timeout:        1000 * time.Millisecond,
	interval:       1000 * time.Millisecond,
	bind:           "0.0.0.0",
	statBufferSize: 50,
}

func main() {
	flag.DurationVar(&opts.timeout, "timeout", opts.timeout, "timeout for a single echo request")
	flag.DurationVar(&opts.interval, "interval", opts.interval, "polling interval")
	flag.UintVar(&opts.statBufferSize, "buf", opts.statBufferSize, "buffer size for statistics")
	flag.StringVar(&opts.bind, "bind", opts.bind, "bind address")
	flag.Parse()

	log.SetFlags(0)

	for _, arg := range flag.Args() {
		if remote, err := net.ResolveIPAddr("ip4", arg); err == nil {
			u := unit{
				host:   arg,
				remote: remote,
				stats: &stats{
					results: make([]time.Duration, opts.statBufferSize),
				},
			}

			if arg == remote.String() {
				u.display = arg
			} else {
				u.display = fmt.Sprintf("%s (%s)", arg, remote)
			}

			opts.remotes = append(opts.remotes, u)
		} else {
			log.Printf("host %s: %v", arg, err)
		}
	}

	if instance, err := ping.New(opts.bind); err == nil {
		instance.Timeout = opts.timeout
		instance.Attempts = 1
		opts.pinger = instance
	} else {
		panic(err)
	}

	go work()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("[interrupt received] %s", <-ch)

	opts.pinger.Close()
	log.Println("---- statistics: best/worst/mean/stddev ----")
	for _, u := range opts.remotes {
		sent, lost, _, best, worst, mean, std := u.compute()
		log.Printf("%-40s sent %d (%0.1f%% loss) %v/%v/%v/%v", u.display, sent, lost, best, worst, mean, std)
	}
}

func work() {
	for {
		for _, u := range opts.remotes {
			go func(u unit) { u.ping(opts.pinger) }(u)
		}
		time.Sleep(opts.interval)
	}
}

func (u *unit) ping(pinger *ping.Pinger) {
	rtt, err := pinger.PingRTT(u.remote)
	u.addResult(rtt, err)

	if err == nil {
		log.Printf("%-40s rtt=%v", u.display, rtt)
	} else {
		log.Printf("%-40s err=%v", u.display, err)
	}
}

func (s *stats) addResult(rtt time.Duration, err error) {
	s.mtx.Lock()
	if err == nil {
		s.results[s.received%len(s.results)] = rtt
		s.received++
	} else {
		s.lost++
	}
	s.mtx.Unlock()
}

func (s *stats) compute() (pktSent int, pktLoss float64, last, best, worst, mean, stddev time.Duration) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if s.received == 0 {
		if s.lost > 0 {
			pktLoss = 1.0
		}
		return
	}

	collection := s.results[:]
	pktSent = s.received + s.lost
	size := len(s.results)

	if s.received < size {
		collection = s.results[:s.received]
		size = s.received
	}

	last = collection[s.received%size]
	pktLoss = float64(s.lost) / float64(size)
	best, worst = collection[0], collection[0]

	total := time.Duration(0)
	for _, rtt := range collection {
		if rtt < best {
			best = rtt
		}
		if rtt > worst {
			worst = rtt
		}
		total += rtt
	}

	mean = time.Duration(float64(total) / float64(s.received))

	stddevNum := float64(0)
	for _, rtt := range collection {
		stddevNum += math.Pow(float64(rtt-mean), 2)
	}
	stddev = time.Duration(math.Sqrt(stddevNum / float64(mean)))

	return
}
