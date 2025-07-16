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
	port         int
	state        *state.LEDState
	conn         *net.UDPConn
	ctx          context.Context
	cancel       context.CancelFunc
	lastSequence uint8
	verbose      bool
}

func NewServer(port int, s *state.LEDState) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		port:    port,
		state:   s,
		ctx:     ctx,
		cancel:  cancel,
		verbose: false, // Disable verbose logging by default
	}
}

// processPacket processes a validated DDP packet
func (s *Server) processPacket(header *DDPHeader, data []byte) error {
	headerSize := MinHeaderSize
	if header.HasTimecode {
		headerSize = MaxHeaderSize
	}

	payload := data[headerSize : headerSize+int(header.DataLength)]

	if s.verbose {
		typeStr := "undefined"
		switch header.DataType.Type {
		case TypeRGB:
			typeStr = "RGB"
		case TypeHSL:
			typeStr = "HSL"
		case TypeRGBW:
			typeStr = "RGBW"
		case TypeGrayscale:
			typeStr = "Grayscale"
		}

		customStr := ""
		if header.DataType.IsCustom {
			customStr = " (custom)"
		}

		log.Printf("[DDP] Processing packet: version=%d, seq=%d, type=%s%s (%d bits/element), device=%d, offset=%d, length=%d",
			header.Version, header.Sequence, typeStr, customStr, header.DataType.BitsPerElement,
			header.DeviceID, header.DataOffset, header.DataLength)
	}

	// Handle query packets
	if header.Query {
		if s.verbose {
			log.Printf("[DDP] Query packet received - not implemented")
		}
		return nil
	}

	// Mark that we're receiving live DDP data
	s.state.SetLive()

	// Process RGB data
	leds := s.state.LEDs()
	maxIndex := len(leds)
	startIndex := int(header.DataOffset / 3) // Assuming 3 bytes per LED (RGB)

	pixelCount := 0
	for i := 0; i+2 < len(payload); i += 3 {
		ledIndex := startIndex + (i / 3)
		if ledIndex >= maxIndex {
			break
		}
		s.state.SetLED(ledIndex, color.RGBA{
			R: payload[i],
			G: payload[i+1],
			B: payload[i+2],
			A: 255,
		})
		pixelCount++
	}

	if s.verbose {
		log.Printf("[DDP] Updated %d LEDs starting at index %d", pixelCount, startIndex)
	}

	return nil
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
				n, remoteAddr, err := conn.ReadFromUDP(buf)
				if err != nil {
					if s.ctx.Err() != nil {
						return // Normal shutdown
					}
					log.Printf("[DDP] UDP read error: %v", err)
					continue
				}

				// Parse and validate header
				header, err := ParseHeader(buf[:n])
				if err != nil {
					s.state.ReportActivity(state.ActivityDDP, false) // Report failed DDP activity
					if s.verbose {
						log.Printf("[DDP] Invalid packet from %s: %v", remoteAddr, err)
					}
					continue
				}

				// Additional validation
				if err := ValidateHeader(header, &s.lastSequence); err != nil {
					s.state.ReportActivity(state.ActivityDDP, false) // Report failed DDP activity
					if s.verbose {
						log.Printf("[DDP] Packet validation failed from %s: %v", remoteAddr, err)
					}
					continue
				}

				// Process the packet
				if err := s.processPacket(header, buf[:n]); err != nil {
					s.state.ReportActivity(state.ActivityDDP, false) // Report failed DDP activity
					if s.verbose {
						log.Printf("[DDP] Packet processing failed from %s: %v", remoteAddr, err)
					}
					continue
				}

				s.state.ReportActivity(state.ActivityDDP, true) // Report successful DDP activity
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

// SetVerbose enables or disables verbose logging
func (s *Server) SetVerbose(verbose bool) {
	s.verbose = verbose
}
