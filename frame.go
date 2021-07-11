package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// Frame describes the WT-0001 fridge state
// 24-byte payload from fridge
// It's the notification we get over the bluetooth attribute protocol
// e.g.	var notificationBytes = []byte{0xfe, 0xfe, 0x15, 0x02, 0x01, 0x01,
// 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00,
// 0x48, 0x64, 0x0e, 0x03, 0x05, 0x55}
type Frame struct {
	// Headers
	Header      uint16
	DataLen     uint8 // How many bytes follow including this byte through checksum
	CommandCode byte  // Unknown byte 2, maybe this is the command code?

	// Main settings
	Locked  uint8 // Keypad lock
	On      uint8 // Soft power state
	EcoMode uint8 // Power efficient mode
	HLvl    uint8 // Input voltage cutoff level H/M/L
	TempSet int8  // Desired temperature (thermostat)

	// Configuration settings
	E1 int8  // E1: Thermostat setting upper bound
	E2 int8  // E2: Thermostat setting lower bound
	E3 int8  // E3? Advanced Setting Maybe left hysterisis
	E4 int8  // E4 Advanced setting zero when in F mode maybe?
	E5 uint8 // E5 is F or C mode for whole system
	E6 int8  // E6 Advanced setting zero when in F mode maybe?
	E7 int8  // E7 Advanced setting zero when in F mode maybe?
	E8 int8  // E8 Advanced setting Left TC:T<-12degC
	E9 int8  // E9 Advanced setting Maybe start delay (narrow this first before temp values)

	// Sensors
	Temp    int8  // Fridge temp in degrees farenheit, but also in C
	UB17    byte  // Unknown byte 17, fridge battery level?
	InputV1 uint8 // Input voltage MSB
	InputV2 uint8 // Input voltage MSB

	// Error check
	Checksum uint16
}

// TODO use this in frame patch updates when sending to the fridge
func checksum(buf []byte) uint16 {
	sum := uint16(0)

	for ; len(buf) >= 1; buf = buf[1:] {
		sum += uint16(buf[0])
	}

	return sum
}

// 0xfe, 0xfe, 0x15, 0x01, 0x01, 0x01, 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00, 0x47, 0x64, 0x0e, 0x03, 0x05, 0x53

func (f *Frame) Valid() error {
	buf, err := f.MarshalBinary()

	if err != nil {
		return fmt.Errorf("Checksum failed: %s", err)
	}
	// Sum first 22 bytes
	summable := buf[:len(buf)-2-1]
	fmt.Println("summable", summable)
	calculatedSum := checksum(summable)

	if calculatedSum != f.Checksum {
		return fmt.Errorf("Frame data not valid: %v != %v", calculatedSum, f.Checksum)
	}
	return nil
}

func (f *Frame) MarshalJSON() ([]byte, error) {
	j, err := json.Marshal(*f)
	if err != nil {
		return nil, fmt.Errorf("Frame MarshalJSON error: %s", err)
	}
	return j, nil
}

// TODO
// func (f *Frame) UnmarshalJSON() ([]byte, error) {
//  // TODO use checksum here
// 	return json.Unmarshal(f)
// }

func (f Frame) String() string {
	return fmt.Sprintf(
		"UB1=%v CommandCode=%d Lock=%d On=%d Eco=%d HLvl=%d TempSet=%d E1=%d E2=%d E3=%d E4=%d E5=%d E6=%d E7=%d E8=%d E9=%d Temp=%d UB17=%d V=%d.%d CS=%d",
		f.DataLen,
		f.CommandCode,
		f.Locked,
		f.On,
		f.EcoMode,
		f.HLvl,
		f.TempSet,
		f.E1,
		f.E2,
		f.E3,
		f.E4,
		f.E5,
		f.E6,
		f.E7,
		f.E8,
		f.E9,
		f.Temp,
		f.UB17,
		f.InputV1,
		f.InputV2,
		f.Checksum,
	)
}

func (f *Frame) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, f); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

func (f *Frame) UnmarshalBinary(input []byte) error {
	// read into a new frame
	r := bytes.NewReader(input)
	if err := binary.Read(r, binary.BigEndian, f); err != nil {
		return err
	}
	return nil
}

// NewFrame creates a frame from byte buffer
func NewFrame(input []byte) (Frame, error) {
	var fr Frame
	err := fr.UnmarshalBinary(input)
	if err != nil {
		return fr, fmt.Errorf("Failed to UnmarshalBinary: %s", err)
	}
	err = fr.Valid()
	return fr, err
}
