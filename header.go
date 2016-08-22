package lsep

import (
	"encoding/binary"
	"fmt"
)

type Frame struct {
	HeaderByte  byte
	Header      Header
	LengthBytes []byte
	Length      uint64
}

type Header struct {
	Version           HeaderVersion
	Options           HeaderOptions
	FrameHeaderLength FrameHeaderLength
}

// LSEP Versions are all mutually exclusive, ie, a Version 2 client is not
// compatible with a Version 1 client or Version 3 client.
type HeaderVersion uint8

const (
	// Only current implemented version
	HeaderVersion1 = 0
)

// The LSEP Option indicates how data was compressed on the wire.
// Atm only Raw is supported, the other three options may change
type HeaderOptions uint8

const (
	// Send data raw over the socket
	HeaderOptionRaw = 0
)

// The Length of the actual Length field in bytes
type FrameHeaderLength uint8

func (h *Header) Encode() byte {
	if h.FrameHeaderLength == 0 {
		h.FrameHeaderLength = 1 // Minimum Number is 1
	}

	var raw = byte(0)

	raw = byte(h.Version) & 0x7
	raw = raw<<2 | (byte(h.Options) & 0x3)
	raw = raw<<3 | (byte(h.FrameHeaderLength-1) & 0x07)

	return raw
}

func (h *Header) Decode(raw byte) {

	h.Version = HeaderVersion(((raw >> 5) & 0x7))
	h.Options = HeaderOptions(((raw >> 3) & 0x3))
	h.FrameHeaderLength = FrameHeaderLength((raw & 0x7) + 1)

}

func calcByteness(integer uint64) uint8 {
	if integer == 0 {
		return 1
	}

	var bytes uint8 = 0

	for integer != 0 {
		integer >>= 8
		bytes++
	}

	return bytes
}

func CreateFrame(ver HeaderVersion, opts HeaderOptions, dataLength uint64) Frame {
	f := Frame{}

	f.LengthBytes = make([]byte, 8)
	binary.BigEndian.PutUint64(f.LengthBytes, dataLength)
	f.LengthBytes = f.LengthBytes[8-calcByteness(dataLength):]

	f.Length = dataLength

	f.Header.FrameHeaderLength = FrameHeaderLength(calcByteness(dataLength))
	f.Header.Version = ver
	f.Header.Options = opts

	f.HeaderByte = f.Header.Encode()

	return f
}

func (f *Frame) GetFrame() []byte {
	b := make([]byte, 1+len(f.LengthBytes))
	b[0] = f.HeaderByte
	if copy(b[1:], f.LengthBytes) != int(f.Header.FrameHeaderLength) {
		panic(fmt.Sprintf("Frame Copy fail: %X should be %d bytes long", f.LengthBytes, int(f.Header.FrameHeaderLength)))
	}
	return b
}

// ParseFrame attempts to interprete the data array as a frame header
// It must be true that len(data)==2, otherwise the parsing fails
// If the data array does not contain enough bytes to parse as a full header
// parsing fails and an error is returned. The frame is nil but the unused byte array
// is set to the number of bytes needed to parse the frame.
//
// If parsing was good, it returns a frame and a slice containing all
// unused bytes, so they can instantly be read.
func ParseFrame(data []byte) (*Frame, []byte, error) {
	if len(data) < 2 {
		return nil, nil, fmt.Errorf("Insufficient data for frame")
	}
	h := &Header{}
	h.Decode(data[0])

	if h.Version != HeaderVersion1 {
		return nil, nil, fmt.Errorf("Invalid Header Version")
	}

	if h.Options != HeaderOptionRaw {
		return nil, nil, fmt.Errorf("Invalid Options Value")
	}

	if int(h.FrameHeaderLength)+1 > len(data) {
		return nil, make([]byte, int(h.FrameHeaderLength)+1-len(data)), fmt.Errorf("Missing Frame Data: %d more bytes needed", int(h.FrameHeaderLength)+1-len(data))
	}

	lenDat := data[1 : 1+int(h.FrameHeaderLength)]

	for len(lenDat) < 8 {
		lenDat = append([]byte{byte(0x00)}, lenDat...)
	}

	length := binary.BigEndian.Uint64(lenDat)

	f := CreateFrame(h.Version, h.Options, length)

	return &f, data[h.FrameHeaderLength+1:], nil
}
