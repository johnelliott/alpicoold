package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/agent"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	log "github.com/sirupsen/logrus"
)

// Pi stuff
var zeroAdapter = "hci0"

// Characteristics
var writeableFridgeUUID = "00001235-0000-1000-8000-00805f9b34fb" // Writable
var readeableFridgeUUID = "00001236-0000-1000-8000-00805f9b34fb" // Read Notify
var descriptorUUID = "00002902-0000-1000-8000-00805f9b34fb"

// Commands
var magicPayload = []byte{0xfe, 0xfe, 0x3, 0x1, 0x2, 0x0}

// var maybeUnlockbytes = []byte{0xfe, 0xfe, 0x11, 0x2, 0x1, 0x0, 0x1, 0x0, 0x24, 0x44, 0xfc, 0x4, 0x0, 0x1, 0x0, 0x0, 0xfb, 0x0, 0x4, 0x75}
// TODO add a factory reset thing gleaned from wireshark

var (
	adapterName = flag.String("adapter", zeroAdapter, "adapter name, e.g. hci0")
	addr        = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	timeout     = flag.Duration("timeout", 20*time.Second, "overall program timeout")
)

func main() {
	flag.Parse()

	// env vars
	LOGLEVEL := os.Getenv("LOGLEVEL")
	switch LOGLEVEL {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.DebugLevel)
	}

	// https://rafallorenz.com/go/handle-signals-to-graceful-shutdown-http-server/

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when addthe signal is sent.
	sig := make(chan os.Signal, 1)
	signal.Notify(
		sig,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	// main context
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Listen for control-c
	go func() {
		s := <-sig
		log.Debug("Got signal:", s)
		cancel()
	}()

	// Do work for program
	clientContext, cancel := context.WithCancel(ctx)
	defer cancel()
	err := client(clientContext, *adapterName, *addr)
	if err != nil {
		panic(err)
	}
}

func client(ctx context.Context, adapterID, hwaddr string) error {
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

	dev, err := findDevice(a, hwaddr)
	if err != nil {
		return fmt.Errorf("findDevice: %s", err)
	}

	/*
		watchProps, err := dev.WatchProperties()
		if err != nil {
			return err
		}
		go func() {
			for propUpdate := range watchProps {
				log.Tracef("--> device updated %s=%v", propUpdate.Name, propUpdate.Value)
			}
		}()
	*/

	err = connect(dev, ag, adapterID)
	if err != nil {
		return err
	}

	// Kick off listening for state notifications
	watchStateCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	err = watchState(watchStateCtx, a, dev)
	if err != nil {
		return err
	}

	log.Trace("Client blocking and waiting")
	// Wait for quit signal
	select {
	case <-ctx.Done():
		log.Error("Cancel client:", ctx.Err())
		log.Trace("Disconnecting from bluetooth...")
		err := dev.Disconnect()
		if err != nil {
			log.Error(err)
			panic(err)
		}
		log.Trace("Disconnected from bluetooth")
		return nil
	}
}

func findDevice(a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {

	dev, err := discover(a, hwaddr)
	if err != nil {
		return nil, err
	}
	if dev == nil {
		return nil, errors.New("Device not found, is it advertising?")
	}

	return dev, nil
}

func discover(a *adapter.Adapter1, hwaddr string) (*device.Device1, error) {

	err := a.FlushDevices()
	if err != nil {
		return nil, err
	}

	// TODO use a discovery filter in here
	discovery, cancel, err := api.Discover(a, nil)
	defer cancel()
	if err != nil {
		return nil, err
	}

	for ev := range discovery {

		dev, err := device.NewDevice1(ev.Path)
		if err != nil {
			return nil, err
		}

		if dev == nil || dev.Properties == nil {
			continue
		}

		p := dev.Properties

		n := p.Alias
		if p.Name != "" {
			n = p.Name
		}
		log.Debugf("Discovered (%s) %s", n, p.Address)

		if p.Address != hwaddr {
			continue
		}

		return dev, nil
	}

	return nil, nil
}

func connect(dev *device.Device1, ag *agent.SimpleAgent, adapterID string) error {

	props, err := dev.GetProperties()
	if err != nil {
		return fmt.Errorf("Failed to load props: %s", err)
	}

	log.Debugf("Found device name=%s addr=%s rssi=%d", props.Name, props.Address, props.RSSI)

	if props.Connected {
		log.Trace("Device is connected")
		return nil
	}

	// My wt-0001 fridge doesn't need to pair or trust because it's a stupid device

	if !props.Connected {
		log.Trace("Connecting device")
		err = dev.Connect()
		if err != nil {
			if !strings.Contains(err.Error(), "Connection refused") {
				return fmt.Errorf("Connect failed: %s", err)
			}
		}
	}

	return nil
}

// TODO make this function take a generic thing?
// or maybe not because we need to send the version number to get notifications
func watchState(ctx context.Context, a *adapter.Adapter1, dev *device.Device1) error {
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
		return watchState(ctx, a, dev)
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
	writexCtx, cancel := context.WithCancel(ctx)
	go func(writexCtx context.Context) {
		defer cancel()
		log.Trace("magic payload writer starting")
		// Set up a timer to send the stupid notification payload
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Error("Cancel: magic payload loop", ctx.Err())
				return
			case <-ticker.C:
				log.Trace("Writing magic payload", magicPayload)
				err = char.WriteValue(magicPayload, nil)
				if err != nil {
					panic(err)
				}
			}
		}
	}(writexCtx)

	notifChar, err := dev.GetCharByUUID(readeableFridgeUUID)
	if err != nil {
		return err
	}

	// e.g. https://git.tcp.direct/kayos/prototooth/src/release/gattc_linux.go#L223
	propsC, err := notifChar.WatchProperties()
	if err != nil {
		return err
	}
	stateUpdaterCtx, cancel := context.WithCancel(ctx)
	go func(ctx context.Context) {
		defer cancel()
		log.Trace("state updater starting")
		var f Frame
		for {
			select {
			case <-ctx.Done():
				log.Error("Cancel: fridge state loop", ctx.Err())
				return
			case update := <-propsC:
				log.Tracef("--> update name=%s int=%s val=%v", update.Name, update.Interface, update.Value)
				if update.Interface == "org.bluez.GattCharacteristic1" && update.Name == "Value" {
					value := update.Value.([]byte)
					err = f.UnmarshalBinary(value)
					if err != nil {
						log.Error("other frame UnmarshalBinary", err)
					}
					log.Debugf("f %s", f)
				}
			}
		}
	}(stateUpdaterCtx)

	err = notifChar.StartNotify()
	if err != nil {
		return err
	}

	// so maybe nothing here cancels, it just clones context and does the watchig and writing
	// <-ctx.Done()
	log.Trace("watchState returning now")
	return nil
}

/*
// TODO state setting
// This is probably full state set
data, err := hex.DecodeString(maybeUnlock)
if err != nil {
	return err
}
// log.Trace("Sending unlock", data)
char.WriteValue(data, nil)
*/
