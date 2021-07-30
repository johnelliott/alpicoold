package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-acme/lego/platform/config/env"
	"github.com/johnelliott/alpicoold/pkg/k25"
	log "github.com/sirupsen/logrus"
)

var (
	// Flags
	adapterNameF = flag.String("adapter", zeroAdapter, "adapter name, e.g. hci0")
	addrF        = flag.String("fridgeaddr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	timeoutF     = flag.Duration("timeout", 20*time.Minute, "overall program timeout")
	pollrateF    = flag.Duration("pollrate", 1*time.Second, "magic payload polling rate")

	// HomeKit
	storagePathF = flag.String("fridgestoragepath", "./var/local/homekitdb", "path for sqlite storage of homekit data")

	// Camera
	minVideoBitrateF    = flag.Int("min_video_bitrate", 0, "minimum video bit rate in kbps")
	camRotationDegreesF = flag.Int("cam_rot_deg", 0, "raspi camera rotation in degrees")
	multiStreamF        = flag.Bool("multi_stream", false, "Allow mutliple clients to view the stream simultaneously")
	inputDeviceF        = flag.String("input_device", "v4l2", "video input device")
	inputFilenameF      = flag.String("input_filename", "/dev/video0", "video input device filename")
	loopbackFilenameF   = flag.String("loopback_filename", "/dev/video1", "video loopback device filename")
	h264DecoderF        = flag.String("h264_decoder", "", "h264 video decoder")
	h264EncoderF        = flag.String("h264_encoder", "h264_omx", "h264 video encoder")

	initialFridgeSettings = k25.Settings{}

	// App settings
	// TODO JSON log setting and control that below
	pollrate           time.Duration
	minVideoBitrate    int
	camRotationDegrees int
	multiStream        bool
	timeout            time.Duration
	addr               string
	adapterName        string
	inputDevice        string
	inputFilename      string
	loopbackFilename   string
	h264Decoder        string
	h264Encoder        string
)

//var dataDir *string = flag.String("data_dir", "Camera", "Path to data directory")
// var verbose *bool = flag.Bool("verbose", true, "Verbose logging")
// var pin *string = flag.String("pin", "00102003", "PIN for HomeKit pairing")
// var port *string = flag.String("port", "", "Port on which transport is reachable")

type statusReportC chan k25.StatusReport
type tempSettingsC chan float64
type settingsC chan k25.Settings

// Fridge represents a full fridge state
type Fridge struct {
	mu            sync.RWMutex
	status        k25.StatusReport
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
		prev := f.status
		f.status = r
		f.mu.Unlock()
		// Log if on state changed
		sr := f.GetStatusReport()
		if prev.On != sr.On {
			f.Log().Warn("on state changed")
		}
	}
}

// Log some basic stats to the console
func (f *Fridge) Log() *log.Entry {
	r := f.GetStatusReport()

	voltageStr := fmt.Sprintf("%d.%dv", r.InputV1, r.InputV2)
	return log.WithFields(log.Fields{
		"eco":      r.EcoMode,
		"input":    voltageStr,
		"lck":      r.Locked,
		"on":       r.On,
		"set-temp": r.TempSet,
		"temp":     r.Temp,
	})
}

// SetOn Sends the fridge state to the fridge
func (f *Fridge) SetOn(turnOn bool) {
	log.Warnf("SetOn: %v", turnOn)
	s := f.GetStatusReport().Settings
	if s.On != turnOn {
		s.On = turnOn
		f.settingsC <- s
	}
}

// SetEcoMode Sends the fridge state to the fridge
func (f *Fridge) SetEcoMode(useEcoMode bool) {
	log.Warnf("SetEcoMode: %v", useEcoMode)
	s := f.GetStatusReport().Settings
	if s.EcoMode != useEcoMode {
		s.EcoMode = useEcoMode
		f.settingsC <- s
	}
}

// SetLocked Sends the fridge state to the fridge
func (f *Fridge) SetLocked(lockIt bool) {
	log.Warnf("SetLocked: %v", lockIt)
	s := f.GetStatusReport().Settings
	if s.Locked != lockIt {
		s.Locked = lockIt
		f.settingsC <- s
	}
}

// GetStatusReport gets the fridge state
func (f *Fridge) GetStatusReport() k25.StatusReport {
	f.mu.RLock()
	defer f.mu.RUnlock()
	log.Trace("getting status report", f.status.Temp)
	return f.status
}

