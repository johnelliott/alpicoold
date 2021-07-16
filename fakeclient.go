package main

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// FakeClient is an imaginary client for homekit preparation
func FakeClient(ctx context.Context, wg *sync.WaitGroup, responses chan int) {
	log.Trace("FakeClient start")

	// Start some stuff
	log.Trace("FakeClient setup tasks done")
	tic := time.NewTicker(700 * time.Millisecond)

	log.Trace("FakeClient starting wait/block/exit")

	go func() {
		wg.Add(1)
		defer func() {
			log.Trace("Fake client calling done on main wait group")
			wg.Done()
		}()
		log.Trace("Fake client looping now")
		for {
			select {
			case <-ctx.Done():
				log.Trace("FakeClient ctx canceled")
				return
			case <-tic.C:
				log.Trace("Sending fake update")
				responses <- 100
				log.Trace("Fake update sent")
			}
		}
	}()
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
