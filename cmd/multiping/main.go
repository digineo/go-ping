package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	ping "github.com/digineo/go-ping"
)

type stats struct {
	received int
	lost     int
	results  []time.Duration // ring buffer, index = .received
	lastErr  error
	mtx      sync.RWMutex
}

type unit struct {
	host    string
	remote  *net.IPAddr
	display string
	*stats
}

var opts = struct {
	timeout        time.Duration
	interval       time.Duration
	statBufferSize uint
	bind           string
	remotes        []unit

	pinger *ping.Pinger
	table  *tview.Table
}{
	timeout:        1000 * time.Millisecond,
	interval:       1000 * time.Millisecond,
	bind:           "0.0.0.0",
	statBufferSize: 50,
}

func main() {
	flag.DurationVar(&opts.timeout, "timeout", opts.timeout, "timeout for a single echo request")
	flag.DurationVar(&opts.interval, "interval", opts.interval, "polling interval")
	flag.UintVar(&opts.statBufferSize, "buf", opts.statBufferSize, "buffer size for statistics")
	flag.StringVar(&opts.bind, "bind", opts.bind, "bind address")
	flag.Parse()

	log.SetFlags(0)

	for _, arg := range flag.Args() {
		if remote, err := net.ResolveIPAddr("ip4", arg); err == nil {
			u := unit{
				host:   arg,
				remote: remote,
				stats: &stats{
					results: make([]time.Duration, opts.statBufferSize),
				},
			}

			if arg == remote.String() {
				u.display = arg
			} else {
				u.display = fmt.Sprintf("%s (%s)", arg, remote)
			}

			opts.remotes = append(opts.remotes, u)
		} else {
			log.Printf("host %s: %v", arg, err)
		}
	}

	if instance, err := ping.New(opts.bind); err == nil {
		instance.Timeout = opts.timeout
		instance.Attempts = 1
		opts.pinger = instance
		defer opts.pinger.Close()
	} else {
		panic(err)
	}

	go work()

	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(false).SetFixed(2, 0)
	table.SetTitle(" multiping (press [q] to exit) ")
	opts.table = table
	cols := 9

	table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("%-60s", "Host")).SetAlign(tview.AlignLeft))
	table.SetCell(0, 1, tview.NewTableCell("  sent").SetAlign(tview.AlignRight))
	table.SetCell(0, 2, tview.NewTableCell("  loss").SetAlign(tview.AlignRight))
	table.SetCell(0, 3, tview.NewTableCell("  last").SetAlign(tview.AlignRight))
	table.SetCell(0, 4, tview.NewTableCell("  best").SetAlign(tview.AlignRight))
	table.SetCell(0, 5, tview.NewTableCell("  worst").SetAlign(tview.AlignRight))
	table.SetCell(0, 6, tview.NewTableCell("  mean").SetAlign(tview.AlignRight))
	table.SetCell(0, 7, tview.NewTableCell("  stddev").SetAlign(tview.AlignRight))
	table.SetCell(0, 8, tview.NewTableCell("last err").SetAlign(tview.AlignLeft))

	for r, u := range opts.remotes {
		for c := 0; c < cols; c++ {
			var cell *tview.TableCell
			switch c {
			case 0:
				cell = tview.NewTableCell(u.display).SetAlign(tview.AlignLeft)
			case 8:
				cell = tview.NewTableCell("").SetAlign(tview.AlignLeft)
			default:
				cell = tview.NewTableCell("n/a").SetAlign(tview.AlignRight)
			}
			table.SetCell(r+2, c, cell)
		}
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				app.Stop()
				return nil
			}
		}
		return event
	})

	go func() {
		time.Sleep(time.Second)
		for {
			for i, u := range opts.remotes {
				sent, lost, last, best, worst, mean, stddev := u.compute()
				r := i + 2

				opts.table.GetCell(r, 1).SetText(strconv.Itoa(sent))
				opts.table.GetCell(r, 2).SetText(fmt.Sprintf("%0.2f%%", lost))
				opts.table.GetCell(r, 3).SetText(ts(last))
				opts.table.GetCell(r, 4).SetText(ts(best))
				opts.table.GetCell(r, 5).SetText(ts(worst))
				opts.table.GetCell(r, 6).SetText(ts(mean))
				opts.table.GetCell(r, 7).SetText(stddev.String())

				if u.lastErr != nil {
					opts.table.GetCell(r, 8).SetText(fmt.Sprintf("%v", u.lastErr))
				}
			}
			app.Draw()
			time.Sleep(time.Second)
		}
	}()

	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}
}

func work() {
	for {
		for i, u := range opts.remotes {
			go func(u unit, i int) {
				time.Sleep(time.Duration(i) * time.Millisecond)
				u.ping(opts.pinger)
			}(u, i)
		}
		time.Sleep(opts.interval)
	}
}

const tsDividend = float64(time.Millisecond) / float64(time.Nanosecond)

func ts(dur time.Duration) string {
	if 10*time.Microsecond < dur && dur < time.Second {
		return fmt.Sprintf("%0.2fms", float64(dur.Nanoseconds())/tsDividend)
	}
	return dur.String()
}

func (u *unit) ping(pinger *ping.Pinger) {
	u.addResult(pinger.PingRTT(u.remote))
}

func (s *stats) addResult(rtt time.Duration, err error) {
	s.mtx.Lock()
	if err == nil {
		s.results[s.received%len(s.results)] = rtt
		s.received++
	} else {
		s.lastErr = err
		s.lost++
	}
	s.mtx.Unlock()
}

func (s *stats) compute() (pktSent int, pktLoss float64, last, best, worst, mean, stddev time.Duration) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	if s.received == 0 {
		if s.lost > 0 {
			pktLoss = 1.0
		}
		return
	}

	collection := s.results[:]
	pktSent = s.received + s.lost
	size := len(s.results)

	if s.received < size {
		collection = s.results[:s.received]
		size = s.received
	}

	last = collection[s.received%size]
	pktLoss = float64(s.lost) / float64(size)
	best, worst = collection[0], collection[0]

	total := time.Duration(0)
	for _, rtt := range collection {
		if rtt < best {
			best = rtt
		}
		if rtt > worst {
			worst = rtt
		}
		total += rtt
	}

	mean = time.Duration(float64(total) / float64(s.received))

	stddevNum := float64(0)
	for _, rtt := range collection {
		stddevNum += math.Pow(float64(rtt-mean), 2)
	}
	stddev = time.Duration(math.Sqrt(stddevNum / float64(mean)))

	return
}
