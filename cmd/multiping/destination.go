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

type stat struct {
	pktSent   int
	pktLoss   float64
	last      time.Duration
	best      time.Duration
	worst     time.Duration
	mean      time.Duration
	stddev    time.Duration
	lastError string
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

func (s *history) compute() (st stat) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if s.received == 0 {
		if s.lost > 0 {
			st.pktLoss = 1.0
		}
		return
	}

	collection := s.results[:]
	st.pktSent = s.received + s.lost
	size := len(s.results)

	if s.received < size {
		collection = s.results[:s.received]
		size = s.received
	}

	st.last = collection[s.received%size]
	st.pktLoss = float64(s.lost) / float64(size)
	st.best, st.worst = collection[0], collection[0]

	total := time.Duration(0)
	for _, rtt := range collection {
		if rtt < st.best {
			st.best = rtt
		}
		if rtt > st.worst {
			st.worst = rtt
		}
		total += rtt
	}

	st.mean = time.Duration(float64(total) / float64(s.received))

	stddevNum := float64(0)
	for _, rtt := range collection {
		stddevNum += math.Pow(float64(rtt-st.mean), 2)
	}
	st.stddev = time.Duration(math.Sqrt(stddevNum / float64(st.mean)))

	return
}
