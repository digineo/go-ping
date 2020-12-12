package monitor

import (
	"errors"
	"log"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/digineo/go-ping/internal"
	"golang.org/x/net/icmp"
)

var (
	sequence      = uint16(os.Getpid())
	errPacketLost = errors.New("packet lost")
)

// Monitor manages the goroutines responsible for collecting Ping RTT data.
type Monitor struct {
	HistorySize int // Number of results per target to keep
	PayloadSize uint16
	Privileged  bool

	conn     internal.Conn
	interval time.Duration

	targets  map[string]*target // mapping from external key
	inFlight []*target          // mapping from sequence to target

	mtx  sync.RWMutex
	stop chan struct{}
}

const (
	defaultHistorySize = 10
	defaultPayloadSize = 8
)

// New creates and configures a new Ping instance. You need to call
// AddTarget()/RemoveTarget() to manage monitored targets.
func New(interval time.Duration) *Monitor {
	return &Monitor{
		HistorySize: defaultHistorySize,
		PayloadSize: defaultPayloadSize,
		targets:     make(map[string]*target),
		inFlight:    make([]*target, math.MaxUint16),
		stop:        make(chan struct{}),
		interval:    interval,
	}
}

func (m *Monitor) Start(bind4, bind6 string) error {
	if m.conn.Receiver != nil {
		panic("already started")
	}
	m.conn.Privileged = m.Privileged
	m.conn.Receiver = m.receive

	err := m.conn.Open(bind4, bind6)
	if err != nil {
		return err
	}

	go m.run()

	return nil
}

// Stop brings the monitoring gracefully to a halt.
func (m *Monitor) Stop() {
	close(m.stop)
}

func (m *Monitor) run() {
	ticker := time.NewTicker(m.interval)

	m.pingTargets()
	for {
		select {
		case <-m.stop:
			ticker.Stop()

			return
		case <-ticker.C:
			m.pingTargets()
		}
	}
}

func (m *Monitor) pingTargets() {
	m.mtx.RLock()
	keys := make([]string, len(m.targets))
	for key := range m.targets {
		keys = append(keys, key)
	}
	m.mtx.RUnlock()

	if len(keys) == 0 {
		return
	}

	sleep := m.interval / time.Duration(len(keys))

	for i := range keys {
		select {
		case <-m.stop:
			return
		case <-time.After(sleep):
			m.pingTarget(keys[i])
		}
	}
}

func (m *Monitor) pingTarget(key string) {
	m.mtx.RLock()
	t, found := m.targets[key]
	m.mtx.RUnlock()

	if !found {
		// removed in the meanwhile
		return
	}

	sequence++
	if sequence == 0 {
		// 0 is never used for inFlight requests
		sequence++
	}

	// target lock
	t.Lock()
	if t.seq > 0 {
		// packet lost
		m.inFlight[t.seq] = nil
		t.history.AddResult(0, errPacketLost)
	}
	t.seq = sequence
	t.Unlock()

	t.sent = time.Now()
	m.inFlight[sequence] = t

	err := m.conn.WriteTo(&t.addr, int(sequence), t.payload)
	if err != nil {
		log.Printf("unable to write to %v: %v", t.addr, err)
	}
}

func (m *Monitor) receive(body *icmp.Echo, icmpError error, addr net.IPAddr, tRecv *time.Time) {
	var t *target

	if icmpError != nil {
		// TODO handle error
		return
	}

	seq := uint16(body.Seq)
	t = m.inFlight[seq]

	if t == nil {
		return
	}

	var matches bool

	t.Lock()
	if t.seq == seq {
		matches = true
		t.seq = 0
	}
	t.Unlock()

	// also compare body? bytes.Equal(body.Data, t.payload)
	if matches {
		t.history.AddResult(tRecv.Sub(t.sent), icmpError)
	}
	m.inFlight[seq] = nil
}

// AddTarget adds a target to the monitored list. If the target with the given
// ID already exists, it is removed first and then readded. This allows
// the easy restart of the monitoring.
func (m *Monitor) AddTarget(key string, addr net.IPAddr) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.targets[key] != nil {
		// target already exists
		return
	}

	t := &target{
		addr:    addr,
		history: NewHistory(m.HistorySize),
	}

	t.payload.Resize(m.PayloadSize)

	m.targets[key] = t
}

// RemoveTarget removes a target from the monitoring list.
func (m *Monitor) RemoveTarget(key string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if target, found := m.targets[key]; found {
		delete(m.targets, key)
		m.inFlight[target.seq] = nil
	}
}

// Export calculates the metrics for each monitored target and returns it as a simple map.
func (m *Monitor) Export() map[string]*Metrics {
	result := make(map[string]*Metrics)

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	for id, target := range m.targets {
		if metrics := target.history.Compute(); metrics != nil {
			result[id] = metrics
		}
	}

	return result
}
