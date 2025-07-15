package state

import (
	"testing"
	"time"
)

func TestLiveFunctionality(t *testing.T) {
	state := NewLEDState(10, "#000000")

	// Initially, live should be false
	if state.IsLive() {
		t.Error("Expected IsLive() to be false initially")
	}

	// After calling SetLive(), it should be true
	state.SetLive()
	if !state.IsLive() {
		t.Error("Expected IsLive() to be true after SetLive()")
	}

	// Test custom timeout
	state.SetLiveTimeout(100 * time.Millisecond)
	state.SetLive()

	// Should still be live immediately
	if !state.IsLive() {
		t.Error("Expected IsLive() to be true immediately after SetLive()")
	}

	// Wait for timeout to expire
	time.Sleep(150 * time.Millisecond)

	// Should no longer be live
	if state.IsLive() {
		t.Error("Expected IsLive() to be false after timeout")
	}
}

func TestLiveTimeout(t *testing.T) {
	state := NewLEDState(10, "#000000")

	// Test that default timeout is reasonable (should be 5 seconds)
	state.SetLive()
	if !state.IsLive() {
		t.Error("Expected IsLive() to be true after SetLive()")
	}

	// Should still be live after 1 second
	time.Sleep(1 * time.Second)
	if !state.IsLive() {
		t.Error("Expected IsLive() to still be true after 1 second")
	}

	// Change timeout to very short duration
	state.SetLiveTimeout(50 * time.Millisecond)
	state.SetLive()

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)
	if state.IsLive() {
		t.Error("Expected IsLive() to be false after short timeout")
	}
}
