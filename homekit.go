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

	info := accessory.Info{
		Name:             "K25",
		SerialNumber:     "1",
		Manufacturer:     "johnelliott.org",
		Model:            "WT-0001 Bridge",
		FirmwareRevision: "v0.0.1",
		ID:               1,
	}
	ac := accessory.NewTemperatureSensor(info, 10, -20, 30, 1)
	config := hc.Config{Pin: "80000000"}
	t, err := hc.NewIPTransport(config, ac.Accessory)
	if err != nil {
		log.Error(err)
	}

	hc.OnTermination(func() {
		wg.Add(1)
		defer func() {
			log.Trace("HK client bridge termination wg done")
			wg.Done()
		}()
		<-t.Stop()
	})

	go func() {
		wg.Add(1)
		defer func() {
			log.Trace("HK client loop calling done on main wait group")
			wg.Done()
		}()
		log.Trace("HK client looping now")
		for {
			select {
			case <-ctx.Done():
				log.Trace("HKClient ctx canceled")
				return
			case s := <-fridgeStatus:
				log.Trace("Setting homekit TempSensor value", s.Temp)
				var t float64
				if s.E5 == 1 {
					t = (float64(s.Temp) - 32) * 5 / 9
				} else {
					t = float64(s.Temp)
				}
				// if s.On == 1 {
				// 	ac.Thermostat.CurrentHeatingCoolingState.SetValue(3)
				// } else {
				// 	ac.Thermostat.CurrentHeatingCoolingState.SetValue(0)
				// }
				ac.TempSensor.CurrentTemperature.SetValue(t)
				// ac.Thermostat.TargetTemperature.SetValue(float64(s.TempSet))
				// ac.Thermostat.CurrentHeatingCoolingState.SetMaxValue(int(s.E1))
				// ac.Thermostat.CurrentHeatingCoolingState.SetMinValue(int(s.E2))
				// ac.Thermostat.TemperatureDisplayUnits.SetValue(int(s.E5)) // 0=C, 1=F
			}
		}
	}()
	// wait for and trash one value
	<-fridgeStatus
	// Start homekit transport
	t.Start()
}
