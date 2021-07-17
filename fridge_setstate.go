package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// e.g. var writeCommandBytes = []byte{0xfe, 0xfe, 0x11, 0x02, 0x01, 0x01,
// 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00,
// 0x04, 0x94}

var dataLenSetState byte = 0x11
var cmdCodeSetState byte = 0x2

type SetStateCommand struct {
	Preamble    uint16
	DataLen     byte
	CommandCode byte
	Settings
	Checksum uint16
}

func NewSetStateCommand(s Settings) ([]byte, error) {
	// Known data
	c := SetStateCommand{
		Preamble:    Preamble,
		DataLen:     dataLenSetState,
		CommandCode: cmdCodeSetState,
		Settings:    s,
	}

	c.updateCRC()

	b, err := c.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize State command: %s", err)
	}
	return b, err
}

func (c *SetStateCommand) CRC() uint16 {
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
		uint16(c.E9)
	return checksum
}

func (c *SetStateCommand) updateCRC() {
	c.Checksum = c.CRC()
}

func (c *SetStateCommand) Valid() error {
	if c.Preamble != Preamble {
		return fmt.Errorf("Incorrect preamble bytes")
	}
	if c.CommandCode != cmdCodeSetState {
		return fmt.Errorf("Incorrect command code")
	}
	if c.DataLen != dataLenSetState {
		return fmt.Errorf("Incorrect data payload length")
	}
	if c.Checksum != c.CRC() {
		return fmt.Errorf("CRC does not validate")
	}
	return nil
}

func (c *SetStateCommand) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, c); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

func (c *SetStateCommand) UnmarshalBinary(input []byte) error {
	r := bytes.NewReader(input)
	if err := binary.Read(r, binary.BigEndian, c); err != nil {
		return err
	}
	return c.Valid()
}

func (f *SetStateCommand) MarshalJSON() ([]byte, error) {
	j, err := json.Marshal(*f)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %s", err)
	}
	return j, nil
}

func (f *SetStateCommand) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, f)
	if err != nil {
		return fmt.Errorf("Error marshaling JSON: %s", err)
	}
	return nil
}