package monitor

import (
	"net"
	"sync"
	"time"

	"github.com/digineo/go-ping/internal"
)

// target represents a ping target.
type target struct {
	addr    net.IPAddr
	history History
	payload internal.Payload
	seq     uint16 // sequence number of the in-flight packet

	sent time.Time // timestamp of last sending
	sync.Mutex
}
