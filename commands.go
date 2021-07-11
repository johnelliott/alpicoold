package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// The WT-0001 frame header for all frames
var Preamble uint16 = 0xfefe

// Sensors contains all the sensors exposed via bluetooth
type Sensors struct {
	Temp    byte // Fridge temp in degrees farenheit, but also in C
	UB17    byte // Unknown byte 17, fridge battery level?
	InputV1 byte // Input voltage volts
	InputV2 byte // Input voltage volt tenths
}

// Settings contains the main fridge settings set by the user
type Settings struct {
	Locked  byte // Keypad lock
	On      byte // Soft power state
	EcoMode byte // Power efficient mode
	HLvl    byte // Input voltage cutoff level H/M/L
	TempSet byte // Desired temperature (thermostat)
	E1      byte // E1: Thermostat setting upper bound
	E2      byte // E2: Thermostat setting lower bound
	E3      byte // E3? Advanced Setting Maybe left hysterisis
	E4      byte // E4 Advanced setting zero when in F mode maybe?
	E5      byte // E5 is F or C mode for whole system
	E6      byte // E6 Advanced setting zero when in F mode maybe?
	E7      byte // E7 Advanced setting zero when in F mode maybe?
	E8      byte // E8 Advanced setting Left TC:T<-12degC
	E9      byte // E9 Advanced setting Maybe start delay (narrow this first before temp values)
}

// Ping requests notifications, and is static, so it needs no associated code
// Command code 1
var PingCommand = []byte{0xfe, 0xfe, 0x3, 0x1, 0x2, 0x0} // Get back notification state

// 19-byte payload from fridge
// e.g. var writeCommandBytes = []byte{0xfe, 0xfe, 0x11, 0x02, 0x01, 0x01,
// 0x01, 0x00, 0x42, 0x44, 0xfc, 0x04, 0x00, 0x01, 0x00, 0x00, 0xfb, 0x00,
// 0x04, 0x94}
// Command code 2
type SetStateCommand struct {
	Preamble    uint16
	DataLen     byte
	CommandCode byte // 2
	Settings
	Checksum uint16
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
	return nil
}
func NewSetStateCommand(s Settings) ([]byte, error) {
	// Known data
	c := SetStateCommand{
		Preamble:    Preamble,
		DataLen:     4,
		CommandCode: 5,
		Settings:    s,
	}
	// KISS checksum
	checksum := c.Preamble +
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

	c.Checksum = checksum
	fmt.Printf("c.Checksum %0#x\n", c.Checksum)

	b, err := c.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize State command: %s", err)
	}
	return b, err
}

// TODO factory reset command code 3

// StatusReport is the bluetooth notification payload with full fridge state
type StatusReport struct {
	Preamble    uint16
	DataLen     byte
	CommandCode byte // 4?
	Settings
	Sensors
	Checksum uint16
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

type SetTempCommand struct {
	Preamble    uint16
	DataLen     byte // 4
	CommandCode byte // 5
	Temp        byte
	Checksum    uint16
}

func NewSetTempCommand(temp byte) ([]byte, error) {
	// Known data
	c := SetTempCommand{
		Preamble:    Preamble,
		DataLen:     4,
		CommandCode: 5,
		Temp:        temp,
	}
	// KISS checksum
	checksum := c.Preamble +
		uint16(c.DataLen) +
		uint16(c.CommandCode) +
		uint16(c.Temp)

	c.Checksum = checksum
	fmt.Printf("c.Checksum %0#x\n", c.Checksum)

	b, err := c.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize temp command: %s", err)
	}
	return b, err
}
func (c *SetTempCommand) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, c); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil

}
func (c *SetTempCommand) UnmarshalBinary(input []byte) error {
	r := bytes.NewReader(input)
	if err := binary.Read(r, binary.BigEndian, c); err != nil {
		return err
	}
	return nil
}
func (r *SetTempCommand) MarshalJSON() ([]byte, error) {
	j, err := json.Marshal(*r)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize temp command: %s", err)
	}
	return j, nil
}

/*
var tempSet38DegFPayload = []byte{0xfe, 0xfe, 0x4, 0x5, 0x26, 0x2, 0x2b} // Set temp

// PingCommand is how temp is set
// This is a send back notification thing, seems to just send
// So because there's no data payload, this is always the same packet
type PingCommand struct {
	Preamble    uint16
	DataLen     byte
	CommandCode byte // 1
	Checksum    uint16
}

func (c *PingCommand) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, c); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}
func (c *PingCommand) UnmarshalBinary(input []byte) error {
	// read into a new frame
	r := bytes.NewReader(input)
	if err := binary.Read(r, binary.BigEndian, c); err != nil {
		return err
	}
	return nil
}

var commandCodes = map[string]int{
	"ping":  1,
	"temp":  2,
	"state": 3,
}

// GenericCommand is how any command to this fridge is set
type GenericCommand struct {
	Header
	Payload []byte
	Checksum
}

func (c *GenericCommand) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, c); err != nil {
		return buf.Bytes(), err
	}
	switch c.CommandCode {
	case 0:
		return nil, nil
	}

	return buf.Bytes(), nil
}
*/
