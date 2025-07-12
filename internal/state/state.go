package state

import (
    "fmt"
    "image/color"
    "sync"
)

type LEDState struct {
    mu         sync.RWMutex
    power      bool
    brightness int // 0-255
    leds       []color.RGBA
}

// NewLEDState constructs a LEDState with n LEDs initialized to hex colour
func NewLEDState(n int, hex string) *LEDState {
    leds := make([]color.RGBA, n)
    c := parseHex(hex)
    for i := range leds {
        leds[i] = c
    }
    return &LEDState{power: true, brightness: 255, leds: leds}
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
