package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-acme/lego/platform/config/env"
	log "github.com/sirupsen/logrus"
)

var (
	// Flags
	adapterNameF = flag.String("adapter", zeroAdapter, "adapter name, e.g. hci0")
	addrF        = flag.String("fridgeaddr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	storagePathF = flag.String("fridgestoragepath", "./var/local/homekitdb", "path for sqlite storage of homekit data")
	timeoutF     = flag.Duration("timeout", 20*time.Minute, "overall program timeout")
	pollrateF    = flag.Duration("pollrate", 1*time.Second, "magic payload polling rate")

	initialFridgeSettings = Settings{}

	// App settings
	pollrate    time.Duration
	timeout     time.Duration
	addr        string
	adapterName string
)

type statusReportC chan StatusReport
type tempSettingsC chan float64
type settingsC chan Settings

// Fridge represents a full fridge state
type Fridge struct {
	mu            sync.RWMutex
	status        StatusReport
	inlet         statusReportC
	tempSettingsC tempSettingsC
	settingsC     settingsC
}

// MonitorMu routine, mutex based
func (f *Fridge) MonitorMu() {
	// TODO add canceling
	for r := range f.inlet {
		log.Trace("Fridge got status update", r.Temp)
		f.mu.Lock()
		f.status = r
		f.mu.Unlock()
	}
}

// SetOn Sends the fridge state to the fridge
func (f *Fridge) SetOn(turnOn bool) {
	log.Warnf("SetOn: %v", turnOn)
	s := f.GetStatusReport().Settings
	if turnOn {
		s.On = 1
	} else {
		s.On = 0
	}
	f.settingsC <- s
}

// SetEcoMode Sends the fridge state to the fridge
func (f *Fridge) SetEcoMode(useEcoMode bool) {
	log.Warnf("SetEcoMode: %v", useEcoMode)
	s := f.GetStatusReport().Settings
	if useEcoMode {
		s.EcoMode = 1
	} else {
		s.EcoMode = 0
	}
	f.settingsC <- s
}

// SetLocked Sends the fridge state to the fridge
func (f *Fridge) SetLocked(lockIt bool) {
	log.Warnf("SetLocked: %v", lockIt)
	s := f.GetStatusReport().Settings
	if lockIt {
		s.Locked = 1
	} else {
		s.Locked = 0
	}
	f.settingsC <- s
}

// GetStatusReport gets the fridge state
func (f *Fridge) GetStatusReport() StatusReport {
	f.mu.RLock()
	defer f.mu.RUnlock()
	log.Trace("getting status report", f.status.Temp)
	return f.status
}

func (f *Fridge) CycleCompressor(onTime time.Duration) {
	log.Info("Fridge quick compressor cycle")
	// Capture settings
	s := f.GetStatusReport().Settings
	// wait if we see that the struct is just initialized
	// TODO do this better, this is a lazy way
	if s == initialFridgeSettings {
		log.Trace("Waiting to see some initialized data")
		time.Sleep(2 * time.Second)
		f.CycleCompressor(onTime)
		return
	}
	prevSettings := s
	// Turn down temp
	if s.On != 1 {
		// Turn on
		s.On = 1
		// Choose freezing
		if s.E5 != 0 {
			s.TempSet = 0xff - 10 // minus 10 c
		} else {
			s.TempSet = 0 // TODO fix this to C or f
		}
		log.Tracef("Fridge going to cold setting: On=%v TempSet=%v", s.On, s.TempSet)
		// block writing while we're cycling
		f.settingsC <- s
		// TODO see if there's a way to avoid this 30s window where things could get clobbered
		// time after func turn off
		time.AfterFunc(onTime, func() {
			log.Tracef("Fridge going back to prev settings: %v", prevSettings.TempSet)
			f.settingsC <- prevSettings
		})
	}
}

func main() {
	flag.Parse()
	log.Warn("timeout", timeout)
	log.Warn("pollrate", pollrate)

	// Use env to override app settings
	timeout = env.GetOrDefaultSecond("TIMEOUT_SEC", *timeoutF)
	pollrate = env.GetOrDefaultSecond("POLLRATE_SEC", *pollrateF)
	adapterName = env.GetOrDefaultString("ADAPTER_NAME", *adapterNameF)
	addr = env.GetOrDefaultString("FRIDGE_ADDR", *addrF)
	storagePath := env.GetOrDefaultString("STORAGE_PATH", *storagePathF)

	log.Warn("timeout", timeout)
	log.Warn("pollrate", pollrate)

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
		log.SetLevel(log.TraceLevel)
	}

	log.SetFormatter(&log.JSONFormatter{})

	// main context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Subtask quit response channels
	wg := sync.WaitGroup{}

	// Subtask contexts
	clientContext, cancelClient := context.WithCancel(ctx)
	defer cancelClient()

	// fakeClientContext, cancelFakeClientContext := context.WithCancel(ctx)
	// defer cancelFakeClientContext()

	HKClientContext, cancelHKClientContext := context.WithCancel(ctx)
	defer cancelHKClientContext()

	// Data setup
	fridge := Fridge{
		inlet:         make(statusReportC),
		tempSettingsC: make(tempSettingsC),
		settingsC:     make(settingsC),
	}
	// Collect updates into status
	go func() { fridge.MonitorMu() }()

	// Listen for control-c subtask
	go func() {
		// https://rafallorenz.com/go/handle-signals-to-graceful-shutdown-http-server/
		// Set up channel on which to send signal notifications.
		// We must use a buffered channel or risk missing the signal
		// if we're not ready to receive when addthe signal is sent.
		sig := make(chan os.Signal, 1)
		signal.Notify(
			sig,
			syscall.SIGTERM,
			syscall.SIGHUP,  // kill -SIGHUP XXXX
			syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
			syscall.SIGQUIT, // kill -SIGQUIT XXXX
		)
		log.Trace("Listening for signals")
		s := <-sig
		log.Debug("Got signal:", s)
		cancel()
	}()

	go func() {
		log.Debug("Fridge interval turnon/turnoff start")
		// cycle on startup of daemon
		go fridge.CycleCompressor(15 * time.Second)
		ticker := time.NewTicker(8 * time.Hour)
		for range ticker.C {
			log.Debug("Fridge compressor cycle tick")
			go fridge.CycleCompressor(15 * time.Second)
		}
	}()

	// Kick off bluetooth client
	go func() {
		log.Debug("Launching client")
		err := Client(clientContext, &wg, &fridge, adapterName, addr)
		if err == context.Canceled || err == context.DeadlineExceeded {
			log.Debug("Client: ", err)
		} else if err != nil {
			log.Error(err)
		}
		log.Debug("Client done")
		cancel() // M
	}()

	// Kick off homekit client
	go HKClient(HKClientContext, &wg, storagePath, &fridge)

	// fakeResultsC := make(chan int)
	// go FakeClient(fakeClientContext, &wg, fakeResultsC)

	log.Trace("Main waiting...")
	for {
		select {
		// case r := <-fakeResultsC:
		// 	log.Infof("FakeClient result: %v\n", r)
		case <-ctx.Done():
			log.Debug("Main context canceled")

			// bail hard if this takes too long
			go func() {
				theFinalCountdown := 30 * time.Second
				log.Debugf("Waiting %v then exiting", theFinalCountdown)
				time.AfterFunc(theFinalCountdown, func() {
					panic("Took too long to exit\n")
				})
			}()

			log.Trace("Waiting for wait group...")
			// Clean up others
			wg.Wait()
			log.Trace("Wait group done waiting")
			return
		}
	}
}
