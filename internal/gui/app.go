package gui

import (
	"fmt"
	"image/color"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wled-simulator/internal/state"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

type GUI struct {
	app        fyne.App
	window     fyne.Window
	rectangles []*canvas.Rectangle
	state      *state.LEDState
	rows       int
	cols       int
	wiring     string
}

func NewApp(app fyne.App, s *state.LEDState, rows, cols int, wiring string, controls bool) *GUI {
	totalLEDs := rows * cols
	gui := &GUI{
		app:        app,
		state:      s,
		rectangles: make([]*canvas.Rectangle, totalLEDs),
		rows:       rows,
		cols:       cols,
		wiring:     wiring,
	}
	gui.window = app.NewWindow("WLED Simulator")

	// Create a grid container with the specified number of columns
	grid := container.NewGridWithColumns(cols)

	// Add rectangles in row-major order for display (left-to-right, top-to-bottom)
	for i := 0; i < totalLEDs; i++ {
		rect := canvas.NewRectangle(color.Black)
		rect.Resize(fyne.NewSize(20, 20))
		gui.rectangles[i] = rect
		grid.Add(rect)
	}

	gui.window.SetContent(grid)
	gui.window.Resize(fyne.NewSize(float32(cols*25), float32(rows*25)))

	// Default close behavior just quits the app
	gui.window.SetCloseIntercept(func() {
		gui.app.Quit()
	})

	// Start update loop
	go gui.updateLoop()

	return gui
}

// ledIndexToGridPosition converts a linear LED index to grid position based on wiring pattern
func (g *GUI) ledIndexToGridPosition(ledIndex int) (row, col int) {
	if g.wiring == "col" {
		// Column-major: LEDs go top-to-bottom, then left-to-right
		row = ledIndex % g.rows
		col = ledIndex / g.rows
	} else {
		// Row-major: LEDs go left-to-right, then top-to-bottom (default)
		row = ledIndex / g.cols
		col = ledIndex % g.cols
	}
	return row, col
}

// gridPositionToDisplayIndex converts grid position to display rectangle index
func (g *GUI) gridPositionToDisplayIndex(row, col int) int {
	// Display is always row-major (left-to-right, top-to-bottom)
	return row*g.cols + col
}

// updateLoop periodically updates the LED display
func (g *GUI) updateLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		g.updateDisplay()
	}
}

// updateDisplay updates all rectangles from the current LED state
func (g *GUI) updateDisplay() {
	leds := g.state.LEDs()
	fyne.DoAndWait(func() {
		for ledIndex, ledColor := range leds {
			if ledIndex < len(leds) {
				// Convert LED index to grid position based on wiring
				row, col := g.ledIndexToGridPosition(ledIndex)

				// Convert grid position to display rectangle index
				displayIndex := g.gridPositionToDisplayIndex(row, col)

				if displayIndex < len(g.rectangles) {
					g.rectangles[displayIndex].FillColor = ledColor
					g.rectangles[displayIndex].Refresh()
				}
			}
		}
	})
}

// SetOnClose sets a custom close handler for the window
func (g *GUI) SetOnClose(handler func()) {
	g.window.SetCloseIntercept(handler)
}

func (g *GUI) Run() {
	fmt.Println("GUI: Showing window...")

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Handle shutdown signal
	go func() {
		<-c
		fmt.Println("GUI: Received shutdown signal, quitting application...")
		fyne.DoAndWait(func() {
			g.app.Quit()
		})
	}()

	g.window.ShowAndRun()
	fmt.Println("GUI: Window closed")
}
