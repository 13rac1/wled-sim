package ddp

import (
	"context"
	"encoding/binary"
	"fmt"
	"image/color"
	"log"
	"net"

	"wled-simulator/internal/state"
)

// DDP Protocol constants
const (
	DDPVersion    = 1
	MinHeaderSize = 10
	MaxHeaderSize = 14
)

// DDP Flags (byte 0)
const (
	FlagVersionMask  = 0xC0 // VV bits (7-6)
	FlagVersionShift = 6
	FlagTimecode     = 0x10 // T bit (4)
	FlagStorage      = 0x08 // S bit (3)
	FlagReply        = 0x04 // R bit (2)
	FlagQuery        = 0x02 // Q bit (1)
	FlagPush         = 0x01 // P bit (0)
)

// DDP Data Types (byte 2) - bit fields: C R TTT SSS
const (
	DataTypeCustomMask   = 0x80 // C bit (7) - 0=standard, 1=custom
	DataTypeReservedMask = 0x40 // R bit (6) - reserved, should be 0
	DataTypeTypeMask     = 0x38 // TTT bits (5-3) - data type
	DataTypeSizeMask     = 0x07 // SSS bits (2-0) - size in bits per element
)

// Data type values (TTT bits)
const (
	TypeUndefined = 0 // 000
	TypeRGB       = 1 // 001
	TypeHSL       = 2 // 010
	TypeRGBW      = 3 // 011
	TypeGrayscale = 4 // 100
)

// Size values (SSS bits) - bits per pixel element
const (
	SizeUndefined = 0 // 0 bits
	Size1Bit      = 1 // 1 bit
	Size4Bit      = 2 // 4 bits
	Size8Bit      = 3 // 8 bits
	Size16Bit     = 4 // 16 bits
	Size24Bit     = 5 // 24 bits
	Size32Bit     = 6 // 32 bits
)

// DataTypeInfo represents parsed data type information
type DataTypeInfo struct {
	IsCustom       bool
	Type           uint8
	Size           uint8
	BitsPerElement int
}

// parseDataType parses the data type byte into its component fields
func parseDataType(dataTypeByte uint8) DataTypeInfo {
	info := DataTypeInfo{
		IsCustom: (dataTypeByte & DataTypeCustomMask) != 0,
		Type:     (dataTypeByte & DataTypeTypeMask) >> 3,
		Size:     dataTypeByte & DataTypeSizeMask,
	}

	// Convert size enum to actual bits
	switch info.Size {
	case Size1Bit:
		info.BitsPerElement = 1
	case Size4Bit:
		info.BitsPerElement = 4
	case Size8Bit:
		info.BitsPerElement = 8
	case Size16Bit:
		info.BitsPerElement = 16
	case Size24Bit:
		info.BitsPerElement = 24
	case Size32Bit:
		info.BitsPerElement = 32
	default:
		info.BitsPerElement = 0 // Undefined
	}

	return info
}

// DDP Device IDs (byte 3)
type DeviceID uint8

const (
	DeviceIDReserved    DeviceID = 0   // Reserved
	DeviceIDDefault     DeviceID = 1   // Default output device
	DeviceIDJSONControl DeviceID = 246 // JSON control
	DeviceIDJSONConfig  DeviceID = 250 // JSON config
	DeviceIDJSONStatus  DeviceID = 251 // JSON status
	DeviceIDDMXTransit  DeviceID = 254 // DMX transit
	DeviceIDAllDevices  DeviceID = 255 // All devices
)

// DDPHeader represents a parsed DDP packet header
type DDPHeader struct {
	Version     uint8
	HasTimecode bool
	Storage     bool
	Reply       bool
	Query       bool
	Push        bool
	Sequence    uint8
	DataType    DataTypeInfo
	DeviceID    DeviceID
	DataOffset  uint32
	DataLength  uint16
	Timecode    uint32 // Only present if HasTimecode is true
}

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
		verbose: true, // Enable verbose logging by default
	}
}

