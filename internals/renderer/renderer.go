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

func GetBrailleChar(matrix [4][2]bool) rune {
	var r rune = 0x2800
	if matrix[0][0] {
		r |= 0x01
	} // Dot 1
	if matrix[1][0] {
		r |= 0x02
	} // Dot 2
	if matrix[2][0] {
		r |= 0x04
	} // Dot 3
	if matrix[0][1] {
		r |= 0x08
	} // Dot 4
	if matrix[1][1] {
		r |= 0x10
	} // Dot 5
	if matrix[2][1] {
		r |= 0x20
	} // Dot 6
	if matrix[3][0] {
		r |= 0x40
	} // Dot 7
	if matrix[3][1] {
		r |= 0x80
	} // Dot 8
	return r
}

// braillePixels is a flat pixel buffer at 2x terminal width, 4x terminal height.
var braillePixels [][]bool
var brailleW, brailleH int

func initBrailleBuffer(s tcell.Screen) {
	w, h := s.Size()
	brailleW, brailleH = w*2, h*4
	braillePixels = make([][]bool, brailleH)
	for i := range braillePixels {
		braillePixels[i] = make([]bool, brailleW)
	}
}

func clearBraille() {
	for y := range braillePixels {
		for x := range braillePixels[y] {
			braillePixels[y][x] = false
		}
	}
}

func setBraillePixel(x, y int) {
	if x >= 0 && x < brailleW && y >= 0 && y < brailleH {
		braillePixels[y][x] = true
	}
}

func flushBraille(s tcell.Screen, style tcell.Style) {
	termW := brailleW / 2
	termH := brailleH / 4
	for ty := 0; ty < termH; ty++ {
		for tx := 0; tx < termW; tx++ {
			var matrix [4][2]bool
			for row := 0; row < 4; row++ {
				for col := 0; col < 2; col++ {
					matrix[row][col] = braillePixels[ty*4+row][tx*2+col]
				}
			}
			ch := GetBrailleChar(matrix)
			if ch != 0x2800 {
				s.SetContent(tx, ty, ch, nil, style)
			}
		}
	}
}

func translateCoordinate(s tcell.Screen, x, y float64) (int, int) {
	width, height := s.Size()
	xp := (x + 1) / 2 * float64(width)
	yp := (1 - (y+1)/2) * float64(height)
	return int(xp), int(yp)
}

// translateCoordinateBraille maps normalized [-1,1] to braille pixel space.
func translateCoordinateBraille(x, y float64) (int, int) {
	xp := (x + 1) / 2 * float64(brailleW)
	yp := (1 - (y+1)/2) * float64(brailleH)
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
	boxStyle := tcell.StyleDefault.Foreground(color.Blue).Background(color.Reset)

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
		maybePanic := recover()
		s.Fini()
		if maybePanic != nil {
			panic(maybePanic)
		}
	}
	defer quit()

	initBrailleBuffer(s)

	go handleExit(s)
	gameLoop(s, boxStyle)
}

func handleExit(s tcell.Screen) {
	for {
		ev := <-s.EventQ()
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

// drawLineBraille is like drawLine but writes into the braille pixel buffer.
func drawLineBraille(x1, y1, x2, y2 int) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)

	steps := math.Abs(dx)
	if math.Abs(dy) > steps {
		steps = math.Abs(dy)
	}

	if steps == 0 {
		setBraillePixel(x1, y1)
		return
	}

	xInc := dx / steps
	yInc := dy / steps

	currentX := float64(x1)
	currentY := float64(y1)

	for i := 0; i <= int(steps); i++ {
		setBraillePixel(int(math.Round(currentX)), int(math.Round(currentY)))
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

		x1, y1 := translateCoordinateBraille(px1, py1)
		x2, y2 := translateCoordinateBraille(px2, py2)

		drawLineBraille(x1, y1, x2, y2)
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
		clearBraille()

		x1, y1 := translateCoordinate(s, x, y)
		drawLine(s, x1, y1, x1+5, y1+5, style)

		drawPoint(s, x1, y1, style)
		drawBox(s, x1, y1, x1+5, y1+5, style)
		x = x - 0.01
		angle = angle + 0.05
		cube := rotatedCube(cubeEdges, angle)
		drawCube(s, cube, style)
		flushBraille(s, style)

		s.Show()
		time.Sleep(frameDuration)
	}
}
