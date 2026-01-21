package renderer

import (
	"log"
	"math"
	"os"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

var FPS = 60
var DELTATIME = 0.6

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
	boxStyle := tcell.StyleDefault.Foreground(color.Reset).Background(color.Blue)

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

func project(x, y, z float64) (float64, float64) {
	if z == 0 {
		z = 1
	}
	return x / z, y / z
}

func drawLine(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)

	// Determine how many steps we need (DDA algorithm style)
	steps := math.Abs(dx)
	if math.Abs(dy) > steps {
		steps = math.Abs(dy)
	}

	if steps == 0 {
		s.Put(x1, y1, ".", style)
		return
	}

	xInc := dx / steps
	yInc := dy / steps

	currentX := float64(x1)
	currentY := float64(y1)

	for i := 0; i <= int(steps); i++ {
		s.Put(int(math.Round(currentX)), int(math.Round(currentY)), ".", style)
		currentX += xInc
		currentY += yInc
	}
}

func drawCube(s tcell.Screen, edges [][]float64, style tcell.Style) {
	connections := [][]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, // Back face
		{4, 5}, {5, 6}, {6, 7}, {7, 4}, // Front face
		{0, 4}, {1, 5}, {2, 6}, {3, 7}, // Connecting struts
	}

	for _, conn := range connections {
		v1, v2 := edges[conn[0]], edges[conn[1]]

		px1, py1 := project(v1[0], v1[1], v1[2]+2.0)
		px2, py2 := project(v2[0], v2[1], v2[2]+2.0)

		x1, y1 := translateCoordinate(s, px1, py1)
		x2, y2 := translateCoordinate(s, px2, py2)

		drawLine(s, x1, y1, x2, y2, style)
	}
}

func rotateXZ(point []float64, angle float64) []float64 {
	x, y, z := point[0], point[1], point[2]
	cos := math.Cos(angle)
	sin := math.Sin(angle)

	x1 := x*cos - z*sin
	z1 := x*sin + z*cos
	return []float64{x1, y, z1}
}

func rotatedCube(src [][]float64, angle float64) [][]float64 {
	dst := make([][]float64, len(src))
	for i := range src {
		dst[i] = rotateXZ(src[i], angle)
	}
	return dst
}
func gameLoop(s tcell.Screen, style tcell.Style) {
	frameDuration := time.Second / 60
	x, y := 0.1, 0.1
	cubeEdges := [][]float64{
		{-0.5, -0.5, -0.5}, // 0
		{0.5, -0.5, -0.5},  // 1
		{0.5, 0.5, -0.5},   // 2
		{-0.5, 0.5, -0.5},  // 3

		{-0.5, -0.5, 0.5}, // 4
		{0.5, -0.5, 0.5},  // 5
		{0.5, 0.5, 0.5},   // 6
		{-0.5, 0.5, 0.5},  // 7
	}
	angle := 0.05
	for {
		s.Clear()

		x1, y1 := translateCoordinate(s, x, y)
		drawLine(s, x1, y1, x1+5, y1+5, style)

		drawPoint(s, x1, y1, style)
		drawBox(s, x1, y1, x1+5, y1+5, style)
		x = x - 0.01
		angle = angle + 0.05
		cube := rotatedCube(cubeEdges, angle)
		drawCube(s, cube, style)

		s.Show()
		time.Sleep(frameDuration)

	}
}
