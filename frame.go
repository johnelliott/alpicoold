package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

var (
	// Pi stuff
	zeroAdapter = "hci0"
	// Characteristics
	serviceUUID         = "00001234-0000-1000-8000-00805f9b34fb"
	writeableFridgeUUID = "00001235-0000-1000-8000-00805f9b34fb" // Writable
	readeableFridgeUUID = "00001236-0000-1000-8000-00805f9b34fb" // Read Notify
	descriptorUUID      = "00002902-0000-1000-8000-00805f9b34fb"
	// Commands
	magicPayload = []byte{0xfe, 0xfe, 0x3, 0x1, 0x2, 0x0}
)

// Frame describes the WT-0001 fridge state
// TODO :GoAddTags for JSON
type Frame struct {
	Header  [2]byte
	UB1     byte // always seems to be 21
	UB2     byte
	Locked  byte // `json:"keypadLock"`
	On      byte // Soft power state has to be somewhere here
	EcoMode byte
	HLvl    byte  // Input voltage cutoff level H/M/L
	TempSet int8  // Fridge temp set by user
	E1      int8  // E1: Thermostat setting upper bound
	E2      int8  // E2: Thermostat setting lower bound, Number overflows, so 253 is -3F, 0 is 0F (negatives count down from 256)
	E3      byte  // E3? Advanced Setting Maybe left hysterisis
	E4      byte  // Advanced setting zero when in F mode maybe?
	E5      byte  // E5 is F or C mode for whole system
	E6      int8  // Advanced setting zero when in F mode maybe?
	E7      int8  // E7: Advanced setting zero when in F mode maybe?
	E8      int8  // E8: Advanced setting Left TC:T<-12degC
	E9      int8  // E4? Advanced setting Maybe start delay (narrow this first before temp values)
	Temp    int8  // Fridge temp in degrees farenheit, but also in C
	UB17    byte  // Fridge Battery meter?, doesn't change with C and F modes
	InputV1 uint8 // Input voltage MSB
	InputV2 uint8 // Input voltage LSB
	UB20    byte  // maybe CRC?
	UB21    byte  // maybe CRC?
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
// 	return json.Unmarshal(f)
// }

func (f Frame) String() string {
	return fmt.Sprintf(
		"UB1=%d UB2=%d Lock=%d On=%d Eco=%d HLvl=%d TempSet=%d E1=%d E2=%d E3=%d E4=%d E5=%d E6=%d E7=%d E8=%d E9=%d Temp=%d UB17=%d V=%d.%d UB20=%d UB21=%d",
		f.UB1,
		f.UB2,
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
		f.UB20,
		f.UB21,
	)
}

func (f *Frame) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, f); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

func (f *Frame) UnmarshalBinary(input []byte) error {
	// read into a new frame
	r := bytes.NewReader(input)
	if err := binary.Read(r, binary.LittleEndian, f); err != nil {
		return err
	}
	return nil
}

// NewFrame creates a frame from byte buffer
func NewFrame(input []byte) (Frame, error) {
	var fr Frame
	err := fr.UnmarshalBinary(input)
	return fr, err
}
