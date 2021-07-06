package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Frame describes the WT-0001 fridge state
// TODO :GoAddTags for JSON
type Frame struct {
	Header              [2]byte
	UB1                 byte // always seems to be 21
	UB2                 byte
	KeypadLock          byte // `json:"keypadLock"`
	PoweredOn           byte // Soft power state has to be somewhere here
	EcoMode             byte
	InputVoltCutoffLvl  byte // Car battery protection level
	Thermostat          int8 // Fridge temp set by user
	ThermoMaxDegSetting int8
	ThermoMinDegSetting int8  // Number overflows, so 253 is -3F, 0 is 0F (negatives count down from 256)
	UB10                byte  // Advanced Setting Maybe left hysterisis
	UB11                byte  // Advanced setting zero when in F mode maybe?
	FarenheitMode       byte  // Maybe this
	UB13                int8  // Advanced setting zero when in F mode maybe?
	UB14                int8  // Advanced setting zero when in F mode maybe?
	LeftTCLT12DegC      int8  // Advanced setting Left TC:T<-12degC
	UB16                byte  // Advanced setting Maybe start delay (narrow this first before temp values)
	TempDegreesF        int8  // Fridge temp in degrees farenheit, but also in C
	UB17                byte  // Fridge Battery meter?, doesn't change with C and F modes
	InputVoltageVolts1  uint8 // Input voltage MSB
	InputVoltageVolts2  uint8 // Input voltage LSB
	UB20                byte  // maybe CRC?
	UB21                byte  // maybe CRC?
}

func (f Frame) String() string {
	return fmt.Sprintf(
		"UB1=%d UB2=%d Lock=%d Pow=%d Eco=%d VCut=%d Thermo=%d TMax=%d TMin=%d UB10=%d UB11=%d Fheit=%d UB13=%d UB14=%d Adv1=%d UB16=%d Temp=%d UB17=%d V=%d.%d UB20=%d UB21=%d",
		f.UB1,
		f.UB2,
		f.KeypadLock,
		f.PoweredOn,
		f.EcoMode,
		f.InputVoltCutoffLvl,
		f.Thermostat,
		f.ThermoMaxDegSetting,
		f.ThermoMinDegSetting,
		f.UB10,
		f.UB11,
		f.FarenheitMode,
		f.UB13,
		f.UB14,
		f.LeftTCLT12DegC,
		f.UB16,
		f.TempDegreesF,
		f.UB17,
		f.InputVoltageVolts1,
		f.InputVoltageVolts2,
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
