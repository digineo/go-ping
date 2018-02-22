package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type userInterface struct {
	app          *tview.Application
	table        *tview.Table
	destinations []*destination
}

func buildTUI(destinations []*destination) *userInterface {
	ui := &userInterface{
		app:          tview.NewApplication(),
		table:        tview.NewTable().SetBorders(false).SetFixed(2, 0),
		destinations: destinations,
	}

	ui.table.SetTitle(" multiping (press [q] to exit) ")

	ui.table.SetCell(0, 0, tview.NewTableCell("host").SetAlign(tview.AlignLeft))
	ui.table.SetCell(0, 1, tview.NewTableCell("address").SetAlign(tview.AlignLeft))
	ui.table.SetCell(0, 2, tview.NewTableCell("sent").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 3, tview.NewTableCell("loss").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 4, tview.NewTableCell("last").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 5, tview.NewTableCell("best").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 6, tview.NewTableCell("worst").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 7, tview.NewTableCell("mean").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 8, tview.NewTableCell("stddev").SetAlign(tview.AlignRight))
	ui.table.SetCell(0, 9, tview.NewTableCell("last err").SetAlign(tview.AlignLeft))

	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			ui.app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				ui.app.Stop()
				return nil
			}
		}
		return event
	})

	cols := 10
	for r, u := range destinations {
		for c := 0; c < cols; c++ {
			var cell *tview.TableCell
			switch c {
			case 0:
				cell = tview.NewTableCell(u.host).SetAlign(tview.AlignLeft)
			case 1:
				cell = tview.NewTableCell(u.remote.IP.String()).SetAlign(tview.AlignLeft)
			case 9:
				cell = tview.NewTableCell("").SetAlign(tview.AlignLeft)
			default:
				cell = tview.NewTableCell("n/a").SetAlign(tview.AlignRight)
			}
			ui.table.SetCell(r+2, c, cell)
		}
	}

	return ui
}

func (ui *userInterface) Run() error {
	ui.app.SetRoot(ui.table, true).SetFocus(ui.table)
	return ui.app.Run()
}

func (ui *userInterface) update(interval time.Duration) {
	time.Sleep(interval)
	for {
		for i, u := range ui.destinations {
			sent, loss, last, best, worst, mean, stddev := u.compute()
			r := i + 2

			ui.table.GetCell(r, 2).SetText(strconv.Itoa(sent))
			ui.table.GetCell(r, 3).SetText(fmt.Sprintf("%0.2f%%", loss))
			ui.table.GetCell(r, 4).SetText(ts(last))
			ui.table.GetCell(r, 5).SetText(ts(best))
			ui.table.GetCell(r, 6).SetText(ts(worst))
			ui.table.GetCell(r, 7).SetText(ts(mean))
			ui.table.GetCell(r, 8).SetText(stddev.String())

			if u.lastErr != nil {
				ui.table.GetCell(r, 8).SetText(fmt.Sprintf("%v", u.lastErr))
			}
		}
		ui.app.Draw()
		time.Sleep(interval)
	}
}

const tsDividend = float64(time.Millisecond) / float64(time.Nanosecond)

func ts(dur time.Duration) string {
	if 10*time.Microsecond < dur && dur < time.Second {
		return fmt.Sprintf("%0.2fms", float64(dur.Nanoseconds())/tsDividend)
	}
	return dur.String()
}
