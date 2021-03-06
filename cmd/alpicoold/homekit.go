package main

import (
	"context"
	"image"
	"math"
	"sync"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	hclog "github.com/brutella/hc/log"
	"github.com/brutella/hkcam"
	"github.com/brutella/hkcam/ffmpeg"
	log "github.com/sirupsen/logrus"
)

// HKSettings avoids lots of args to HKClient
type HKSettings struct {
	storagePath     string
	minVideoBitrate int
	multiStream     bool
	// Platform dependent flags
	inputDevice      string
	inputFilename    string
	loopbackFilename string
	h264Decoder      string
	h264Encoder      string
}

// HKClient is an imaginary client for homekit preparation
func HKClient(ctx context.Context, wg *sync.WaitGroup, fridge *Fridge, settings HKSettings) {
	wg.Add(1)
	defer func() {
		wg.Done()
		log.WithFields(log.Fields{
			"client": "HKClient",
		}).Trace("Calling done on main wait group")
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
	lockButton.Switch.On.OnValueRemoteUpdate(fridge.SetLocked)

	// On button
	infoOnButton := accessory.Info{
		Name:         "On K25",
		SerialNumber: "1",
		Manufacturer: "johnelliott.org",
		Model:        "WT-0001 Bridge",
		// FirmwareRevision: "0.0.1",
		// ID:               1,
	}
	onButton := accessory.NewSwitch(infoOnButton)
	onButton.Switch.On.OnValueRemoteUpdate(fridge.SetOn)

	// EcoMode button
	infoEcoModeButton := accessory.Info{
		Name:         "EcoMode K25",
		SerialNumber: "1",
		Manufacturer: "johnelliott.org",
		Model:        "WT-0001 Bridge",
		// FirmwareRevision: "0.0.1",
		// ID:               1,
	}
	ecoModeButton := accessory.NewSwitch(infoEcoModeButton)
	ecoModeButton.Switch.On.OnValueRemoteUpdate(fridge.SetEcoMode)

	// Thermostat
	infoThermo := accessory.Info{
		Name: "Alpicool K25",
		// SerialNumber:     "1",
		Manufacturer: "johnelliott.org",
		Model:        "WT-0001 Bridge",
		// FirmwareRevision: "0.0.1",
		// ID:               2,
	}
	// TODO see if I can set upper and lower bounds properly
	th := accessory.NewThermostat(infoThermo, FtoC(40), FtoC(-10), FtoC(99), 1)
	th.Thermostat.CurrentHeatingCoolingState.SetValue(2)
	th.Thermostat.TargetHeatingCoolingState.SetValue(0)
	th.Thermostat.TemperatureDisplayUnits.SetValue(1) // 0=C, 1=F

	th.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(newTempRawCelsius float64) {
		// Round to something reasonable
		newTemp := math.Round(newTempRawCelsius)
		log.Tracef("New TargetTemperature: %v %v %v", newTempRawCelsius, newTemp, byte(newTemp))
		fridge.tempSettingsC <- newTemp
		// just set it for them for now, do this via commands later
		// th.Thermostat.TargetTemperature.SetValue(newTemp)
	})
	// Camera setup

	if log.GetLevel() == log.TraceLevel {
		// TODO get something like this log.Debug.Enable()
		ffmpeg.EnableVerboseLogging()
	}

	switchInfo := accessory.Info{Name: "Camera", FirmwareRevision: "0.0.9", Manufacturer: "Matthias Hochgatterer"}
	cam := accessory.NewCamera(switchInfo)

	cfg := ffmpeg.Config{
		InputDevice:      settings.inputDevice,
		InputFilename:    settings.inputFilename,
		LoopbackFilename: settings.loopbackFilename,
		H264Decoder:      settings.h264Decoder,
		H264Encoder:      settings.h264Encoder,
		MinVideoBitrate:  settings.minVideoBitrate,
		MultiStream:      settings.multiStream,
	}

	ffmpeg := hkcam.SetupFFMPEGStreaming(cam, cfg)

	// Add a custom camera control service to record snapshots
	cc := hkcam.NewCameraControl()
	cam.Control.AddCharacteristic(cc.Assets.Characteristic)
	cam.Control.AddCharacteristic(cc.GetAsset.Characteristic)
	cam.Control.AddCharacteristic(cc.DeleteAssets.Characteristic)
	cam.Control.AddCharacteristic(cc.TakeSnapshot.Characteristic)
	// End Camera setup

	// Start the hk brige ip transport
	config := hc.Config{Pin: "80000000", StoragePath: settings.storagePath}
	t, err := hc.NewIPTransport(config, th.Accessory, lockButton.Accessory, ecoModeButton.Accessory, onButton.Accessory, cam.Accessory)
	if err != nil {
		log.Error(err)
	}

	// Set up camera snapshots
	t.CameraSnapshotReq = func(width, height uint) (*image.Image, error) {
		return ffmpeg.Snapshot(width, height)
	}

	cc.SetupWithDir(settings.storagePath)
	cc.CameraSnapshotReq = func(width, height uint) (*image.Image, error) {
		return ffmpeg.Snapshot(width, height)
	}

	go func() {
		// Fridge state scanner
		// TODO bring this value in from env/cli
		hkUpdateInterval := time.Second
		ticker := time.NewTicker(hkUpdateInterval)

		log.Trace("HK client looping now")
		for {
			select {
			case <-ctx.Done():
				log.Trace("HKClient ctx canceled")
				<-t.Stop()
				log.Trace("HKClient stopped")
				return
			case <-ticker.C:
				s := fridge.GetStatusReport()
				var t float64
				var tempSetting float64
				if s.CelsiusFahrenheitModeMenuE5 {
					t = FtoC(float64(s.Temp))
					tempSetting = FtoC(float64(s.TempSet))
				} else {
					t = float64(s.Temp)
					tempSetting = float64(s.TempSet)
				}
				log.WithFields(log.Fields{
					"client":      "HKClient",
					"temp":        t,
					"tempSetting": tempSetting,
				}).Trace("settings to HK")

				// switches/buttons
				onButton.Switch.On.SetValue(s.On)
				ecoModeButton.Switch.On.SetValue(s.EcoMode)
				lockButton.Switch.On.SetValue(s.Locked)
				// Required
				if s.On {
					th.Thermostat.CurrentHeatingCoolingState.SetValue(2)
					th.Thermostat.TargetHeatingCoolingState.SetValue(2)
				} else {
					th.Thermostat.CurrentHeatingCoolingState.SetValue(0)
					th.Thermostat.TargetHeatingCoolingState.SetValue(0)
				}
				th.Thermostat.CurrentTemperature.SetValue(t)
				th.Thermostat.TargetTemperature.SetValue(tempSetting)
				th.Thermostat.TemperatureDisplayUnits.SetValue(1) // 0=C, 1=F

				// TODO see if this is settable this often per the spec
				// th.Thermostat.TemperatureDisplayUnits.SetValue(int(s.CelsiusFahrenheitModeMenuE5)) // 0=C, 1=F

				// Optional
				// th.Thermostat.CurrentHeatingCoolingState.SetMaxValue(int(s.LowestTempSettingMenuE1))
				// th.Thermostat.CurrentHeatingCoolingState.SetMinValue(int(s.HighestTempSettingMenuE2))
			}
		}
	}()

	// Start homekit transport
	t.Start()
}

// FtoC converts
func FtoC(f float64) float64 {
	return (f - 32) * 5 / 9
}

// CtoF converts
func CtoF(f float64) float64 {
	return f*9/5 + 32
}
