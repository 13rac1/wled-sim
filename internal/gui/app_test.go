package gui

import (
	"context"
	"image/color"
	"sync"
	"testing"
	"time"

	"wled-simulator/internal/state"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

func TestSafeFyneDo_RespectsCancelledContext(t *testing.T) {
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	gui := &GUI{
		ctx: ctx,
	}

	// Track if the function was called
	called := false
	gui.SafeFyneDo(func() {
		called = true
	}, false)

	// Function should not have been called due to cancelled context
	if called {
		t.Error("SafeFyneDo should not execute function when context is cancelled")
	}
}

func TestSafeFyneDo_ExecutesWhenSafe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gui := &GUI{
		ctx: ctx,
	}

	// Track if the function was called
	called := false
	gui.SafeFyneDo(func() {
		called = true
	}, false)

	// Function should have been called
	if !called {
		t.Error("SafeFyneDo should execute function when context is active")
	}
}

func TestStop_CleansUpTimers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a test rectangle
	rect := canvas.NewRectangle(color.Black)

	gui := &GUI{
		ctx:         ctx,
		cancel:      cancel,
		flashTimers: make(map[*canvas.Rectangle]*time.Timer),
		wg:          sync.WaitGroup{},
	}

	// Add a timer to the map
	timer := time.NewTimer(time.Hour) // Long timer
	gui.timersMutex.Lock()
	gui.flashTimers[rect] = timer
	gui.timersMutex.Unlock()

	gui.stop()

	// Check that timers map is cleared
	gui.timersMutex.Lock()
	timerCount := len(gui.flashTimers)
	gui.timersMutex.Unlock()

	if timerCount != 0 {
		t.Error("stop() should clear all flash timers")
	}

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		// Expected - context should be cancelled
	default:
		t.Error("stop() should cancel the context")
	}
}

func TestFlashLight_RespectsContext(t *testing.T) {
	// Use headless test app
	testApp := test.NewApp()
	defer testApp.Quit()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ledState := state.NewLEDState(1, "#000000")
	gui := NewApp(testApp, ledState, 1, 1, "row", false)

	// Replace the GUI's context with our cancelled one
	gui.ctx = ctx

	rect := canvas.NewRectangle(color.Black)

	// Try to flash light after context cancellation
	gui.flashLight(rect, color.RGBA{255, 0, 0, 255})

	// Should not have added any timers
	gui.timersMutex.Lock()
	timerCount := len(gui.flashTimers)
	gui.timersMutex.Unlock()

	if timerCount != 0 {
		t.Error("flashLight should not add timers when context is cancelled")
	}
}

func TestConcurrentShutdown(t *testing.T) {
	// This test tries to reproduce race conditions
	testApp := test.NewApp()
	defer testApp.Quit()

	ledState := state.NewLEDState(10, "#000000")
	gui := NewApp(testApp, ledState, 2, 5, "row", false)

	// Start some activity that would normally cause GUI updates
	var wg sync.WaitGroup

	// Simulate multiple concurrent operations (reduced intensity)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				// Try to flash lights concurrently
				gui.flashLight(gui.jsonLightRect, color.RGBA{0, 255, 0, 255})
				gui.flashLight(gui.ddpLightRect, color.RGBA{255, 0, 0, 255})
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}

	// Let operations run for a bit
	time.Sleep(25 * time.Millisecond)

	// Shutdown while operations are running
	gui.stop()

	// Wait for all operations to complete
	wg.Wait()

	// Wait a bit more for any remaining timer callbacks
	time.Sleep(100 * time.Millisecond)

	// Verify context is cancelled
	select {
	case <-gui.ctx.Done():
		// Expected
	default:
		t.Error("context should be cancelled after stop()")
	}

	// Verify timers are cleared (check with mutex protection)
	// Allow for some timing tolerance in tests
	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		gui.timersMutex.Lock()
		timerCount := len(gui.flashTimers)
		gui.timersMutex.Unlock()

		if timerCount == 0 {
			break // Success
		}

		if attempt == maxAttempts-1 {
			t.Errorf("expected 0 timers after shutdown, got %d (after %d attempts)", timerCount, maxAttempts)
		} else {
			// Wait a bit longer and try again
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestUpdateDisplay_RespectsContext(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	ledState := state.NewLEDState(4, "#000000")
	gui := NewApp(testApp, ledState, 2, 2, "row", false)

	// Set a color to verify no update happens
	originalColors := make([]color.Color, len(gui.rectangles))
	for i, rect := range gui.rectangles {
		originalColors[i] = rect.FillColor
	}

	// Cancel context
	gui.cancel()

	// Change LED state
	ledState.SetLED(0, color.RGBA{255, 0, 0, 255})

	// Try to update display
	gui.updateDisplay()

	// Verify colors didn't change (since context is cancelled)
	for i, rect := range gui.rectangles {
		if rect.FillColor != originalColors[i] {
			t.Error("updateDisplay should not change colors when context is cancelled")
		}
	}
}

func TestTimerCallbackRaceCondition(t *testing.T) {
	// This test specifically targets the timer callback race condition
	testApp := test.NewApp()
	defer testApp.Quit()

	ledState := state.NewLEDState(1, "#000000")
	gui := NewApp(testApp, ledState, 1, 1, "row", false)

	rect := canvas.NewRectangle(color.Black)

	// Start a flash with a very short timer
	originalFlashTimers := gui.flashTimers
	gui.flashTimers = make(map[*canvas.Rectangle]*time.Timer)

	// Create a timer that will fire very soon
	gui.flashTimers[rect] = time.AfterFunc(1*time.Millisecond, func() {
		gui.SafeFyneDo(func() {
			rect.FillColor = color.RGBA{128, 128, 128, 255}
			rect.Refresh()
		}, true)
		gui.timersMutex.Lock()
		delete(gui.flashTimers, rect)
		gui.timersMutex.Unlock()
	})

	// Immediately shutdown
	gui.stop()

	// Wait for timer to potentially fire
	time.Sleep(10 * time.Millisecond)

	// Verify timers are cleaned up
	gui.timersMutex.Lock()
	timerCount := len(gui.flashTimers)
	gui.timersMutex.Unlock()

	if timerCount != 0 {
		t.Error("Timer should be cleaned up after stop()")
	}

	// Restore original timers
	gui.flashTimers = originalFlashTimers
}

func BenchmarkSafeFyneDo(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gui := &GUI{
		ctx: ctx,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gui.SafeFyneDo(func() {
				// Minimal work
			}, false)
		}
	})
}

func BenchmarkSafeFyneDoWithCancelledContext(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	gui := &GUI{
		ctx: ctx,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gui.SafeFyneDo(func() {
				// This should not execute
			}, false)
		}
	})
}
