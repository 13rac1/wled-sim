package ddp

import (
	"testing"
)

func TestParseDataType(t *testing.T) {
	tests := []struct {
		name           string
		input          uint8
		expectedCustom bool
		expectedType   uint8
		expectedSize   uint8
		expectedBits   int
	}{
		{
			name:           "RGB 8-bit standard",
			input:          0x0B, // 00001011: C=0, R=0, TTT=001, SSS=011
			expectedCustom: false,
			expectedType:   TypeRGB,
			expectedSize:   Size8Bit,
			expectedBits:   8,
		},
		{
			name:           "Undefined type",
			input:          0x00, // 00000000: all bits zero
			expectedCustom: false,
			expectedType:   TypeUndefined,
			expectedSize:   SizeUndefined,
			expectedBits:   0,
		},
		{
			name:           "RGB 16-bit standard",
			input:          0x0C, // 00001100: C=0, R=0, TTT=001, SSS=100
			expectedCustom: false,
			expectedType:   TypeRGB,
			expectedSize:   Size16Bit,
			expectedBits:   16,
		},
		{
			name:           "HSL 8-bit standard",
			input:          0x13, // 00010011: C=0, R=0, TTT=010, SSS=011
			expectedCustom: false,
			expectedType:   TypeHSL,
			expectedSize:   Size8Bit,
			expectedBits:   8,
		},
		{
			name:           "RGBW 8-bit standard",
			input:          0x1B, // 00011011: C=0, R=0, TTT=011, SSS=011
			expectedCustom: false,
			expectedType:   TypeRGBW,
			expectedSize:   Size8Bit,
			expectedBits:   8,
		},
		{
			name:           "Grayscale 8-bit standard",
			input:          0x23, // 00100011: C=0, R=0, TTT=100, SSS=011
			expectedCustom: false,
			expectedType:   TypeGrayscale,
			expectedSize:   Size8Bit,
			expectedBits:   8,
		},
		{
			name:           "Custom RGB 8-bit",
			input:          0x8B, // 10001011: C=1, R=0, TTT=001, SSS=011
			expectedCustom: true,
			expectedType:   TypeRGB,
			expectedSize:   Size8Bit,
			expectedBits:   8,
		},
		{
			name:           "RGB 1-bit",
			input:          0x09, // 00001001: C=0, R=0, TTT=001, SSS=001
			expectedCustom: false,
			expectedType:   TypeRGB,
			expectedSize:   Size1Bit,
			expectedBits:   1,
		},
		{
			name:           "RGB 4-bit",
			input:          0x0A, // 00001010: C=0, R=0, TTT=001, SSS=010
			expectedCustom: false,
			expectedType:   TypeRGB,
			expectedSize:   Size4Bit,
			expectedBits:   4,
		},
		{
			name:           "RGB 24-bit",
			input:          0x0D, // 00001101: C=0, R=0, TTT=001, SSS=101
			expectedCustom: false,
			expectedType:   TypeRGB,
			expectedSize:   Size24Bit,
			expectedBits:   24,
		},
		{
			name:           "RGB 32-bit",
			input:          0x0E, // 00001110: C=0, R=0, TTT=001, SSS=110
			expectedCustom: false,
			expectedType:   TypeRGB,
			expectedSize:   Size32Bit,
			expectedBits:   32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDataType(tt.input)

			if result.IsCustom != tt.expectedCustom {
				t.Errorf("IsCustom = %v, want %v", result.IsCustom, tt.expectedCustom)
			}
			if result.Type != tt.expectedType {
				t.Errorf("Type = %d, want %d", result.Type, tt.expectedType)
			}
			if result.Size != tt.expectedSize {
				t.Errorf("Size = %d, want %d", result.Size, tt.expectedSize)
			}
			if result.BitsPerElement != tt.expectedBits {
				t.Errorf("BitsPerElement = %d, want %d", result.BitsPerElement, tt.expectedBits)
			}
		})
	}
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name          string
		packet        []byte
		expectedError string
		checkHeader   func(*testing.T, *DDPHeader)
	}{
		{
			name:          "packet too short",
			packet:        []byte{0x41, 0x00, 0x0B, 0x01, 0x00},
			expectedError: "packet too short",
		},
		{
			name:   "valid RGB packet with push",
			packet: []byte{0x41, 0x05, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x00, 0x00, 0x00, 0xFF, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if h.Version != 1 {
					t.Errorf("Version = %d, want 1", h.Version)
				}
				if !h.Push {
					t.Errorf("Push = false, want true")
				}
				if h.Sequence != 5 {
					t.Errorf("Sequence = %d, want 5", h.Sequence)
				}
				if h.DataType.Type != TypeRGB {
					t.Errorf("DataType.Type = %d, want %d", h.DataType.Type, TypeRGB)
				}
				if h.DataType.Size != Size8Bit {
					t.Errorf("DataType.Size = %d, want %d", h.DataType.Size, Size8Bit)
				}
				if h.DeviceID != DeviceIDDefault {
					t.Errorf("DeviceID = %d, want %d", h.DeviceID, DeviceIDDefault)
				}
				if h.DataLength != 6 {
					t.Errorf("DataLength = %d, want 6", h.DataLength)
				}
			},
		},
		{
			name:   "valid packet without push",
			packet: []byte{0x40, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if h.Push {
					t.Errorf("Push = true, want false")
				}
				if h.Version != 1 {
					t.Errorf("Version = %d, want 1", h.Version)
				}
			},
		},
		{
			name:   "packet with timecode",
			packet: []byte{0x51, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x12, 0x34, 0x56, 0x78, 0xFF, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if !h.HasTimecode {
					t.Errorf("HasTimecode = false, want true")
				}
				if h.Timecode != 0x12345678 {
					t.Errorf("Timecode = 0x%08X, want 0x12345678", h.Timecode)
				}
			},
		},
		{
			name:          "unsupported version",
			packet:        []byte{0x81, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x00, 0x00},
			expectedError: "unsupported DDP version",
		},
		{
			name:          "reserved bit set in data type",
			packet:        []byte{0x41, 0x00, 0x4B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x00, 0x00},
			expectedError: "data type reserved bit is set",
		},
		{
			name:          "timecode flag set but packet too short",
			packet:        []byte{0x51, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
			expectedError: "packet with timecode flag too short",
		},
		{
			name:          "data length mismatch",
			packet:        []byte{0x41, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0xFF, 0x00, 0x00},
			expectedError: "packet data too short",
		},
		{
			name:   "query packet",
			packet: []byte{0x42, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if !h.Query {
					t.Errorf("Query = false, want true")
				}
			},
		},
		{
			name:   "storage flag set",
			packet: []byte{0x49, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if !h.Storage {
					t.Errorf("Storage = false, want true")
				}
			},
		},
		{
			name:   "reply flag set",
			packet: []byte{0x45, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if !h.Reply {
					t.Errorf("Reply = false, want true")
				}
			},
		},
		{
			name:   "broadcast device ID",
			packet: []byte{0x41, 0x00, 0x0B, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xFF, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if h.DeviceID != DeviceIDAllDevices {
					t.Errorf("DeviceID = %d, want %d", h.DeviceID, DeviceIDAllDevices)
				}
			},
		},
		{
			name:   "data offset test",
			packet: []byte{0x41, 0x00, 0x0B, 0x01, 0x00, 0x00, 0x00, 0x09, 0x00, 0x03, 0xFF, 0x00, 0x00},
			checkHeader: func(t *testing.T, h *DDPHeader) {
				if h.DataOffset != 9 {
					t.Errorf("DataOffset = %d, want 9", h.DataOffset)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := ParseHeader(tt.packet)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
					return
				}
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if header == nil {
				t.Error("Header is nil")
				return
			}

			if tt.checkHeader != nil {
				tt.checkHeader(t, header)
			}
		})
	}
}

