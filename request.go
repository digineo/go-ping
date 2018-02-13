package ping

// A request is a currently running ICMP echo request waiting for an answer.
type request struct {
	wait   chan struct{}
	result error
}

// respond is responsible for finishing this request. It takes an error
// as failure reason.
func (req *request) respond(err error) {
	req.result = err
	close(req.wait)
}
