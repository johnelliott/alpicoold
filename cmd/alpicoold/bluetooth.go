package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/johnelliott/alpicoold/pkg/k25"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	log "github.com/sirupsen/logrus"
)

var (
	// Pi stuff
	zeroAdapter = "hci0"

	// Characteristics
	serviceUUID         = "00001234-0000-1000-8000-00805f9b34fb"
	writeableFridgeUUID = "00001235-0000-1000-8000-00805f9b34fb" // Writable
	readeableFridgeUUID = "00001236-0000-1000-8000-00805f9b34fb" // Read Notify
	descriptorUUID      = "00002902-0000-1000-8000-00805f9b34fb"
)

// Client is the main bluetooth client that looks at the fridge
func Client(ctx context.Context, wg *sync.WaitGroup, fridge *Fridge, adapterID, hwaddr string) error {
	log := log.WithFields(log.Fields{
		"client": "BluetoothClient",
	})
	wg.Add(1)
	defer func() {
		wg.Done()
		log.Trace("Calling done on main wait group")
	}()

	// clean up connection on exit
	defer func() {
		api.Exit()
		log.Trace("Api exit done")
	}()

	log.Infof("Discovering %s on %s", hwaddr, adapterID)

	a, err := adapter.NewAdapter1FromAdapterID(adapterID)
	if err != nil {
		return err
	}

	//Connect DBus System bus
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	// do not reuse agent0 from service
	agent.NextAgentPath()

	ag := agent.NewSimpleAgent()
	err = agent.ExposeAgent(conn, ag, agent.CapNoInputNoOutput, true)
	if err != nil {
		return fmt.Errorf("SimpleAgent: %s", err)
	}

	findContext, cancelFindDevice := context.WithCancel(ctx)
	defer cancelFindDevice()
	dev, err := findDevice(findContext, a, hwaddr)
	if err != context.Canceled && err != nil {
		return fmt.Errorf("findDevice: %s", err)
	}

	// Connect to the device
	err = connect(dev, ag, adapterID)
	if err != nil {
		return err
	}

	// Kick off listening for commands

	// Kick off listening for state notifications
	watchStateCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	err = WatchState(watchStateCtx, fridge, a, dev)
	if err != nil {
		return err
	}

	log.Trace("Client blocking and waiting")
	// Wait for quit signal
	select {
	case <-ctx.Done():
		log.Tracef("Cancel: bluetooth client: %v", ctx.Err())
		// wait for cycling to be done
		log.Trace("await compressor cycle")
		fridge.cycleCompressorWg.Wait()
		log.Trace("compressor cycle done, disconnecting bluetooth")
		err := dev.Disconnect()
		if err != nil {
			log.Error(err)
			return err
		}
		log.Trace("Disconnected from bluetooth")

		return nil
	}
}

func findDevice(ctx context.Context, a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {
	devices, err := a.GetDevices()
	if err != nil {
		return nil, err
	}

	for _, dev := range devices {
		devProps, err := dev.GetProperties()
		if err != nil {
			log.Errorf("Failed to load dev props: %s", err)
			continue
		}

		log.Info(devProps.Address)
		if devProps.Address != hwaddr {
			continue
		}

		log.Infof("Found cached device Connected=%t Trusted=%t Paired=%t", devProps.Connected, devProps.Trusted, devProps.Paired)
		return dev, nil
	}

	// Start discovery if we don't see ours
	discoverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	dev, err := discover(discoverCtx, a, hwaddr)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, errors.New("Device not found, is it advertising?")
	}
	log.Debug("Found device")

	return dev, nil
}

