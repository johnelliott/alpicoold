package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

var dataLenStatusReport byte = 0x15
var cmdCodeStatusReport byte = 0x1

// StatusReport is the bluetooth notification payload with full fridge state
type StatusReport struct {
	Preamble    uint16
	DataLen     byte
	CommandCode byte
	Settings
	Sensors
	Checksum uint16
}

// TODO verify this is correct
func (c *StatusReport) CRC() uint16 {
	// KISS checksum
	// sum of bytes before CRC
	// big endian
	checksum := c.Preamble>>8 +
		c.Preamble&0xff +
		uint16(c.DataLen) +
		uint16(c.CommandCode) +
		uint16(c.Locked) +
		uint16(c.On) +
		uint16(c.EcoMode) +
		uint16(c.HLvl) +
		uint16(c.TempSet) +
		uint16(c.E1) +
		uint16(c.E2) +
		uint16(c.E3) +
		uint16(c.E4) +
		uint16(c.E5) +
		uint16(c.E6) +
		uint16(c.E7) +
		uint16(c.E8) +
		uint16(c.E9) +
		uint16(c.Temp) +
		uint16(c.UB17) +
		uint16(c.InputV1) +
		uint16(c.InputV2)
	return checksum
}

func (c *StatusReport) Valid() error {
	if c.Preamble != Preamble {
		return fmt.Errorf("Incorrect preamble bytes")
	}
	if c.CommandCode != cmdCodeStatusReport {
		return fmt.Errorf("Incorrect command code")
	}
	if c.DataLen != dataLenStatusReport {
		return fmt.Errorf("Incorrect data payload length")
	}
	if c.Checksum != c.CRC() {
		return fmt.Errorf("CRC does not validate")
	}
	return nil
}

func (r *StatusReport) UnmarshalBinary(input []byte) error {
	rd := bytes.NewReader(input)
	if err := binary.Read(rd, binary.BigEndian, r); err != nil {
		return err
	}
	return nil
}
func (r *StatusReport) MarshalJSON() ([]byte, error) {
	j, err := json.Marshal(*r)
	if err != nil {
		return nil, fmt.Errorf("Frame MarshalJSON error: %s", err)
	}
	return j, nil
}
