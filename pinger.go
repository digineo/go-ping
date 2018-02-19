package ping

import (
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

const (
	// ProtocolICMP is the number of the Internet Control Message Protocol
	ProtocolICMP = 1

	// ProtocolIPv6ICMP is ICMP for IPv6
	ProtocolIPv6ICMP = 58
)

// sequence number for this process
var sequence uint32

// Pinger is a instance for ICMP echo requests
type Pinger struct {
	Local    net.IP        // IP address to bind on
	Attempts uint          // number of attempts
	Timeout  time.Duration // timeout per request

	requests map[uint16]*request // currently running requests
	mtx      sync.RWMutex        // lock for the requests map
	id       uint16
	conn     *icmp.PacketConn
	wg       sync.WaitGroup
}

// New creates a new Pinger. This will open the raw socket and start the
// receiving logic. You'll need to call Close() to cleanup.
func New(bind string) (*Pinger, error) {
	// Socket öffnen
	conn, err := icmp.ListenPacket("ip4:icmp", bind)
	if err != nil {
		return nil, err
	}

	pinger := Pinger{
		conn:     conn,
		id:       uint16(os.Getpid()),
		requests: make(map[uint16]*request),
	}

	pinger.wg.Add(1)
	go pinger.receiver()

	return &pinger, nil
}

// Close will close the ICMP socket.
func (pinger *Pinger) Close() {
	pinger.conn.Close()
	pinger.wg.Wait()
}
