package main

import (
	"fmt"
	"testing"
)

func TestSetTempCommand(t *testing.T) {
	t.Run("UnmarshalBinary", func(t *testing.T) {
		setTempBytes := []byte{0xfe, 0xfe, 0x04, 0x05, 0x25, 0x02, 0x2a}
		c := SetTempCommand{}
		err := c.UnmarshalBinary(setTempBytes)
		if err != nil {
			t.Fatalf("Failed to UnmarshalBinary: %s", err)
		}
		if c.DataLen != 4 {
			t.Fatalf("Bad data length %v", c.DataLen)
		}
		if c.CommandCode != 5 {
			t.Fatalf("Bad command code %v", c.CommandCode)
		}
		// 37 degrees f is 0x25
		if c.Temp != 0x25 {
			t.Fatalf("Bad temperature %v", c.Temp)
		}
	})

	t.Run("MarshalBinary", func(t *testing.T) {
		c := SetTempCommand{
			Preamble:    0xfefe,
			DataLen:     0x4,
			CommandCode: 0x5,
			Temp:        0x26,
			Checksum:    0x22b,
		}

		b, err := c.MarshalBinary()
		if err != nil {
			t.Fatalf("Failed to MarshalBinary: %s", err)
		}

		expected := "fefe040526022b"
		result := fmt.Sprintf("%x", b)
		if result != expected {
			t.Fatalf("Failed to marshal to binary: %v %v", result, expected)
		}
	})
}
