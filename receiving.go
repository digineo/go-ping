package ping

import (
	"net"
	"time"

	"golang.org/x/net/icmp"
)

// process will finish a currently running Echo Request, if the body is
// an ICMP Echo reply to a request from us.
func (pinger *Pinger) process(body *icmp.Echo, icmpError error, addr net.IPAddr, tRecv *time.Time) {
	// search for existing running echo request
	pinger.mtx.Lock()
	req := pinger.requests[uint16(body.Seq)]
	if _, ok := req.(*simpleRequest); ok {
		// a simpleRequest is finished on the first reply
		delete(pinger.requests, uint16(body.Seq))
	}
	pinger.mtx.Unlock()

	if req != nil {
		req.handleReply(icmpError, addr, tRecv)
	}
}