// CycleCompressor spins up compressor to defeat power bank auto-off
func (f *Fridge) CycleCompressor(ctx context.Context, onTime time.Duration) {
	log.Info("Fridge quick compressor cycle")
	// Capture settings
	// wait if we see that the struct is just initialized
	s := f.GetStatusReport()
	// TODO do this better, this is a lazy way
	ticker := time.NewTicker(2 * time.Second)
Lerp:
	for {
		select {
		case <-ticker.C:
			s = f.GetStatusReport()
			if s.Settings != initialFridgeSettings {
				break Lerp
			} else {
				log.Trace("Waiting to see fridge initialized data")
			}
		case <-ctx.Done():
			return
		}
	}

	if s.InputV1 > 13 {
		// Voltage is high enough that we're not on a 12v regulated battery
		log.Info("Fridge input voltage over 13v; skipping compressor cycle")
		return
	}

	prevSettings := s.Settings
	f.Log().Trace("Cycling compressor...")
	// Turn down temp
	if !s.On {
		// Turn on
		s.On = true

		// Choose temp to set
		current := float64(s.Temp)
		upperBound := float64(s.LowestTempSettingMenuE1)
		lowerBound := float64(s.HighestTempSettingMenuE2)
		hysterisis := float64(s.HysteresisMenuE3)
		deltaToTriggerCompressor := hysterisis + 1
		// Guard against out of range values
		loweredT := math.Min( // Guard against high values that won't trigger cooling
			upperBound,
			math.Max( // Guard aginst too-low values and integer overflow
				lowerBound,
				current-deltaToTriggerCompressor, // Ideal temp to trigger cooling
			),
		)
		// Lowest allowed by fridge settings

		log.Infof("CycleCompressor loweredT=%#v", loweredT)
		s.TempSet = int8(loweredT)
		log.WithFields(log.Fields{
			"temp set": s.TempSet,
			"on":       s.On,
		}).Debugf("Fridge going to cold setting")
		// block writing while we're cycling
		f.settingsC <- s.Settings
		// TODO see if there's a way to avoid this 30s window where things could get clobbered
		// time after func turn off
		time.AfterFunc(onTime, func() {
			s := f.GetStatusReport().Settings
			s.On = prevSettings.On
			s.TempSet = prevSettings.TempSet
			log.WithFields(log.Fields{
				"temp set": s.TempSet,
				"on":       s.On,
			}).Debugf("Fridge going back to prev settings")
			f.settingsC <- s
		})
	}
}

func main() {
	flag.Parse()
	log.Info("timeout", timeout)
	log.Info("pollrate", pollrate)

	// Use env to override app settings
	timeout = env.GetOrDefaultSecond("TIMEOUT_SEC", *timeoutF)
	pollrate = env.GetOrDefaultSecond("POLLRATE_SEC", *pollrateF)
	adapterName = env.GetOrDefaultString("ADAPTER_NAME", *adapterNameF)
	addr = env.GetOrDefaultString("FRIDGE_ADDR", *addrF)
	storagePath := env.GetOrDefaultString("STORAGE_PATH", *storagePathF)
	minVideoBitrate = env.GetOrDefaultInt("CAM_MIN_VIDEO_BITRATE", *minVideoBitrateF)
	camRotationDegrees = env.GetOrDefaultInt("CAM_ROTATION_DEGREES", *camRotationDegreesF)
	multiStream = env.GetOrDefaultBool("CAM_MULTI_STREAM", *multiStreamF)

	inputDevice = env.GetOrDefaultString("INPUT_DEVICE", *inputDeviceF)
	inputFilename = env.GetOrDefaultString("INPUT_FILENAME", *inputFilenameF)
	loopbackFilename = env.GetOrDefaultString("LOOPBACK_FILENAME", *loopbackFilenameF)
	h264Encoder = env.GetOrDefaultString("H264ENCODER", *h264EncoderF)
	h264Decoder = env.GetOrDefaultString("H264DNECODER", *h264DecoderF)

	log.Info("timeout", timeout)
	log.Info("pollrate", pollrate)

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

	// log.SetFormatter(&log.JSONFormatter{})

	// main context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Subtask quit response channels
	wg := sync.WaitGroup{}

	// Subtask contexts
	clientContext, cancelClient := context.WithCancel(ctx)
	defer cancelClient()

	cycleCompressorContext, cancelCycleCompressor := context.WithCancel(ctx)
	defer cancelCycleCompressor()

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
		log.Debug("Fridge comp. cycles start")
		cycleOnTime := 15 * time.Second // TODO make this come from env/flags
		ccc1, cccc1 := context.WithCancel(cycleCompressorContext)
		defer cccc1()
		ccc2, cccc2 := context.WithCancel(cycleCompressorContext)
		defer cccc2()
		// cycle on startup of daemon
		go fridge.CycleCompressor(ccc1, cycleOnTime)
		// TODO make this 8 hours a flag
		ticker := time.NewTicker(8 * time.Hour)
		for range ticker.C {
			go fridge.CycleCompressor(ccc2, cycleOnTime)
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
	go HKClient(HKClientContext, &wg, &fridge, HKSettings{
		storagePath,
		minVideoBitrate,
		multiStream,
		inputDevice,
		inputFilename,
		loopbackFilename,
		h264Decoder,
		h264Encoder,
	})

	// go CameraClient(cameraClientContext, &wg, cameraResultsC)

	log.Trace("Main waiting...")
	for {
		select {
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
