package k25

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// Preamble is the WT-0001 frame header for all frames
var Preamble uint16 = 0xfefe

// Sensors contains all the sensors exposed via bluetooth
type Sensors struct {
	Temp    int8 // Fridge temp in degrees farenheit, but also in C
	UB17    int8 // Unknown byte 17, fridge battery level?
	InputV1 int8 // Input voltage volts
	InputV2 int8 // Input voltage volt tenths
}

// Settings contains the main fridge settings set by the user
type Settings struct {
	Locked                      bool // Keypad lock
	On                          bool // Soft power state
	EcoMode                     bool // Power efficient mode
	HLvl                        int8 // Input voltage cutoff level H/M/L
	TempSet                     int8 // Desired temperature (thermostat)
	HighestTempSettingMenuE2    int8 // E2: Thermostat setting lower bound
	LowestTempSettingMenuE1     int8 // E1: Thermostat setting upper bound
	HysteresisMenuE3            int8 // E3: Hysteresis i.e. Temp return setting
	SoftStartDelayMinMenuE4     int8 // E4: Soft on start delay in minutes
	CelsiusFahrenheitModeMenuE5 bool // E5  Celsius or Fahrenheit mode for fridge
	// Temperature compensation values are labeled in celsius in the alpicool
	// app. The values depend on the CelsiusFahrenheitModeMenuE5 setting, and
	// are in Fahrenheit units when the fridge is in Fahrenheit mode
	TempCompGTEMinus6DegCelsiusMenuE6                    int8 // E6: High range temperature compensation
	TempCompGTEMinus12DegCelsiusLTMinus6DegCelsiusMenuE7 int8 // E8: Mid range temperature compensation
	TempCompLTMinus12DegCelsiusMenuE8                    int8 // E8: Low range temperature compensation
	TempCompShutdownMenuE9                               int8 // E9: Shutdown? Perhaps a lower bound?
}

// PingCommand requests notifications, and is static, so it needs no associated code
// Command code 1
var PingCommand = []byte{0xfe, 0xfe, 0x3, 0x1, 0x2, 0x0} // Get back notification state

// TODO factory reset command code 3

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

// CRC cyclic redundancy check
func (c *StatusReport) CRC() uint16 {
	// KISS checksum
	// sum of bytes before CRC
	// big endian
	checksum := c.Preamble>>8 +
		c.Preamble&0xff +
		uint16(uint8(c.DataLen)) +
		uint16(uint8(c.CommandCode)) +
		uint16(uint8(c.HLvl)) +
		uint16(uint8(c.TempSet)) +
		uint16(uint8(c.LowestTempSettingMenuE1)) +
		uint16(uint8(c.HighestTempSettingMenuE2)) +
		uint16(uint8(c.HysteresisMenuE3)) +
		uint16(uint8(c.SoftStartDelayMinMenuE4)) +
		uint16(uint8(c.TempCompGTEMinus6DegCelsiusMenuE6)) +
		uint16(uint8(c.TempCompGTEMinus12DegCelsiusLTMinus6DegCelsiusMenuE7)) +
		uint16(uint8(c.TempCompLTMinus12DegCelsiusMenuE8)) +
		uint16(uint8(c.TempCompShutdownMenuE9)) +
		uint16(uint8(c.Temp)) +
		uint16(uint8(c.UB17)) +
		uint16(uint8(c.InputV1)) +
		uint16(uint8(c.InputV2))

	// flags
	for _, b := range []bool{c.Locked, c.On, c.EcoMode, c.CelsiusFahrenheitModeMenuE5} {
		if b {
			checksum++
		}
	}
	return checksum
}

// Valid checks for a valid data frame
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

	// validate this isn't nonsense
	if e := c.Valid(); e != nil {
		return nil, e
	}

	b, err := c.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize State command: %s", err)
	}
	return b, err
}

// CRC cyclic redundancy check
func (c *SetStateCommand) CRC() uint16 {
	// KISS checksum
	// sum of bytes before CRC
	// big endian
	checksum := c.Preamble>>8 +
		c.Preamble&0xff +
		uint16(uint8(c.DataLen)) +
		uint16(uint8(c.CommandCode)) +
		uint16(uint8(c.HLvl)) +
		uint16(uint8(c.TempSet)) +
		uint16(uint8(c.LowestTempSettingMenuE1)) +
		uint16(uint8(c.HighestTempSettingMenuE2)) +
		uint16(uint8(c.HysteresisMenuE3)) +
		uint16(uint8(c.SoftStartDelayMinMenuE4)) +
		uint16(uint8(c.TempCompGTEMinus6DegCelsiusMenuE6)) +
		uint16(uint8(c.TempCompGTEMinus12DegCelsiusLTMinus6DegCelsiusMenuE7)) +
		uint16(uint8(c.TempCompLTMinus12DegCelsiusMenuE8)) +
		uint16(uint8(c.TempCompShutdownMenuE9))

	// Flags
	for _, b := range []bool{c.Locked, c.On, c.EcoMode, c.CelsiusFahrenheitModeMenuE5} {
		if b {
			checksum++
		}
	}
	return checksum
}

func (c *SetStateCommand) updateCRC() {
	c.Checksum = c.CRC()
}

// Valid checks for a valid data frame
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
	if c.TempSet > c.HighestTempSettingMenuE2 {
		return fmt.Errorf("Temp setting > maximum")
	}
	if c.TempSet < c.LowestTempSettingMenuE1 {
		return fmt.Errorf("Temp setting < minimum")
	}
	return nil
}

// MarshalBinary serializes a state set command
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
