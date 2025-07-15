package gui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"os/signal"
	"sync"
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
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewApp(app fyne.App, s *state.LEDState, rows, cols int, wiring string, controls bool) *GUI {
	totalLEDs := rows * cols
	ctx, cancel := context.WithCancel(context.Background())

	gui := &GUI{
		app:        app,
		state:      s,
		rectangles: make([]*canvas.Rectangle, totalLEDs),
		rows:       rows,
		cols:       cols,
		wiring:     wiring,
		ctx:        ctx,
		cancel:     cancel,
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
		gui.stop()
		gui.app.Quit()
	})

	// Start update loop
	gui.wg.Add(1)
	go gui.updateLoop()

	return gui
}

// stop cancels the context and waits for goroutines to finish
func (g *GUI) stop() {
	g.cancel()
	g.wg.Wait()
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
	defer g.wg.Done()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			// Context cancelled, stop updating
			return
		case <-ticker.C:
			g.updateDisplay()
		}
	}
}

// updateDisplay updates all rectangles from the current LED state
func (g *GUI) updateDisplay() {
	// Check if context is cancelled before attempting GUI operations
	select {
	case <-g.ctx.Done():
		return
	default:
	}

	leds := g.state.LEDs()

	// Use fyne.Do instead of fyne.DoAndWait to avoid blocking if app is quitting
	fyne.Do(func() {
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
	g.window.SetCloseIntercept(func() {
		g.stop()
		handler()
	})
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
		g.stop()
		fyne.DoAndWait(func() {
			g.app.Quit()
		})
	}()

	g.window.ShowAndRun()
	fmt.Println("GUI: Window closed")
}
