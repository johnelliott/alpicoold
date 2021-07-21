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

		// sum should be 0x22a
		sumMSB := b[5]
		sumLSB := b[6]
		if sumMSB != 0x2 {
			t.Fatalf("Bad first checksum byte %#v, b=%#v", sumMSB, b)
		}
		if sumLSB != 0x2a {
			t.Fatalf("Bad final checksum byte %#v, b=%#v", sumLSB, b)
		}
	})

	t.Run("CRC", func(t *testing.T) {
		c := SetStateCommand{
			Preamble:    Preamble,
			DataLen:     17,
			CommandCode: 2,
			Settings: Settings{
				Locked:  true,
				On:      true,
				EcoMode: true,
				HLvl:    1,
				TempSet: 67,
				E1:      68,
				E2:      -4,
				E3:      4,
				E4:      0,
				E5:      true,
				E6:      0,
				E7:      0,
				E8:      -5,
				E9:      0,
			},
		}

		expected := uint16(0x0496)
		result := c.CRC()
		if result != expected {
			t.Fatalf("Fail:\n%v\n%v", result, expected)
		}
	})

	t.Run("MarshalBinary", func(t *testing.T) {
		c := SetStateCommand{
			Preamble:    Preamble,
			DataLen:     0x11,
			CommandCode: 0x2,
			Settings: Settings{
				Locked:  true,
				On:      true,
				EcoMode: true,
				HLvl:    1,
				TempSet: 0x43,
				E1:      0x44,
				E2:      -4,
				E3:      0x04,
				E4:      0x00,
				E5:      true,
				E6:      0x00,
				E7:      0x00,
				E8:      -5,
				E9:      0x00,
			},
			Checksum: 0x496,
		}
		expected := "0xfe 0xfe 0x11 0x02 0x01 0x01 0x01 0x01 0x43 0x44 0xfc 0x04 0x00 0x01 0x00 0x00 0xfb 0x00 0x04 0x96"

		b, err := c.MarshalBinary()
		if err != nil {
			t.Fatalf("Failed to MarshalBinary: %s", err)
		}
		result := fmt.Sprintf("% 0#x", b)
		if result != expected {
			t.Fatalf("Fail:\n%v\n%v", result, expected)
		}
	})
}

func TestSetStateCommand(t *testing.T) {
	t.Run("NewSetStateCommand", func(t *testing.T) {
		// 0xfe, 0xfe, 0x11, 0x02,			0x00, 0x01, 0x01, 0x02, 0x24, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x04, 0x77
		// 0xfe, 0xfe, 0x11, 0x02,			0x01, 0x01, 0x01, 0x01, 0x43, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x04, 0x96
		// 0xfe 0xfe 0x11 0x02 0x01 0x01 0x01 0x01 0x43 0x44 0xfc 0x04 0x00 0x01 0x00 0x00 0xfb 0x00 0x04 0x96

		b, err := NewSetStateCommand(Settings{
			Locked:  true,
			On:      true,
			EcoMode: true,
			HLvl:    1,
			TempSet: 0x43,
			E1:      0x44,
			E2:      -4,
			E3:      0x04,
			E4:      0x00,
			E5:      true,
			E6:      0x00,
			E7:      0x00,
			E8:      -5,
			E9:      0x00,
		})
		if err != nil {
			t.Fatalf("Failed to make new set state command: %s", err)
		}
		expected := "0xfe 0xfe 0x11 0x02 0x01 0x01 0x01 0x01 0x43 0x44 0xfc 0x04 0x00 0x01 0x00 0x00 0xfb 0x00 0x04 0x96"
		result := fmt.Sprintf("% 0#x", b)
		if result != expected {
			t.Fatalf("Fail:\n%v\n%v", result, expected)
		}
	})
}
