package main

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"

	log "github.com/sirupsen/logrus"
)

//go:embed status.tmpl
var statusTmpl string

var statuspage = template.Must(template.New("status").Parse(statusTmpl))

func handleStatus(f *Fridge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := f.GetStatusReport()
		json, err := s.MarshalJSON()
		if err != nil {
			panic(err)
		}
		data := struct{ InitialJSON string }{InitialJSON: fmt.Sprintf("%s", json)}
		err = statuspage.ExecuteTemplate(w, "status", data)
		if err != nil {
			panic(err)
		}
	}
}

func handleJSON(f *Fridge) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method == http.MethodGet {
			s := f.GetStatusReport()
			json, err := s.MarshalJSON()
			if err != nil {
				panic(err)
			}
			w.Write(json)
		} else {
			fmt.Println(r.Method)
		}
	}
}

// JSONClient serves json
func JSONClient(ctx context.Context, port string, f *Fridge) {
	// TODO use context to cancel
	if port == "" {
		port = "8080"
	}

	log.Debugf("JSON server starting on port %s ...", port)
	http.HandleFunc("/", handleStatus(f))
	http.HandleFunc("/status", handleJSON(f))
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil); err != nil {
		log.Panic(err)
	}
}
