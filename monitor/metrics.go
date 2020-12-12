package monitor

import "time"

// Metrics is a dumb data point computed from a history of Results.
type Metrics struct {
	PacketsSent int           // number of packets sent
	PacketsLost int           // number of packets lost
	Best        time.Duration // best rtt
	Worst       time.Duration // worst rtt
	Median      time.Duration // median rtt
	Mean        time.Duration // mean rtt
	StdDev      time.Duration // std deviation
}
