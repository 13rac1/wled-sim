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
	"fyne.io/fyne/v2/widget"
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
	// Activity lights
	jsonLightRect *canvas.Rectangle
	ddpLightRect  *canvas.Rectangle
	flashTimers   map[*canvas.Rectangle]*time.Timer
	timersMutex   sync.Mutex // Protect flashTimers map
}

// safeFyneDo safely executes a function with fyne.Do, checking context
func (g *GUI) safeFyneDo(fn func()) {
	select {
	case <-g.ctx.Done():
		// Context cancelled, don't update GUI
		return
	default:
		// Safe to update GUI
		fyne.Do(fn)
	}
}

func NewApp(app fyne.App, s *state.LEDState, rows, cols int, wiring string, controls bool) *GUI {
	totalLEDs := rows * cols
	ctx, cancel := context.WithCancel(context.Background())

	gui := &GUI{
		app:         app,
		state:       s,
		rectangles:  make([]*canvas.Rectangle, totalLEDs),
		rows:        rows,
		cols:        cols,
		wiring:      wiring,
		ctx:         ctx,
		cancel:      cancel,
		flashTimers: make(map[*canvas.Rectangle]*time.Timer),
	}
	gui.window = app.NewWindow("WLED Simulator")

	// Create activity lights using canvas.Rectangle with grey fill and black stroke
	gui.jsonLightRect = canvas.NewRectangle(color.RGBA{128, 128, 128, 255})
	gui.jsonLightRect.StrokeColor = color.Black
	gui.jsonLightRect.StrokeWidth = 1

	gui.ddpLightRect = canvas.NewRectangle(color.RGBA{128, 128, 128, 255})
	gui.ddpLightRect.StrokeColor = color.Black
	gui.ddpLightRect.StrokeWidth = 1

	// Create labels with smaller font size for status information using canvas.Text
	jsonLabel := canvas.NewText("JSON", color.RGBA{100, 100, 100, 255})
	jsonLabel.TextSize = 10
	jsonLabel.Alignment = fyne.TextAlignLeading

	ddpLabel := canvas.NewText("DDP", color.RGBA{100, 100, 100, 255})
	ddpLabel.TextSize = 10
	ddpLabel.Alignment = fyne.TextAlignLeading

	// Create containers for the rectangle objects with proper sizing
	jsonLightContainer := container.NewWithoutLayout(gui.jsonLightRect)
	gui.jsonLightRect.Resize(fyne.NewSize(12, 12))
	gui.jsonLightRect.Move(fyne.NewPos(0, 0))
	jsonLightContainer.Resize(fyne.NewSize(12, 12))

	ddpLightContainer := container.NewWithoutLayout(gui.ddpLightRect)
	gui.ddpLightRect.Resize(fyne.NewSize(12, 12))
	gui.ddpLightRect.Move(fyne.NewPos(0, 0))
	ddpLightContainer.Resize(fyne.NewSize(12, 12))

	// Create containers for the text labels with proper sizing
	jsonLabelContainer := container.NewWithoutLayout(jsonLabel)
	jsonLabel.Resize(fyne.NewSize(40, 12))
	jsonLabel.Move(fyne.NewPos(10, 0))
	jsonLabelContainer.Resize(fyne.NewSize(40, 12))

	ddpLabelContainer := container.NewWithoutLayout(ddpLabel)
	ddpLabel.Resize(fyne.NewSize(40, 12))
	ddpLabel.Move(fyne.NewPos(10, 0))
	ddpLabelContainer.Resize(fyne.NewSize(40, 12))

	// Create horizontal containers to align labels with lights in a status bar layout
	jsonContainer := container.NewHBox(
		jsonLightContainer,
		jsonLabelContainer,
	)

	ddpContainer := container.NewHBox(
		ddpLightContainer,
		ddpLabelContainer,
	)

	// Create the activity container as a horizontal status bar
	activityContainer := container.NewHBox(
		jsonContainer,
		widget.NewLabel("    "), // Spacer between groups
		ddpContainer,
	)

	// Create a resizable grid container for LEDs
	grid := container.NewGridWithColumns(cols)

	// Add rectangles in row-major order for display (left-to-right, top-to-bottom)
	ledSize := float32(16) // 16x16 pixel LEDs
	for i := 0; i < totalLEDs; i++ {
		rect := canvas.NewRectangle(color.Black)
		rect.Resize(fyne.NewSize(ledSize, ledSize))
		gui.rectangles[i] = rect
		grid.Add(rect)
	}

	// Calculate grid size and wrap in a resizable container
	gridWidth := float32(cols) * ledSize
	gridHeight := float32(rows) * ledSize

	// Use a simple container that allows the grid to be resizable
	gridContainer := container.NewBorder(nil, nil, nil, nil, grid)

	// Create main container with activity lights at top and LED grid below
	mainContainer := container.NewBorder(
		activityContainer, // top
		nil,               // bottom
		nil,               // left
		nil,               // right
		gridContainer,     // center (resizable)
	)

	gui.window.SetContent(mainContainer)

	// Calculate proper window size based on the actual grid content
	activityHeight := float32(35) // Height for activity lights area
	padding := float32(20)        // Padding around the grid

	// Set window size based on grid dimensions with some spacing
	windowWidth := gridWidth + padding
	if windowWidth < 120 { // Minimum width for activity lights
		windowWidth = 120
	}

	gui.window.Resize(fyne.NewSize(windowWidth, gridHeight+activityHeight+padding))

	// Set up graceful shutdown on window close
	gui.window.SetCloseIntercept(func() {
		fmt.Println("GUI: Window closing, shutting down gracefully...")
		gui.stop()
		gui.app.Quit()
	})

	// Start update loop
	gui.wg.Add(1)
	go gui.updateLoop()

	// Start activity monitoring
	gui.wg.Add(1)
	go gui.monitorActivity()

	return gui
}

