package ping

type ICMPError int

func (e *ICMPError) Error() string {
	// FIXME
	return "icmperror"
}

// timeoutError implements the net.Error interface. Originally taken from
// https://github.com/golang/go/blob/release-branch.go1.8/src/net/net.go#L505-L509
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
