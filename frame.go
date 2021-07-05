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
	Thermostat          byte // Fridge temp set by user
	ThermoMaxDegSetting byte
	ThermoMinDegSetting byte // Number overflows, so 253 is -3F, 0 is 0F (negatives count down from 256)
	UB10                byte // Maybe left hysterisis
	UB11                byte // Advanced setting zero when in F mode maybe?
	FarenheitMode       byte // Maybe this
	UB13                byte // Advanced setting? zero when in F mode maybe?
	UB14                byte // Advanced setting? zero when in F mode maybe?
	UB15                byte // Advanced setting? Left TC:T<-12degC
	UB16                byte // Advanced setting? Maybe start delay (narrow this first before temp values)
	TempDegreesF        byte // Fridge temp in degrees farenheit, but also in C
	UB17                byte // Fridge Battery meter?, doesn't change with C and F modes
	InputVoltageVolts1  byte // Input voltage MSB
	InputVoltageVolts2  byte // Input voltage LSB
	UB20                byte // maybe CRC?
	UB21                byte // maybe CRC?
}

func (f Frame) String() string {
	return fmt.Sprintf(
		"UB1=%d UB2=%d KeypadLock=%d PoweredOn=%d EcoMode=%d InputVoltCutoffLvl=%d Thermostat=%d ThermoMaxDegSetting=%d ThermoMinDegSetting=%d UB10=%d UB11=%d FarenheitMode=%d UB13=%d UB14=%d UB15=%d UB16=%d TempDegreesF=%d UB17=%d InputVoltageVolts1=%d InputVoltageVolts2=%d UB20=%d UB21=%d",
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
		f.UB15,
		f.UB16,
		f.TempDegreesF,
		f.UB17,
		f.InputVoltageVolts1,
		f.InputVoltageVolts2,
		f.UB20,
		f.UB21,
	)
}

func (f Frame) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, f); err != nil {
		return buf.Bytes(), err
	}
	return buf.Bytes(), nil
}

func (f Frame) UnmarshalBinary(input []byte) (err error) {
	readableValue := bytes.NewReader(input)
	if err = binary.Read(readableValue, binary.LittleEndian, &f); err != nil {
		return
	}
	return
}

// NewFrame creates a frame from byte buffer
func NewFrame(input []byte) (fr Frame, err error) {
	return fr, fr.UnmarshalBinary(input)
}
