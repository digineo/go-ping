package ping

import (
	"crypto/rand"

	"github.com/digineo/go-logwrap"
)

var (
	log = &logwrap.Instance{}

	// SetLogger allows updating the Logger. For details, see
	// "github.com/digineo/go-logwrap".Instance.SetLogger.
	SetLogger = log.SetLogger
)

// Payload represents additional data appended to outgoing ICMP Echo
// Requests.
type Payload []byte

// Resize will assign a new payload of the given size to p.
func (p *Payload) Resize(size uint16) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		log.Errorf("error resizing payload: %v", err)
		return
	}
	*p = Payload(buf)
}
