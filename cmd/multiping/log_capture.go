package main

import (
	"bytes"
	"log"
)

type logInterceptor struct {
	keep     int
	messages []string
}

func (li *logInterceptor) Write(p []byte) (n int, err error) {
	li.messages = append(li.messages, string(bytes.TrimSpace(p)))

	if li.keep > 0 {
		li.truncate()
	}

	return len(p), nil
}

func interceptLog(keep int) *logInterceptor {
	li := &logInterceptor{keep: keep}
	log.SetOutput(li)
	return li
}

func (li *logInterceptor) truncate() {
	if delta := len(li.messages) - li.keep; delta > 0 {
		li.messages = li.messages[delta:len(li.messages)]
	}
}