// stop cancels the context and waits for goroutines to finish
func (g *GUI) stop() {
	g.cancel()

	// Clean up flash timers (with mutex protection)
	g.timersMutex.Lock()
	for light, timer := range g.flashTimers {
		timer.Stop()
		delete(g.flashTimers, light)
	}
	// Clear the map completely
	g.flashTimers = make(map[*canvas.Rectangle]*time.Timer)
	g.timersMutex.Unlock()

	// Wait longer for any in-flight timer callbacks to complete
	time.Sleep(200 * time.Millisecond)

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

	// Use safeFyneDo wrapper to avoid race conditions during shutdown
	g.safeFyneDo(func() {
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

	// Handle shutdown signal in a separate goroutine
	go func() {
		<-c
		fmt.Println("GUI: Received shutdown signal, closing window...")
		// Use fyne.DoAndWait to ensure window close happens in UI thread
		fyne.DoAndWait(func() {
			g.window.Close()
		})
	}()

	g.window.ShowAndRun()
	fmt.Println("GUI: Window closed")
}

// monitorActivity monitors activity events and flashes the appropriate lights
func (g *GUI) monitorActivity() {
	defer g.wg.Done()

	for {
		select {
		case <-g.ctx.Done():
			return
		case event := <-g.state.ActivityChannel():
			g.handleActivityEvent(event)
		}
	}
}

// handleActivityEvent processes an activity event and flashes the appropriate light
func (g *GUI) handleActivityEvent(event state.ActivityEvent) {
	var light *canvas.Rectangle
	switch event.Type {
	case state.ActivityJSON:
		light = g.jsonLightRect
	case state.ActivityDDP:
		light = g.ddpLightRect
	}

	if light != nil {
		if event.Success {
			g.flashLight(light, color.RGBA{0, 255, 0, 255}) // Bright green for success
		} else {
			g.flashLight(light, color.RGBA{255, 0, 0, 255}) // Bright red for failure
		}
	}
}

// flashLight flashes a light with the specified color for a brief moment
func (g *GUI) flashLight(light *canvas.Rectangle, flashColor color.RGBA) {
	// Check context before starting any timer operations
	select {
	case <-g.ctx.Done():
		return
	default:
	}

	// Cancel any existing timer for this light (with mutex protection)
	g.timersMutex.Lock()
	if timer, exists := g.flashTimers[light]; exists {
		timer.Stop()
	}
	g.timersMutex.Unlock()

	// Change to flash color immediately
	g.safeFyneDo(func() {
		light.FillColor = flashColor
		light.Refresh()
	})

	// Set timer to revert to inactive color (longer duration for visibility)
	timer := time.AfterFunc(500*time.Millisecond, func() {
		select {
		case <-g.ctx.Done():
			// Context cancelled, just clean up timer
			g.timersMutex.Lock()
			delete(g.flashTimers, light)
			g.timersMutex.Unlock()
			return
		default:
		}

		g.safeFyneDo(func() {
			light.FillColor = color.RGBA{128, 128, 128, 255} // Gray (inactive)
			light.Refresh()
		})
		// Clean up timer from map (with mutex protection)
		g.timersMutex.Lock()
		delete(g.flashTimers, light)
		g.timersMutex.Unlock()
	})

	// Add timer to map (with mutex protection)
	g.timersMutex.Lock()
	g.flashTimers[light] = timer
	g.timersMutex.Unlock()
}
