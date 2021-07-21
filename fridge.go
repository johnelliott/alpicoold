package main

// The WT-0001 frame header for all frames
var Preamble uint16 = 0xfefe

// Sensors contains all the sensors exposed via bluetooth
type Sensors struct {
	Temp    int8  // Fridge temp in degrees farenheit, but also in C
	UB17    byte  // Unknown byte 17, fridge battery level?
	InputV1 uint8 // Input voltage volts
	InputV2 uint8 // Input voltage volt tenths
}

// Settings contains the main fridge settings set by the user
type Settings struct {
	Locked  bool  // Keypad lock
	On      bool  // Soft power state
	EcoMode bool  // Power efficient mode
	HLvl    uint8 // Input voltage cutoff level H/M/L
	TempSet int8  // Desired temperature (thermostat)
	E1      int8  // E1: Thermostat setting upper bound
	E2      int8  // E2: Thermostat setting lower bound
	E3      int8  // E3? Advanced Setting Maybe left hysterisis
	E4      int8  // E4 Advanced setting zero when in F mode maybe?
	E5      bool  // E5 is F or C mode for whole system
	E6      int8  // E6 Advanced setting zero when in F mode maybe?
	E7      int8  // E7 Advanced setting zero when in F mode maybe?
	E8      int8  // E8 Advanced setting Left TC:T<-12degC
	E9      uint8 // E9 Advanced setting Maybe start delay (narrow this first before temp values)
}

// Ping requests notifications, and is static, so it needs no associated code
// Command code 1
var PingCommand = []byte{0xfe, 0xfe, 0x3, 0x1, 0x2, 0x0} // Get back notification state

// TODO factory reset command code 3

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
