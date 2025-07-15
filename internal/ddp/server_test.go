package ddp

import (
	"testing"

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
