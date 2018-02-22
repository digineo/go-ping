package main

import (
	"math"
	"net"
	"sync"
	"time"

	ping "github.com/digineo/go-ping"
)

type history struct {
	received int
	lost     int
	results  []time.Duration // ring buffer, index = .received
	lastErr  error
	mtx      sync.RWMutex
}

type destination struct {
	host    string
	remote  *net.IPAddr
	display string
	*history
}

func (u *destination) ping(pinger *ping.Pinger) {
	u.addResult(pinger.PingRTT(u.remote))
}

func (s *history) addResult(rtt time.Duration, err error) {
	s.mtx.Lock()
	if err == nil {
		s.results[s.received%len(s.results)] = rtt
		s.received++
	} else {
		s.lastErr = err
		s.lost++
	}
	s.mtx.Unlock()
}

func (s *history) compute() (pktSent int, pktLoss float64, last, best, worst, mean, stddev time.Duration) {
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
