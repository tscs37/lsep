package lsep

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteness(t *testing.T) {
	assert := assert.New(t)

	assert.EqualValues(1, calcByteness(0), "0 length = 1 byte")

	assert.EqualValues(1, calcByteness(1), "1 length = 1 byte")

	for n := uint(0); n < 8; n++ {
		assert.EqualValues(1, calcByteness(1<<n), "%d length = %d bytes", 1<<n, 1)
	}

	assert.EqualValues(1, calcByteness(255), "255 length = 1 byte")
	assert.EqualValues(2, calcByteness(256), "1 length = 1 byte")

	for n := uint(8); n < 64; n++ {
		assert.EqualValues(n/8+1, calcByteness(1<<n), "%d length = %d bytes", 1<<n, n/8)
	}

	assert.EqualValues(8, calcByteness((1<<64)-1), "(1<<64)-1 length = 8 byte")
}

func TestHeader_Encode(t *testing.T) {
	assert := assert.New(t)

	var h = &Header{
		Version:           HeaderVersion(0x0),
		Options:           HeaderOptions(0x0),
		FrameHeaderLength: 0x8, // Subtracts 1 internally
	}

	res := h.Encode()

	assert.EqualValues(res, 7, "Encoding Frame Header Length")

	h2 := &Header{}

	h2.Decode(res)

	assert.EqualValues(h2.FrameHeaderLength, 8, "Decode FHL")

	h = &Header{
		Version:           HeaderVersion(0xF),
		Options:           HeaderOptions(0x0),
		FrameHeaderLength: 0x0,
	}

	res = h.Encode()

	assert.EqualValues(res, 0xE0, "Encode Header", t)

	h2.Decode(res)

	assert.EqualValues(byte(h2.Version), 0x7, "Decode Header")

	h = &Header{
		Version:           HeaderVersion(0x0),
		Options:           HeaderOptions(0xF),
		FrameHeaderLength: 0x0,
	}

	res = h.Encode()

	assert.EqualValues(res, 0x3<<3, "Encode Options")

	h2.Decode(res)

	assert.EqualValues(byte(h2.Options), 0x3, "Decode Options")
}

func TestCreateFrame(t *testing.T) {
	for i := uint(1); i < 9; i++ {
		f := CreateFrame(HeaderVersion1, HeaderOptionRaw, 1<<(8*i)-1)

		if uint(len(f.GetFrame())) != i+1 {
			t.Logf("Encoding %d long message, wrong length: %d, expected %d", uint(1<<(8*i)-1), len(f.GetFrame()), i+1)
			t.Fail()
		}

		if uint(1<<(8*i)) > 0 {

			f = CreateFrame(HeaderVersion1, HeaderOptionRaw, 1<<(8*i))

			if uint(len(f.GetFrame())) != i+2 {
				t.Logf("Encoding %d long message, wrong length: %d, expected %d", uint(1<<(8*i)), len(f.GetFrame()), i+1)
				t.Fail()
			}

		}

	}
}

func TestFrame_GetFrame(t *testing.T) {
	assert := assert.New(t)

	f := CreateFrame(HeaderVersion1, HeaderOptionRaw, 17000)

	assert.Equal(f.GetFrame(), []byte{0x01, 0x42, 0x68}, "Frame is properly Encoded")

	f2, unused, err := ParseFrame(f.GetFrame())

	assert.Nil(err, "Cannot fail parsing of correct frame")

	assert.NotZero(unused, "There is no leftover data")

	assert.Equal(f2.Length, f.Length, "Frame Decode & Encode must preserve Frame Length")

	assert.Equal(f2.HeaderByte, f.HeaderByte, "Frame Decode & Encode must preserve Header")
}
