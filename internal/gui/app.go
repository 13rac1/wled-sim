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
}

func NewApp(app fyne.App, s *state.LEDState, leds int, controls bool) *GUI {
	gui := &GUI{
		app:        app,
		state:      s,
		rectangles: make([]*canvas.Rectangle, leds*2),
	}
	gui.window = app.NewWindow("WLED Simulator")

	grid := container.NewGridWrap(fyne.NewSize(20, 20))
	for i := 0; i < leds*2; i++ {
		rect := canvas.NewRectangle(color.Black)
		gui.rectangles[i] = rect
		grid.Add(rect)
	}

	gui.window.SetContent(grid)

	// Default close behavior just quits the app
	gui.window.SetCloseIntercept(func() {
		gui.app.Quit()
	})

	// Start update loop
	go gui.updateLoop()

	return gui
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
		for i, led := range leds {
			if i < len(g.rectangles) {
				g.rectangles[i].FillColor = led
				g.rectangles[i].Refresh()
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
