package main

import (
	"context"
	"flag"
	"image"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	hclog "github.com/brutella/hc/log"
	"github.com/brutella/hkcam"
	"github.com/brutella/hkcam/ffmpeg"
	log "github.com/sirupsen/logrus"
)

// HKClient is an imaginary client for homekit preparation
func HKClient(ctx context.Context, wg *sync.WaitGroup, storagePath string, minVideoBitrate int, multiStream bool, fridge *Fridge) {
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

	th.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(newTempRawCelcius float64) {
		// Round to something reasonable
		newTemp := math.Round(newTempRawCelcius)
		log.Tracef("New TargetTemperature: %v %v %v", newTempRawCelcius, newTemp, byte(newTemp))
		fridge.tempSettingsC <- newTemp
		// just set it for them for now, do this via commands later
		// th.Thermostat.TargetTemperature.SetValue(newTemp)
	})
	// Camera setup

	// Platform dependent flags
	var inputDevice *string
	var inputFilename *string
	var loopbackFilename *string
	var h264Encoder *string
	var h264Decoder *string

	if runtime.GOOS == "linux" {
		inputDevice = flag.String("input_device", "v4l2", "video input device")
		inputFilename = flag.String("input_filename", "/dev/video0", "video input device filename")
		loopbackFilename = flag.String("loopback_filename", "/dev/video1", "video loopback device filename")
		h264Decoder = flag.String("h264_decoder", "", "h264 video decoder")
		h264Encoder = flag.String("h264_encoder", "h264_omx", "h264 video encoder")
	} else if runtime.GOOS == "darwin" { // macOS
		inputDevice = flag.String("input_device", "avfoundation", "video input device")
		inputFilename = flag.String("input_filename", "default", "video input device filename")
		// loopback is not needed on macOS because avfoundation provides multi-access to the camera
		loopbackFilename = flag.String("loopback_filename", "", "video loopback device filename")
		h264Decoder = flag.String("h264_decoder", "", "h264 video decoder")
		h264Encoder = flag.String("h264_encoder", "libx264", "h264 video encoder")
	} else {
		log.Fatalf("%s platform is not supported", runtime.GOOS)
	}

	if log.GetLevel() == log.TraceLevel {
		// TOOD get something like this log.Debug.Enable()
		ffmpeg.EnableVerboseLogging()
	}

	switchInfo := accessory.Info{Name: "Camera", FirmwareRevision: "0.0.9", Manufacturer: "Matthias Hochgatterer"}
	cam := accessory.NewCamera(switchInfo)

	cfg := ffmpeg.Config{
		InputDevice:      *inputDevice,
		InputFilename:    *inputFilename,
		LoopbackFilename: *loopbackFilename,
		H264Decoder:      *h264Decoder,
		H264Encoder:      *h264Encoder,
		MinVideoBitrate:  minVideoBitrate,
		MultiStream:      multiStream,
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
	config := hc.Config{Pin: "80000000", StoragePath: storagePath}
	t, err := hc.NewIPTransport(config, th.Accessory, lockButton.Accessory, ecoModeButton.Accessory, onButton.Accessory, cam.Accessory)
	if err != nil {
		log.Error(err)
	}

	// Set up camera snapshots
	t.CameraSnapshotReq = func(width, height uint) (*image.Image, error) {
		return ffmpeg.Snapshot(width, height)
	}

	cc.SetupWithDir(storagePath)
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
				log.Tracef("Homekit got fridge status %v", s.Temp)
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
				ecoModeButton.Switch.On.SetValue(s.EcoMode == 1)
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
				th.Thermostat.TemperatureDisplayUnits.SetValue(1) // 0=C, 1=F

				// TODO see if this is settable this often per the spec
				// th.Thermostat.TemperatureDisplayUnits.SetValue(int(s.E5)) // 0=C, 1=F

				// Optional
				// th.Thermostat.CurrentHeatingCoolingState.SetMaxValue(int(s.E1))
				// th.Thermostat.CurrentHeatingCoolingState.SetMinValue(int(s.E2))
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
