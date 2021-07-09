package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// var maybeUnlockbytes = []byte{0xfe, 0xfe, 0x11, 0x2, 0x1, 0x0, 0x1, 0x0, 0x24, 0x44, 0xfc, 0x4, 0x0, 0x1, 0x0, 0x0, 0xfb, 0x0, 0x4, 0x75}
// TODO add a factory reset thing gleaned from wireshark

var (
	adapterName = flag.String("adapter", zeroAdapter, "adapter name, e.g. hci0")
	addr        = flag.String("addr", "", "address of remote peripheral (MAC on Linux, UUID on OS X)")
	timeout     = flag.Duration("timeout", 20*time.Second, "overall program timeout")
	pollrate    = flag.Duration("pollrate", 2*time.Second, "magic payload polling rate")
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

	clientContext, cancelClient := context.WithCancel(ctx)
	defer cancelClient()

	// https://rafallorenz.com/go/handle-signals-to-graceful-shutdown-http-server/

	// Listen for control-c
	go func() {
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
		cancelClient()
	}()

	// Kick off bluetooth client
	go func() {
		log.Trace("Launching client")
		err := Client(clientContext, *adapterName, *addr)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
		cancel()
	}()

	log.Trace("Main waiting...")
	select {
	case <-ctx.Done():
		log.Trace("Main context canceled")
		return
	}
}
