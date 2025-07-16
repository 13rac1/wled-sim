package state

import (
	"fmt"
	"image/color"
	"sync"
	"time"
)

type ActivityType int

const (
	ActivityJSON ActivityType = iota
	ActivityDDP
)

type ActivityEvent struct {
	Type      ActivityType
	Success   bool
	Timestamp time.Time
}

type LEDState struct {
	mu              sync.RWMutex
	power           bool
	brightness      int // 0-255
	leds            []color.RGBA
	lastLiveTime    time.Time          // Timestamp of last DDP packet received
	liveTimeout     time.Duration      // How long to consider live after last packet
	activityChannel chan ActivityEvent // Channel for activity events
}

// NewLEDState constructs a LEDState with n LEDs initialized to hex colour
func NewLEDState(n int, hex string) *LEDState {
	leds := make([]color.RGBA, n)
	c := parseHex(hex)
	for i := range leds {
		leds[i] = c
	}
	return &LEDState{
		power:           true,
		brightness:      255,
		leds:            leds,
		liveTimeout:     5 * time.Second,               // Consider live for 5 seconds after last packet
		activityChannel: make(chan ActivityEvent, 100), // Buffered channel for activity events
	}
}

// parseHex converts "#RRGGBB" to color.RGBA
func parseHex(h string) color.RGBA {
	var r, g, b uint8
	if len(h) == 7 && h[0] == '#' {
		_, _ = fmt.Sscanf(h[1:], "%02x%02x%02x", &r, &g, &b)
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// SetPower sets the on/off state
func (s *LEDState) SetPower(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.power = on
}

func (s *LEDState) Power() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.power
}

func (s *LEDState) SetBrightness(b int) {
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.brightness = b
}

func (s *LEDState) Brightness() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brightness
}

func (s *LEDState) SetLED(i int, c color.RGBA) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i >= 0 && i < len(s.leds) {
		s.leds[i] = c
	}
}

func (s *LEDState) LEDs() []color.RGBA {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]color.RGBA, len(s.leds))
	copy(out, s.leds)
	return out
}

// SetLive marks that DDP data is currently being received
func (s *LEDState) SetLive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastLiveTime = time.Now()
}

// IsLive returns true if DDP data has been received recently
func (s *LEDState) IsLive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastLiveTime.IsZero() {
		return false
	}
	return time.Since(s.lastLiveTime) <= s.liveTimeout
}

// SetLiveTimeout sets the duration for which the device should be considered live after receiving data
func (s *LEDState) SetLiveTimeout(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.liveTimeout = timeout
}

// ReportActivity reports an activity event (non-blocking)
func (s *LEDState) ReportActivity(activityType ActivityType, success bool) {
	event := ActivityEvent{
		Type:      activityType,
		Success:   success,
		Timestamp: time.Now(),
	}

	// Non-blocking send to avoid deadlocks
	select {
	case s.activityChannel <- event:
		// Event sent successfully
	default:
		// Channel is full, drop the event
	}
}

// ActivityChannel returns the activity event channel for consumers
func (s *LEDState) ActivityChannel() <-chan ActivityEvent {
	return s.activityChannel
}
