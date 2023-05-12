package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dtimm/ooobot/pkg/ooobot"

	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"github.com/rs/cors"
)

type options struct{}

func main() {
	var opt options
	flags.Parse(&opt)

	o := ooobot.New()

	r := mux.NewRouter()
	r.HandleFunc("/v1/outofoffice", o.HandleSlackRequest)

	cr := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
	})

	s := &http.Server{
		Handler:      cr.Handler(r),
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(s.ListenAndServe())
}
