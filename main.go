package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	adapterName = flag.String("adapter", zeroAdapter, "adapter name, e.g. hci0")
	addr        = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	timeout     = flag.Duration("timeout", 20*time.Minute, "overall program timeout")
	pollrate    = flag.Duration("pollrate", 1*time.Second, "magic payload polling rate")
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
		log.SetLevel(log.TraceLevel)
	}

	// log.SetFormatter(&log.JSONFormatter{})

	// main context
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
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

	// Subtask comm channels
	fridgeStatusC := make(chan StatusReport)

	// Listen for control-c subtask
	go func() {
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
		log.Trace("Listening for signals")
		s := <-sig
		log.Debug("Got signal:", s)
		cancel()
	}()

	// Kick off bluetooth client
	go func() {
		log.Debug("Launching client")
		err := Client(clientContext, &wg, fridgeStatusC, *adapterName, *addr)
		if err == context.Canceled || err == context.DeadlineExceeded {
			log.Debug("Client: ", err)
		} else if err != nil {
			log.Error(err)
		}
		log.Debug("Client done")
		// cancel() main context is already canceled or things are done
	}()

	// Kick off homekit client
	go HKClient(HKClientContext, &wg, fridgeStatusC)

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
				finalTO := 30 * time.Second
				log.Debugf("Waiting %v then exiting", finalTO)
				time.AfterFunc(finalTO, func() {
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
