package ping

import "errors"

var (
	errClosed = errors.New("pinger closed")
)

// Taken from:
// https://github.com/golang/go/blob/master/src/net/net.go#L417-L421.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
