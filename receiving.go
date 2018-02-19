package ping

import (
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// receiver listens on the raw socket and correlates ICMP Echo Replys with
// currently running requests.
func (pinger *Pinger) receiver() {
	rb := make([]byte, 1500)

	// read incoming packets
	for {
		if n, _, err := pinger.conn.ReadFrom(rb); err != nil {
			if netErr, ok := err.(net.Error); !ok || !netErr.Temporary() {
				break // socket gone
			}
		} else {
			pinger.receive(rb[:n], time.Now())
		}
	}

	// close running requests
	pinger.mtx.RLock()
	for _, req := range pinger.requests {
		req.respond(errClosed, nil)
	}
	pinger.mtx.RUnlock()

	// Close() waits for us
	pinger.wg.Done()
}

// receive takes the raw message and tries to evaluate an ICMP response.
// If that succeedes, the body will given to process() for further processing.
func (pinger *Pinger) receive(bytes []byte, t time.Time) {
	// parse message
	rm, err := icmp.ParseMessage(ProtocolICMP, bytes)
	if err != nil {
		return
	}

	// evaluate message
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		pinger.process(rm.Body, nil, &t)

	case ipv4.ICMPTypeDestinationUnreachable:
		body := rm.Body.(*icmp.DstUnreach)
		if body == nil {
			return
		}

		// parse header of original IP packet
		hdr, err := ipv4.ParseHeader(body.Data)
		if err != nil {
			return
		}

		// parse ICMP message after the IP header
		msg, err := icmp.ParseMessage(ProtocolICMP, body.Data[hdr.Len:])
		if err != nil {
			return
		}

		pinger.process(msg.Body, fmt.Errorf("%s", rm.Type), nil)

	default:
		// other ICMP packet
		log.Printf("got: %+v %d", rm, rm.Body.Len(1))
	}
}

// process will finish a currently running Echo Request, iff the body is
// an ICMP Echo reply to a request from us.
func (pinger *Pinger) process(body icmp.MessageBody, result error, tRecv *time.Time) {
	echo := body.(*icmp.Echo)
	if echo == nil {
		return
	}

	// check if we sent this
	if uint16(echo.ID) != pinger.id {
		return
	}

	// search for existing running echo request
	pinger.mtx.RLock()
	req := pinger.requests[uint16(echo.Seq)]
	pinger.mtx.RUnlock()

	if req != nil {
		req.respond(result, tRecv)
	}
}
