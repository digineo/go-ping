package ping

import (
	"crypto/rand"
	"log/slog"
	"sync/atomic"
)

var logger atomic.Pointer[slog.Logger]

// SetLogger sets the logger used by this package. If l is nil (or
// SetLogger is never called), slog.Default() is used.
func SetLogger(l *slog.Logger) {
	logger.Store(l)
}

func getLogger() *slog.Logger {
	if l := logger.Load(); l != nil {
		return l
	}
	return slog.Default()
}

// Payload represents additional data appended to outgoing ICMP Echo
// Requests.
type Payload []byte

// Resize will assign a new payload of the given size to p.
func (p *Payload) Resize(size uint16) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		getLogger().Error("error resizing payload", slog.Any("error", err))
		return
	}
	*p = Payload(buf)
}