// parseHeader parses and validates a DDP packet header
func (s *Server) parseHeader(data []byte) (*DDPHeader, error) {
	if len(data) < MinHeaderSize {
		return nil, fmt.Errorf("packet too short: got %d bytes, need at least %d", len(data), MinHeaderSize)
	}

	header := &DDPHeader{}

	// Parse byte 0 (flags)
	flags := data[0]
	header.Version = (flags & FlagVersionMask) >> FlagVersionShift
	header.HasTimecode = (flags & FlagTimecode) != 0
	header.Storage = (flags & FlagStorage) != 0
	header.Reply = (flags & FlagReply) != 0
	header.Query = (flags & FlagQuery) != 0
	header.Push = (flags & FlagPush) != 0

	// Validate version
	if header.Version != DDPVersion {
		return nil, fmt.Errorf("unsupported DDP version: got %d, expected %d", header.Version, DDPVersion)
	}

	// Parse byte 1 (sequence)
	header.Sequence = data[1] & 0x0F // Lower 4 bits

	// Parse byte 2 (data type)
	dataTypeByte := data[2]
	header.DataType = parseDataType(dataTypeByte)

	// Check reserved bit in data type (should be 0)
	if dataTypeByte&DataTypeReservedMask != 0 {
		return nil, fmt.Errorf("data type reserved bit is set (should be 0)")
	}

	// Parse byte 3 (device ID)
	header.DeviceID = DeviceID(data[3])

	// Parse bytes 4-7 (data offset, big-endian)
	header.DataOffset = binary.BigEndian.Uint32(data[4:8])

	// Parse bytes 8-9 (data length, big-endian)
	header.DataLength = binary.BigEndian.Uint16(data[8:10])

	// Parse timecode if present
	expectedHeaderSize := MinHeaderSize
	if header.HasTimecode {
		expectedHeaderSize = MaxHeaderSize
		if len(data) < expectedHeaderSize {
			return nil, fmt.Errorf("packet with timecode flag too short: got %d bytes, need %d", len(data), expectedHeaderSize)
		}
		header.Timecode = binary.BigEndian.Uint32(data[10:14])
	}

	// Validate packet size
	expectedPacketSize := expectedHeaderSize + int(header.DataLength)
	if len(data) < expectedPacketSize {
		return nil, fmt.Errorf("packet data too short: got %d bytes, expected %d (header: %d, data: %d)",
			len(data), expectedPacketSize, expectedHeaderSize, header.DataLength)
	}

	return header, nil
}

// validateHeader performs additional validation on the parsed header
func (s *Server) validateHeader(header *DDPHeader) error {
	// Check device ID
	if header.DeviceID != DeviceIDDefault && header.DeviceID != DeviceIDAllDevices {
		return fmt.Errorf("unsupported device ID: %d (expected %d or %d)",
			header.DeviceID, DeviceIDDefault, DeviceIDAllDevices)
	}

	// Check if custom data types are supported (we don't support them)
	if header.DataType.IsCustom {
		return fmt.Errorf("custom data types not supported (C bit set)")
	}

	// Check data type - we only support RGB and undefined
	if header.DataType.Type != TypeRGB && header.DataType.Type != TypeUndefined {
		typeName := "unknown"
		switch header.DataType.Type {
		case TypeHSL:
			typeName = "HSL"
		case TypeRGBW:
			typeName = "RGBW"
		case TypeGrayscale:
			typeName = "Grayscale"
		}
		return fmt.Errorf("unsupported data type: %s (%d), only RGB (%d) and undefined (%d) supported",
			typeName, header.DataType.Type, TypeRGB, TypeUndefined)
	}

	// For RGB data, check that we have 8 bits per element
	if header.DataType.Type == TypeRGB {
		if header.DataType.Size != Size8Bit {
			return fmt.Errorf("unsupported RGB size: %d bits per element (expected 8)",
				header.DataType.BitsPerElement)
		}
	}

	// Check sequence number for duplicates (if not zero)
	if header.Sequence != 0 {
		if header.Sequence == s.lastSequence {
			return fmt.Errorf("duplicate sequence number: %d", header.Sequence)
		}
		s.lastSequence = header.Sequence
	}

	return nil
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
				header, err := s.parseHeader(buf[:n])
				if err != nil {
					if s.verbose {
						log.Printf("[DDP] Invalid packet from %s: %v", remoteAddr, err)
					}
					continue
				}

				// Additional validation
				if err := s.validateHeader(header); err != nil {
					if s.verbose {
						log.Printf("[DDP] Packet validation failed from %s: %v", remoteAddr, err)
					}
					continue
				}

				// Process the packet
				if err := s.processPacket(header, buf[:n]); err != nil {
					if s.verbose {
						log.Printf("[DDP] Packet processing failed from %s: %v", remoteAddr, err)
					}
					continue
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

// SetVerbose enables or disables verbose logging
func (s *Server) SetVerbose(verbose bool) {
	s.verbose = verbose
}
