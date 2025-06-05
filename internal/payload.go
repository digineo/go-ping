package internal

import (
	"encoding/binary"
	"math/rand/v2"
	"time"

	"github.com/digineo/go-logwrap"
)

var (
	Logger = &logwrap.Instance{}

	// SetLogger allows updating the Logger. For details, see
	// "github.com/digineo/go-logwrap".Instance.SetLogger.
	SetLogger = Logger.SetLogger

	rnd *rand.ChaCha8
)

func init() {
	seed := [32]byte{}
	binary.NativeEndian.PutUint64(seed[:], uint64(time.Now().UnixNano()))
	rnd = rand.NewChaCha8(seed)
}

// Payload represents additional data appended to outgoing ICMP Echo
// Requests.
type Payload []byte

// Resize will assign a new payload of the given size to p.
func (p *Payload) Resize(size uint16) {
	buf := make([]byte, size)
	_, _ = rnd.Read(buf)
	*p = Payload(buf)
}