func TestValidateHeader(t *testing.T) {
	tests := []struct {
		name          string
		header        *DDPHeader
		lastSequence  uint8
		expectedError string
	}{
		{
			name: "valid RGB header",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeRGB,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
				Sequence: 1,
			},
		},
		{
			name: "valid undefined type header",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeUndefined,
					Size:           SizeUndefined,
					BitsPerElement: 0,
				},
			},
		},
		{
			name: "valid broadcast device ID",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDAllDevices,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeRGB,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
			},
		},
		{
			name: "unsupported device ID",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDJSONControl,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeRGB,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
			},
			expectedError: "unsupported device ID",
		},
		{
			name: "custom data type not supported",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       true,
					Type:           TypeRGB,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
			},
			expectedError: "custom data types not supported",
		},
		{
			name: "HSL data type not supported",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeHSL,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
			},
			expectedError: "unsupported data type: HSL",
		},
		{
			name: "RGBW data type not supported",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeRGBW,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
			},
			expectedError: "unsupported data type: RGBW",
		},
		{
			name: "Grayscale data type not supported",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeGrayscale,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
			},
			expectedError: "unsupported data type: Grayscale",
		},
		{
			name: "RGB with wrong bit size",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeRGB,
					Size:           Size16Bit,
					BitsPerElement: 16,
				},
			},
			expectedError: "unsupported RGB size: 16 bits per element",
		},
		{
			name: "duplicate sequence number",
			header: &DDPHeader{
				Version:  1,
				DeviceID: DeviceIDDefault,
				DataType: DataTypeInfo{
					IsCustom:       false,
					Type:           TypeRGB,
					Size:           Size8Bit,
					BitsPerElement: 8,
				},
				Sequence: 5,
			},
			lastSequence:  5,
			expectedError: "duplicate sequence number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up lastSequence for duplicate sequence test
			lastSeq := tt.lastSequence

			err := ValidateHeader(tt.header, &lastSeq)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
					return
				}
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
