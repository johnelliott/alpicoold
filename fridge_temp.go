package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

var dataLenSetTemp int8 = 0x4
var cmdCodeSetTemp int8 = 0x5

type SetTempCommand struct {
	Preamble    uint16
	DataLen     int8 // 4
	CommandCode int8 // 5
	Temp        int8
	Checksum    uint16
}

func NewSetTempCommand(temp int8) ([]byte, error) {
	log.Debug("NewSetTempCommand", temp)
	// Known data
	c := SetTempCommand{
		Preamble:    Preamble,
		DataLen:     dataLenSetTemp,
		CommandCode: cmdCodeSetTemp,
		Temp:        temp,
	}
	// KISS checksum
	checksum := c.Preamble>>8 +
		c.Preamble&0xff +
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
