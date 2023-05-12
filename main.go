package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dtimm/ooobot/pkg/ooobot"

	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"github.com/rs/cors"
)

type options struct {
	Port int `short:"p" long:"port" env:"PORT" description:"port to listen on" default:"8080"`
}

func main() {
	var opt options
	flags.Parse(&opt)

	o := ooobot.New()

	r := mux.NewRouter()
	r.HandleFunc("/v1/outofoffice", o.HandleSlackRequest)

	cr := cors.New(cors.Options{
		AllowedOrigins: []string{"*.vmware.com"},
	})

	s := &http.Server{
		Handler:      cr.Handler(r),
		Addr:         fmt.Sprintf("0.0.0.0:%d", opt.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(s.ListenAndServe())
}
