# Distributed Display Protocol (DDP)

This package implements the DDP server for the WLED simulator based on the [DDP specification](http://www.3waylabs.com/ddp/).

## Overview

DDP was designed for sending real-time data to distributed lighting displays where synchronization may be important. It provides a more efficient alternative to DMX-over-Ethernet protocols like Art-Net and E1.31.

## Packet Format

All packets sent and received have a 10 or 14 byte header followed by optional data.

### Header Structure

```
byte  0:    flags: V V x T S R Q P 
            V V:    2-bits for protocol version number, this document specifies version 1 (01).
            x:      reserved
            T:      timecode field added to end of header
                    if T & P are set, Push at specified time
            S:      Storage.  If set, data comes from Storage, not data-field.
            R:      Reply flag, marks reply to Query packet.
                    always set when any packet is sent by a Display.
                    if Reply, Q flag is ignored.
            Q:      Query flag, requests len data from ID at offset (no data sent)
                    if clear, is a Write buffer packet
            P:      Push flag, for display synchronization, or marks last packet of Reply

byte  1:    x x x x n n n n
            x: reserved for future use (set to zero)
            nnnn: sequence number from 1-15, or zero if not used
              the sequence number should be incremented with each new packet sent.
              a sender can send duplicate packets with the same sequence number and DDP header for redundancy.
              a receiver can ignore duplicates received back-to-back.
              the sequence number is ignored if zero.

byte  2:    data type
            set to zero if not used or undefined, otherwise:
            bits: C R TTT SSS
             C is 0 for standard types or 1 for Customer defined
             R is reserved and should be 0.
             TTT is data type 
              000 = undefined
              001 = RGB
              010 = HSL
              011 = RGBW
              100 = grayscale
             SSS is size in bits per pixel element (like just R or G or B data)
              0=undefined, 1=1, 2=4, 3=8, 4=16, 5=24, 6=32

byte  3:    Source or Destination ID
            0 = reserved
            1 = default output device
            2=249 custom IDs, (possibly defined via JSON config)
            246 = JSON control (read/write)
            250 = JSON config  (read/write)
            251 = JSON status  (read only)
            254 = DMX transit
            255 = all devices

byte  4-7:  data offset in bytes
            32-bit number, MSB first

byte  8-9:  data length in bytes (size of data field when writing)
            16-bit number, MSB first
            for Queries, this specifies size of data to read, no data field follows header.

if T flag, header extended 4 bytes for timecode field (not counted in data length)
byte 10-13: timecode

byte 10 or 14: start of data
```

### Protocol Constants

- **Port**: 4048 (UDP)
- **Version**: 1
- **Maximum Data Length**: Typically ~1440 bytes to fit in ethernet packet

### Data Types

| TTT | Type      | Description          |
|-----|-----------|----------------------|
| 000 | Undefined | No specific type     |
| 001 | RGB       | Red, Green, Blue     |
| 010 | HSL       | Hue, Saturation, Lightness |
| 011 | RGBW      | Red, Green, Blue, White |
| 100 | Grayscale | Single intensity value |

### Bits Per Pixel Element

| SSS | Bits | Description     |
|-----|------|-----------------|
| 000 | 0    | Undefined       |
| 001 | 1    | 1 bit per element |
| 010 | 4    | 4 bits per element |
| 011 | 8    | 8 bits per element |
| 100 | 16   | 16 bits per element |
| 101 | 24   | 24 bits per element |
| 110 | 32   | 32 bits per element |

### Device IDs

| ID  | Purpose           | Description                    |
|-----|-------------------|--------------------------------|
| 0   | Reserved          | Not used                       |
| 1   | Default Output    | Standard display device        |
| 2-249 | Custom IDs      | User-defined devices           |
| 246 | JSON Control      | Read/write control commands    |
| 250 | JSON Config       | Read/write configuration       |
| 251 | JSON Status       | Read-only status information   |
| 254 | DMX Transit       | DMX compatibility mode         |
| 255 | All Devices       | Broadcast to all devices       |

## Implementation Notes

The WLED simulator implements:
- Version 1 of the DDP protocol
- RGB data type (001) with 8 bits per element (011)
- Default output device (ID=1)
- Packet validation with verbose error logging
- Sequence number tracking for duplicate detection

## References

- [DDP Specification](http://www.3waylabs.com/ddp/) - Official protocol documentation
- [3WayLabs](http://www.3waylabs.com/) - Protocol author's website 