func discover(ctx context.Context, a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {

	err := a.FlushDevices()
	if err != nil {
		return nil, err
	}

	dFilter := adapter.NewDiscoveryFilter()
	dFilter.AddUUIDs(serviceUUID)
	dFilter.Transport = "le"
	a.SetDiscoveryFilter(dFilter.ToMap())

	discovery, cancelDiscovery, err := api.Discover(a, nil)
	defer cancelDiscovery()
	if err != nil {
		return nil, err
	}

	for {
		select {
		case ev := <-discovery:
			dev, err := device.NewDevice1(ev.Path)
			if err != nil {
				return nil, err
			}

			if dev == nil || dev.Properties == nil {
				continue
			}

			p := dev.Properties

			// n := p.Alias
			// if p.Name != "" {
			// 	n = p.Name
			// }
			// log.Tracef("Discovered (%s) %s", n, p.Address)

			if p.Address != hwaddr {
				// log.Trace("Found the one we want", p.Address)
				continue
			}

			return dev, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func connect(dev *device.Device1, ag *agent.SimpleAgent, adapterID string) error {

	props, err := dev.GetProperties()
	if err != nil {
		return fmt.Errorf("Failed to load props: %s", err)
	}

	log.Debugf("Found device name=%s addr=%s rssi=%d", props.Name, props.Address, props.RSSI)

	if props.Connected {
		log.Info("Device is connected")
		return nil
	}

	// My wt-0001 fridge doesn't need to pair or trust because it's a stupid device

	if !props.Connected {
		log.Trace("Connecting device")
		err = dev.Connect()
		if err != nil {
			// check if error string has specific info
			if !strings.Contains(err.Error(), "Connection refused") {
				return fmt.Errorf("Connect refused: %s", err)
			}
			return fmt.Errorf("Connect failed: %s", err)
		}
		log.Trace("Connected to device")
	}

	return nil
}

// TODO make this function take a generic thing?
// or maybe not because we need to send the version number to get notifications

// Watchstate is what we came to do
func WatchState(ctx context.Context, fridge *Fridge, a *adapter.Adapter1, dev *device.Device1) error {
	log.Trace("watchState running")

	list, err := dev.GetCharacteristics()
	if err != nil {
		return err
	}

	// Retry
	if len(list) == 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
		time.Sleep(2 * time.Second)
		return WatchState(ctx, fridge, a, dev)
	}
	log.Debugf("Found %d characteristics", len(list))

	// TODO actually find the right one in the list and sleep/wait for it
	// e.g. /org/bluez/hci0/dev_D8_17_D1_F1_B9_78/service0004/char0005
	char, err := dev.GetCharByUUID(writeableFridgeUUID)
	if err != nil {
		return err
	}
	log.Debugf("Found writable UUID: %v", char.Properties.UUID)

	// Make a little cancelable pagic payloader
	go func() {
		log.Trace("BT attribute writer starting")
		// Set up a timer to send the stupid notification payload
		ticker := time.NewTicker(pollrate)
		defer ticker.Stop()

		for {
			select {
			case settings := <-fridge.settingsC:
				log.Tracef("Got settings payload %v", settings)
				c, err := k25.NewSetStateCommand(settings)
				if err != nil {
					panic(err)
				}
				log.WithFields(log.Fields{
					"client":  "BluetoothClient",
					"payload": fmt.Sprintf("% x", c),
				}).Infof("Writing set state payload")
				err = char.WriteValue(c, nil)
				if err != nil {
					panic(err)
				}
			case temp := <-fridge.tempSettingsC:
				log.WithFields(log.Fields{
					"temp":   temp,
					"client": "BluetoothClient",
				}).Info("Got celsius temp setting!")
				sr := fridge.GetStatusReport()
				maxTemp := float64(sr.HighestTempSettingMenuE2)
				log.Trace("max temp", maxTemp)
				minTemp := float64(sr.LowestTempSettingMenuE1)
				log.Trace("min temp", minTemp)

				// Use current units
				isF := sr.CelsiusFahrenheitModeMenuE5
				if isF {
					temp = CtoF(temp)
				}
				log.Debug("converted temp", temp)
				// Sanitize hk possible high input
				temp = math.Min(maxTemp, temp)
				log.Debug("capped temp", temp)

				temp = math.Max(minTemp, temp)
				log.Debug("upped temp", temp)
				// Move result back to C if we're in C mode
				if !isF {
					temp = FtoC(temp)
				}

				// Form command bytes
				c, err := k25.NewSetTempCommand(int8(temp))
				if err != nil {
					panic(err)
				}
				log.Info("Writing set temp payload", c)
				err = char.WriteValue(c, nil)
				if err != nil {
					panic(err)
				}
			case <-ticker.C:
				log.Trace("Writing magic payload", k25.PingCommand)
				err = char.WriteValue(k25.PingCommand, nil)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	notifChar, err := dev.GetCharByUUID(readeableFridgeUUID)
	if err != nil {
		return err
	}

	// e.g. https://git.tcp.direct/kayos/prototooth/src/release/gattc_linux.go#L223
	propsC, err := notifChar.WatchProperties()
	if err != nil {
		return err
	}
	go func() {
		log.Trace("state updater starting")
		var f k25.StatusReport
		for update := range propsC {
			log.WithFields(log.Fields{
				"name":      update.Name,
				"interface": update.Interface,
				"value":     update.Value,
			}).Tracef("state update")
			if update.Interface == "org.bluez.GattCharacteristic1" && update.Name == "Value" {
				value := update.Value.([]byte)
				err = f.UnmarshalBinary(value)
				if err != nil {
					log.Error("Other frame UnmarshalBinary", err)
					break
				}
				// Send status to rest of app
				fridge.inlet <- f
			}
		}
	}()

	err = notifChar.StartNotify()
	if err != nil {
		return err
	}

	log.Trace("watchState returning now")
	return nil
}
