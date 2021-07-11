package main

import (
	"fmt"
	"testing"
)

func TestFrame(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		f := Frame{On: 1}
		fmt.Println("f", f)
		_, err := f.MarshalJSON()
		if err != nil {
			t.Fatal("failed")
		}
	})

	t.Run("Binary serialization", func(t *testing.T) {

		//	Header:   0xfefe
		//	DataLen:      0x15
		//	CommandCode:      0x01
		//	Locked:   0x01
		//	On:       0x01
		//	EcoMode:  0x01
		//	HLvl:     0x00
		//	TempSet:  0x42
		//	E1:       0x44
		//	E2:       0xfc
		//	E3:       0x04
		//	E4:       0x00
		//	E5:       0x01
		//	E6:       0x00
		//	E7:       0x00
		//	E8:       0xfb
		//	E9:       0x00
		//	Temp:     0x47
		//	UB17:     0x64
		//	InputV1:  0x0e
		//	InputV2:  0x03
		//	Checksum: 0x0553
		f := Frame{
			Header:      0xfefe,
			DataLen:     0x15,
			CommandCode: 0x01,
			Locked:      0x01,
			On:          0x01,
			EcoMode:     0x01,
			HLvl:        0x00,
			TempSet:     0x42,
			E1:          0x44,
			E2:          -4, // fc
			E3:          0x04,
			E4:          0x00,
			E5:          0x01,
			E6:          0x00,
			E7:          0x00,
			E8:          -5, // fb
			E9:          0x00,
			Temp:        0x47,
			UB17:        0x64,
			InputV1:     0x0e,
			InputV2:     0x03,
			Checksum:    0x553,
		}
		b, err := f.MarshalBinary()
		if err != nil {
			t.Fatalf("Failed to MarshalBinary: %0#x", b)
		}
		// t.Fatalf("look screwy: % 0#x", b)
	})

	// Checksums
	t.Run("Valid Checksum", func(t *testing.T) {
		f := Frame{
			Header:      0xfefe,
			DataLen:     0x15,
			CommandCode: 0x01,
			Locked:      0x01,
			On:          0x01,
			EcoMode:     0x01,
			HLvl:        0x00,
			TempSet:     0x42,
			E1:          0x44,
			E2:          -3, // ff is -1, fe is -2, fc is -3
			E3:          0x04,
			E4:          0x00,
			E5:          0x01,
			E6:          0x00,
			E7:          0x00,
			E8:          -4,
			E9:          0x00,
			Temp:        0x47,
			UB17:        0x64,
			InputV1:     0x0e,
			InputV2:     0x03,
			Checksum:    0x553,
		}
		err := f.Valid()
		if err != nil {
			t.Fatal("Valid checksum false negative", err)
		}
	})
	t.Run("Invalid Checksum", func(t *testing.T) {
		f := Frame{On: 1, Checksum: 1}
		fmt.Println("f", f)
		err := f.Valid()
		if err == nil {
			t.Fatal("Valid checksum false positive", err)
		}
	})

	t.Run("UnmarshalBinary", func(t *testing.T) {
		var validFrame = []byte{0xfe, 0xfe, 0x15, 0x01, 0x01, 0x01, 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x41, 0x64, 0x0e, 0x03, 0x05, 0x4d}
		f := Frame{}
		err := f.UnmarshalBinary(validFrame)
		if err != nil {
			t.Fatal("failed", err)
		}
	})

	t.Run("Creating valid frame", func(t *testing.T) {
		var validFrame = []byte{0xfe, 0xfe, 0x15, 0x01, 0x01, 0x01, 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x41, 0x64, 0x0e, 0x03, 0x05, 0x4d}
		input := validFrame
		_, err := NewFrame(input)
		if err != nil {
			t.Fatalf("fail Creating valid frame: %s", err)
		}
	})
	t.Run("Creating invalid frame should error", func(t *testing.T) {
		f := []byte{0xfe, 0xfe, 0x15, 0x01, 0x01, 0x01, 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x41, 0x64, 0x0e, 0x03, 0x05, 0x4e}
		_, err := NewFrame(f)
		if err == nil {
			t.Fatal("Failed to catch an invalid frame")
		}
	})
}
