package internal

import (
	"math/rand"
	"time"

	"github.com/digineo/go-logwrap"
)

var (
	Logger = &logwrap.Instance{}

	// SetLogger allows updating the Logger. For details, see
	// "github.com/digineo/go-logwrap".Instance.SetLogger.
	SetLogger = Logger.SetLogger
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Payload represents additional data appended to outgoing ICMP Echo
// Requests.
type Payload []byte

// Resize will assign a new payload of the given size to p.
func (p *Payload) Resize(size uint16) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		Logger.Errorf("error resizing payload: %v", err)
		return
	}
	*p = Payload(buf)
}
