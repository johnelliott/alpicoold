package main

import (
	"context"
	"sync"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	hclog "github.com/brutella/hc/log"
	log "github.com/sirupsen/logrus"
)

// HKClient is an imaginary client for homekit preparation
func HKClient(ctx context.Context, wg *sync.WaitGroup, fridgeStatus chan StatusReport) {
	wg.Add(1)
	defer func() {
		log.Trace("HK client calling done on main wait group")
		wg.Done()
	}()
	log.Trace("HKClient start")

	hclog.Debug.SetOutput(log.StandardLogger().WriterLevel(log.TraceLevel))
	hclog.Info.SetOutput(log.StandardLogger().WriterLevel(log.DebugLevel))

	// Set up Lock button
	infoLockButton := accessory.Info{
		Name:         "Lock K25",
		SerialNumber: "1",
		Manufacturer: "johnelliott.org",
		Model:        "WT-0001 Bridge",
		// FirmwareRevision: "0.0.1",
		// ID:               1,
	}
	lockButton := accessory.NewSwitch(infoLockButton)
	lockButton.Switch.On.OnValueRemoteUpdate(func(on bool) {
		log.Warnf("User flip lock switch %v\n", on)
		// just set it for them for now, do this via commands later
		lockButton.Switch.On.SetValue(on)
	})

	// Set up on button
	infoOnButton := accessory.Info{
		Name:         "On K25",
		SerialNumber: "1",
		Manufacturer: "johnelliott.org",
		Model:        "WT-0001 Bridge",
		// FirmwareRevision: "0.0.1",
		// ID:               1,
	}
	onButton := accessory.NewSwitch(infoOnButton)
	onButton.Switch.On.OnValueRemoteUpdate(func(on bool) {
		log.Warnf("User flip ON switch %v\n", on)
		// just set it for them for now, do this via commands later
		onButton.Switch.On.SetValue(on)
	})

	// Set up thermostat and command callbacks
	infoThermo := accessory.Info{
		Name: "Alpicool K25",
		// SerialNumber:     "1",
		Manufacturer: "johnelliott.org",
		Model:        "WT-0001 Bridge",
		// FirmwareRevision: "0.0.1",
		// ID:               2,
	}
	th := accessory.NewThermostat(infoThermo, FtoC(40), FtoC(-10), FtoC(99), 1)
	th.Thermostat.CurrentHeatingCoolingState.SetValue(2)
	th.Thermostat.TargetHeatingCoolingState.SetValue(0)
	th.Thermostat.TemperatureDisplayUnits.SetValue(1) // 0=C, 1=F

	th.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(nt float64) {
		log.Warnf("User wants new temp %v\n", CtoF(nt))
		// just set it for them for now, do this via commands later
		th.Thermostat.TargetTemperature.SetValue(nt)
	})

	config := hc.Config{Pin: "80000000", StoragePath: "./homekitdb"}
	t, err := hc.NewIPTransport(config, th.Accessory, lockButton.Accessory, onButton.Accessory)
	if err != nil {
		log.Error(err)
	}

	go func() {
		// wg.Add(1)
		// defer func() {
		// 	log.Trace("HK client loop calling done on main wait group")
		// 	wg.Done()
		// }()
		log.Trace("HK client looping now")
		for {
			select {
			case <-ctx.Done():
				log.Trace("HKClient ctx canceled")
				<-t.Stop()
				log.Trace("HKClient stopped")
				return
			case s := <-fridgeStatus:
				log.Tracef("Homekit got fridge status %v\n", s.Temp)
				var t float64
				var tempSetting float64
				if s.E5 == 1 {
					t = FtoC(float64(s.Temp))
					tempSetting = FtoC(float64(s.TempSet))
				} else {
					t = float64(s.Temp)
					tempSetting = float64(s.TempSet)
				}

				// switches/buttons
				onButton.Switch.On.SetValue(s.On == 1)
				lockButton.Switch.On.SetValue(s.Locked == 1)
				// Required
				if s.On == 1 {
					th.Thermostat.CurrentHeatingCoolingState.SetValue(2)
					th.Thermostat.TargetHeatingCoolingState.SetValue(2)
				} else {
					th.Thermostat.CurrentHeatingCoolingState.SetValue(0)
					th.Thermostat.TargetHeatingCoolingState.SetValue(0)
				}
				th.Thermostat.CurrentTemperature.SetValue(t)
				th.Thermostat.TargetTemperature.SetValue(tempSetting)
				// th.Thermostat.TemperatureDisplayUnits.SetValue(int(s.E5)) // 0=C, 1=F
				th.Thermostat.TemperatureDisplayUnits.SetValue(1) // 0=C, 1=F

				// Optional
				// th.Thermostat.CurrentHeatingCoolingState.SetMaxValue(int(s.E1))
				// th.Thermostat.CurrentHeatingCoolingState.SetMinValue(int(s.E2))
			}
		}
	}()

	// wait for and trash one value
	// TODO see if I can remove this to gain back 1 second
	// <-fridgeStatus
	// Start homekit transport
	t.Start()
}

func FtoC(f float64) float64 {
	return (f - 32) * 5 / 9
}
func CtoF(f float64) float64 {
	return f*9/5 + 32
}
