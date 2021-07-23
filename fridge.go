package main

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
	LowestTempSettingMenuE1     int8 // E1: Thermostat setting upper bound
	HighestTempSettingMenuE2    int8 // E2: Thermostat setting lower bound
	HysteresisMenuE3            int8 // E3: Hysteresis i.e. Temp return setting
	SoftStartDelayMinMenuE4     int8 // E4: Soft on start delay in minutes
	CelsiusFahrenheitModeMenuE5 bool // E5  Celsius or Fahrenheit mode for fridge
	// Temperature compensation values are labeled in celsius in the alpicool
	// app. The values depend on the CelsiusFahrenheitModeMenuE5 setting, and
	// are in Fahrenheit units when the fridge is in Fahrenheit mode
	TempCompGTE6MinusDegCelsiusMenuE6                    int8 // E6: High range temperature compensation
	TempCompGTE12MinusDegCelsiusLT6MinusDegCelsiusMenuE7 int8 // E8: Mid range temperature compensation
	TempCompLT12MinusDegCelsiusMenuE8                    int8 // E8: Low range temperature compensation
	TempCompShutdownMenuE9                               int8 // E9: Shutdown? Perhaps a lower bound?
}

// PingCommand requests notifications, and is static, so it needs no associated code
// Command code 1
var PingCommand = []byte{0xfe, 0xfe, 0x3, 0x1, 0x2, 0x0} // Get back notification state

// TODO factory reset command code 3
