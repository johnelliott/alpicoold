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

	// App settings
	pollrate    time.Duration
	timeout     time.Duration
	addr        string
	adapterName string
)

type tempSettingsC chan float64
type settingsC chan SetStateCommand

// Fridge represents a full fridge state
type Fridge struct {
	mu            sync.RWMutex
	status        StatusReport
	inlet         chan StatusReport
	tempSettingsC tempSettingsC
	settingsC     settingsC
}

// MonitorMu routine, mutex based
func (f *Fridge) MonitorMu() {
	// TODO add canceling
	for r := range f.inlet {
		log.Debug("Fridge got status update", r.Temp)
		f.mu.Lock()
		f.status = r
		f.mu.Unlock()
	}
}

// SendSettings Sends the fridge state to the fridge
func (f *Fridge) SendSettings(r Settings) {
	log.Warnf("Fridge SendSettings stub: %v", r)

}

// GetStatusReport gets the fridge state
func (f *Fridge) GetStatusReport() StatusReport {
	f.mu.RLock()
	defer f.mu.RUnlock()
	log.Debug("getting status report", f.status.Temp)
	return f.status
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
		inlet:         make(chan StatusReport),
		tempSettingsC: make(tempSettingsC),
		settingsC:     make(chan SetStateCommand),
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

			log.Debug("Waiting for wait group...")
			// Clean up others
			wg.Wait()
			log.Trace("Wait group done waiting")
			return
		}
	}
}
