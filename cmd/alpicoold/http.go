package main

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

var mimeTypeJSON = mime.TypeByExtension(".json")
var contentType = http.CanonicalHeaderKey("content-type")

func handleGet(f *Fridge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		switch r.Method {
		case http.MethodGet:
			s := f.GetStatusReport()
			json, err := s.MarshalJSON()
			if err != nil {
				panic(err)
			}
			w.Header().Set(contentType, mimeTypeJSON)
			w.Write(json)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			log.WithFields(log.Fields{
				"client": "JSONClient",
				"method": r.Method,
			}).Debug("http unsupported method")
		}
	}
}

// JSONClient serves json
func JSONClient(ctx context.Context, wg *sync.WaitGroup, port string, f *Fridge) {
	wg.Add(1)
	defer func() {
		log.WithFields(log.Fields{
			"client": "JSONClient",
		}).Trace("Calling done on main wait group")
		wg.Done()
	}()

	if port == "" {
		port = "80"
	}

	log.WithFields(log.Fields{
		"client": "JSONClient",
	}).Debugf("server starting on port %s", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleGet(f))
	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	server.RegisterOnShutdown(func() {
		log.WithFields(log.Fields{
			"err":    ctx.Err(),
			"client": "JSONClient",
		}).Debug("shutting down")
	})

	go func() {
		<-ctx.Done()
		log.WithFields(log.Fields{"client": "JSONClient"}).Tracef("client shutting down")
		server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		panic(err)
	}
	log.WithFields(log.Fields{"client": "JSONClient"}).Debug("done")
}
