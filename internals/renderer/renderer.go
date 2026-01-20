package renderer

import (
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

var FPS = 60

func translateCoordinate(s tcell.Screen, x, y float64) (int, int) {
	// -1, 1 => 0.. width/hight
	width, height := s.Size()
	xp := (x + 1) / 2 * float64(width)
	yp := (1 - (y+1)/2) * float64(height)

	return int(xp), int(yp)
}

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {

	row := y1
	col := x1
	var width int
	for text != "" {
		text, width = s.Put(col, row, text, style)
		col += width
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
		if width == 0 {
			// incomplete grapheme at end of string
			break
		}
	}
}

func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	for col := x1; col <= x2; col++ {
		s.Put(col, y1, string(tcell.RuneHLine), style)
		s.Put(col, y2, string(tcell.RuneHLine), style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.Put(x1, row, string(tcell.RuneVLine), style)
		s.Put(x2, row, string(tcell.RuneBullet), style)
	}

}

func drawPoint(s tcell.Screen, x, y int, style tcell.Style) {
	s.Put(x, y, string(tcell.RuneBullet), style)

}
func Render() {
	defStyle := tcell.StyleDefault.Background(color.Reset).Foreground(color.Reset)
	boxStyle := tcell.StyleDefault.Foreground(color.Reset).Background(color.Reset)

	// Initialize screen
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	quit := func() {
		// You have to catch panics in a defer, clean up, and
		// re-raise them - otherwise your application can
		// die without leaving any diagnostic trace.
		maybePanic := recover()
		s.Fini()
		if maybePanic != nil {
			panic(maybePanic)
		}
	}
	defer quit()

	// Here's how to get the screen size when you need it.
	// xmax, ymax := s.Size()

	// Here's an example of how to inject a keystroke where it will
	// be picked up by a future read of the event queue.  Note that
	// care should be used to avoid blocking writes to the queue if
	// this is done from the same thread that is responsible for reading
	// the queue, or else a single-party deadlock might occur.
	// s.EventQ() <- tcell.NewEventKey(tcell.KeyRune, rune('a'), 0)

	// Event loop
	go handleExit(s)
	gameLoop(s, boxStyle)

}

func handleExit(s tcell.Screen) {
	for {

		ev := <-s.EventQ()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				s.Clear()
				s.Fini()
				os.Exit(0)
			}

		}
	}
}
func gameLoop(s tcell.Screen, style tcell.Style) {
	x, y := 0.1, 0.1

	for {

		s.Show()
		x1, y1 := translateCoordinate(s, x, y)
		time.Sleep(time.Millisecond * time.Duration(FPS))

		drawPoint(s, x1, y1, style)
		x = x - 0.01
	}
}
