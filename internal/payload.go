package internal

import (
	"encoding/binary"
	"log/slog"
	"math/rand/v2"
	"time"
)

var (
	Logger = slog.Default()
	rnd    *rand.ChaCha8
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
