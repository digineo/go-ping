package ping

import (
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Ping sends ICMP echo requests, retrying upto Pinger.Attempts times.
// Will finishes early on success.
func (pinger *Pinger) Ping(remote net.Addr) (err error) {
	_, err = pinger.PingRTT(remote)
	return
}

// PingRTT sends ICMP echo requests, retrying upto Pinger.Attempts times.
// Will finishes early on success and return the round trip time.
func (pinger *Pinger) PingRTT(remote net.Addr) (rtt time.Duration, err error) {
	// multiple attempts
	for i := uint(0); i < pinger.Attempts; i++ {
		if rtt, err = pinger.once(remote); err == nil {
			break // success
		}
	}
	return
}

// once sends a single Echo Request and waits for an answer. It returns
// the round trip time (RTT) if a reply is received in time.
func (pinger *Pinger) once(remote net.Addr) (time.Duration, error) {
	seq := uint16(atomic.AddUint32(&sequence, 1))
	req := request{
		wait: make(chan struct{}),
	}

	// build packet
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:  int(pinger.id),
			Seq: int(seq),
		},
	}
	// serialize packet
	wb, err := wm.Marshal(nil)
	if err != nil {
		return 0, err
	}

	// enqueue in currently running requests
	pinger.mtx.Lock()
	pinger.requests[seq] = &req
	pinger.mtx.Unlock()

	// start measurement (tStop is set in the receiving end)
	req.tStart = time.Now()

	// send request
	if _, e := pinger.conn.WriteTo(wb, remote); e != nil {
		req.respond(e, nil)
	}

	// wait for answer
	select {
	case <-req.wait:
		err = req.result
	case <-time.After(pinger.Timeout):
		err = &timeoutError{}
	}

	// dequeue request
	pinger.mtx.Lock()
	delete(pinger.requests, seq)
	pinger.mtx.Unlock()

	if err != nil {
		return 0, err
	}
	return req.roundTripTime()
}
