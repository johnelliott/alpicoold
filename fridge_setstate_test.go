package main

import (
	"fmt"
	"testing"
)

func TestSetStateCommand(t *testing.T) {
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
	t.Run("CRC", func(t *testing.T) {
		c := SetStateCommand{
			Preamble:    Preamble,
			DataLen:     0x11,
			CommandCode: 0x2,
			Settings: Settings{
				Locked:  1,
				On:      1,
				EcoMode: 1,
				HLvl:    1,
				TempSet: 0x43,
				E1:      0x44,
				E2:      0xfc,
				E3:      0x04,
				E4:      0x00,
				E5:      0x01,
				E6:      0x00,
				E7:      0x00,
				E8:      0xfb,
				E9:      0x00,
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
				Locked:  1,
				On:      1,
				EcoMode: 1,
				HLvl:    1,
				TempSet: 0x43,
				E1:      0x44,
				E2:      0xfc,
				E3:      0x04,
				E4:      0x00,
				E5:      0x01,
				E6:      0x00,
				E7:      0x00,
				E8:      0xfb,
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

	t.Run("NewSetStateCommand", func(t *testing.T) {
		// 0xfe, 0xfe, 0x11, 0x02,			0x00, 0x01, 0x01, 0x02, 0x24, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x04, 0x77
		// 0xfe, 0xfe, 0x11, 0x02,			0x01, 0x01, 0x01, 0x01, 0x43, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x04, 0x96
		// 0xfe 0xfe 0x11 0x02 0x01 0x01 0x01 0x01 0x43 0x44 0xfc 0x04 0x00 0x01 0x00 0x00 0xfb 0x00 0x04 0x96

		b, err := NewSetStateCommand(Settings{
			Locked:  1,
			On:      1,
			EcoMode: 1,
			HLvl:    1,
			TempSet: 0x43,
			E1:      0x44,
			E2:      0xfc,
			E3:      0x04,
			E4:      0x00,
			E5:      0x01,
			E6:      0x00,
			E7:      0x00,
			E8:      0xfb,
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
