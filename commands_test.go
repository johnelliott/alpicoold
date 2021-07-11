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

	t.Run("NewSetTempCommand", func(t *testing.T) {
		b, err := NewSetTempCommand(0x25)
		if err != nil {
			t.Fatalf("Failed to MarshalBinary: %s", err)
		}
		dl := b[2]
		if dl != 4 {
			t.Fatalf("Bad data length")
		}
		cc := b[3]
		if cc != 5 {
			t.Fatalf("Bad command code %v", cc)
		}
		temp := b[4]
		if temp != 0x25 {
			t.Fatalf("bad temp value: %x", temp)
		}

		// sum should be 0x22c
		sumMSB := b[5]
		sumLSB := b[6]
		if sumMSB != 0xff {
			t.Fatalf("Bad first checksum byte %v", sumMSB)
		}
		if sumLSB != 0x2c {
			t.Fatalf("Bad final checksum byte %v", sumLSB)
		}
	})
}

/*
func TestPingCommand(t *testing.T) {
	t.Run("UnmarshalBinary", func(t *testing.T) {
		c := PingCommand{}
		err := c.UnmarshalBinary(Ping)
		if err != nil {
			t.Fatalf("Failed to UnmarshalBinary: %s", err)
		}
		if c.DataLen != 3 {
			t.Fatalf("Bad data length")
		}
		if c.CommandCode != 1 {
			t.Fatalf("Bad command code")
		}
	})

	t.Run("MarshalBinary", func(t *testing.T) {
		c := PingCommand{
			Preamble:    0xfefe,
			DataLen:     0x3,
			CommandCode: 0x1,
			Checksum:    0x200,
		}

		b, err := c.MarshalBinary()
		if err != nil {
			t.Fatalf("Failed to MarshalBinary: %s", err)
		}

		expected := "fefe03010200"
		result := fmt.Sprintf("%x", b)
		if result != expected {
			t.Fatalf("Failed to UnmarshalBinary: %v %v", result, expected)
		}
	})
*/
