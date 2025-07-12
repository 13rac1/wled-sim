package ddp

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"net"

	"wled-simulator/internal/state"
)

type Server struct {
	port   int
	state  *state.LEDState
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
}

func NewServer(port int, s *state.LEDState) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		port:   port,
		state:  s,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins listening for DDP packets
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	go func() {
		defer conn.Close()
		buf := make([]byte, 1500)
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				n, _, err := conn.ReadFromUDP(buf)
				if err != nil {
					if s.ctx.Err() != nil {
						return // Normal shutdown
					}
					log.Println("UDP read error:", err)
					continue
				}
				if n <= 10 {
					continue // need header + payload
				}
				payload := buf[10:n]
				leds := s.state.LEDs()
				maxIndex := len(leds)
				for i := 0; i+2 < len(payload); i += 3 {
					idx := i / 3
					if idx >= maxIndex {
						break
					}
					s.state.SetLED(idx, color.RGBA{R: payload[i], G: payload[i+1], B: payload[i+2], A: 255})
				}
			}
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.cancel()
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
