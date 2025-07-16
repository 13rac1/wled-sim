package ddp

import (
	"strings"
	"testing"
	"time"

	"wled-simulator/internal/state"
)

func TestServerSetVerbose(t *testing.T) {
	s := NewServer(4048, state.NewLEDState(10, "#000000"))

	// Default should not be verbose
	if s.verbose {
		t.Error("Expected default verbose to be false")
	}

	s.SetVerbose(false)
	if s.verbose {
		t.Error("Expected verbose to be false after SetVerbose(false)")
	}

	s.SetVerbose(true)
	if !s.verbose {
		t.Error("Expected verbose to be true after SetVerbose(true)")
	}
}

func TestServerStop(t *testing.T) {
	s := NewServer(4048, state.NewLEDState(10, "#000000"))

	// Test stopping without starting
	err := s.Stop()
	if err != nil {
		t.Errorf("Unexpected error stopping server that was never started: %v", err)
	}

	// Check context is cancelled
	select {
	case <-s.ctx.Done():
		// Expected
	default:
		t.Error("Expected context to be cancelled after Stop()")
	}
}

func TestPortCollision(t *testing.T) {
	// Use a specific port for testing
	const testPort = 4049
	ledState := state.NewLEDState(10, "#000000")

	// Start first server
	srv1 := NewServer(testPort, ledState)
	go func() {
		err := srv1.Start()
		if err != nil {
			t.Errorf("First server failed unexpectedly: %v", err)
		}
	}()

	// Give the first server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to start second server on same port
	srv2 := NewServer(testPort, ledState)
	err := srv2.Start()

	// Verify we get the expected error
	if err == nil {
		t.Fatal("Expected error when starting server on occupied port")
	}

	expectedErrMsg := "bind: address already in use"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErrMsg, err)
	}

	// Cleanup
	srv1.Stop()
	srv2.Stop()
}
