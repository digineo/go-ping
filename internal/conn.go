package internal

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// ProtocolICMP is the number of the Internet Control Message Protocol
	// (see golang.org/x/net/internal/iana.ProtocolICMP)
	ProtocolICMP = 1

	// ProtocolICMPv6 is the IPv6 Next Header value for ICMPv6
	// see golang.org/x/net/internal/iana.ProtocolIPv6ICMP
	ProtocolICMPv6 = 58
)

var (
	errNotBound      = errors.New("need at least one bind address")
	errSocketMissing = errors.New("socket missing")
	id               = os.Getpid() & 0xffff
)

type Receiver func(body *icmp.Echo, icmpError error, addr net.IPAddr, tRecv *time.Time)

type Conn struct {
	Receiver   Receiver
	Privileged bool

	conn4 net.PacketConn
	conn6 net.PacketConn
}

// New creates a new Pinger. This will open the raw socket and start the
// receiving logic. You'll need to call Close() to cleanup.
func (c *Conn) Open(bind4, bind6 string) error {
	var err error
	var network4, network6 string

	if c.Privileged {
		network4 = "ip4:icmp"
		network6 = "ip6:ipv6-icmp"
	} else {
		network4 = "udp4"
		network6 = "udp6"
	}

	// open sockets
	c.conn4, err = connectICMP(network4, bind4)
	if err != nil {
		return err
	}

	c.conn6, err = connectICMP(network6, bind6)
	if err != nil {
		if c.conn4 != nil {
			c.conn4.Close()
		}
		return err
	}

	if c.conn4 == nil && c.conn6 == nil {
		return errNotBound
	}

	if c.conn4 != nil {
		go c.receiver(ProtocolICMP, c.conn4)
	}
	if c.conn6 != nil {
		go c.receiver(ProtocolICMPv6, c.conn6)
	}

	return nil
}

func (c *Conn) Close() {
	if c.conn4 != nil {
		c.conn4.Close()
	}
	if c.conn6 != nil {
		c.conn6.Close()
	}
}

// receiver listens on the raw socket and correlates ICMP Echo Replys with
// currently running requests.
func (c *Conn) receiver(proto int, conn net.PacketConn) {
	rb := make([]byte, 1500)

	// read incoming packets
	for {
		if n, source, err := conn.ReadFrom(rb); err != nil {
			if netErr, ok := err.(net.Error); !ok || !netErr.Temporary() {
				break // socket gone
			}
		} else {
			var ipAddr net.IPAddr

			switch addr := source.(type) {
			case *net.UDPAddr:
				ipAddr.IP = addr.IP
				ipAddr.Zone = addr.Zone
			case *net.IPAddr:
				ipAddr = *addr
			}

			c.receive(proto, rb[:n], ipAddr, time.Now())
		}
	}
}

// receive takes the raw message and tries to evaluate an ICMP response.
// If that succeeds, the body will given to process() for further processing.
func (c *Conn) receive(proto int, bytes []byte, addr net.IPAddr, t time.Time) {
	// parse message
	m, err := icmp.ParseMessage(proto, bytes)
	if err != nil {
		return
	}

	// evaluate message
	switch m.Type {
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:

		c.Receiver(m.Body.(*icmp.Echo), nil, addr, &t)

	case ipv4.ICMPTypeDestinationUnreachable, ipv6.ICMPTypeDestinationUnreachable:
		body := m.Body.(*icmp.DstUnreach)
		if body == nil {
			return
		}

		var bodyData []byte
		switch proto {
		case ProtocolICMP:
			// parse header of original IPv4 packet
			hdr, err := ipv4.ParseHeader(body.Data)
			if err != nil {
				return
			}
			bodyData = body.Data[hdr.Len:]
		case ProtocolICMPv6:
			// parse header of original IPv6 packet (we don't need the actual
			// header, but want to detect parsing errors)
			_, err := ipv6.ParseHeader(body.Data)
			if err != nil {
				return
			}
			bodyData = body.Data[ipv6.HeaderLen:]
		default:
			return
		}

		// parse ICMP message after the IP header
		msg, err := icmp.ParseMessage(proto, bodyData)
		if err != nil {
			return
		}

		err = fmt.Errorf("%v", m.Type)

		echo, ok := msg.Body.(*icmp.Echo)
		if !ok || echo == nil {
			Logger.Infof("expected *icmp.Echo, got %#v", msg)
			return
		}

		c.Receiver(echo, err, addr, nil)
	}
}

// sendRequest marshals the payload and sends the packet.
// It returns the sequence number and an error if the sending failed.
func (c *Conn) WriteTo(addr *net.IPAddr, seq int, data []byte) error {
	echo := icmp.Echo{
		Seq:  seq,
		Data: data,
	}
	msg := icmp.Message{
		Code: 0,
		Body: &echo,
	}

	var conn net.PacketConn
	if addr.IP.To4() != nil {
		msg.Type = ipv4.ICMPTypeEcho
		conn = c.conn4
	} else {
		msg.Type = ipv6.ICMPTypeEchoRequest
		conn = c.conn6
	}

	if c.Privileged {
		echo.ID = id
	}

	if conn == nil {
		return errSocketMissing
	}

	// serialize packet
	wb, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	// send request
	if c.Privileged {
		_, err = conn.WriteTo(wb, addr)
	} else {
		conn.WriteTo(wb, &net.UDPAddr{
			IP:   addr.IP,
			Zone: addr.Zone,
		})
	}

	return err
}

// connectICMP opens a new ICMP connection, if network and address are not empty.
func connectICMP(network, address string) (*icmp.PacketConn, error) {
	if network == "" || address == "" {
		return nil, nil
	}

	return icmp.ListenPacket(network, address)
}
