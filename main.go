package main

import (
	"fmt"
	"io"
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

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Printf("req url: %s\n", req.URL)
		fmt.Printf("req method: %s\n", req.Method)

		defer req.Body.Close()
		b, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Printf("error reading body: %s\n", err)
		} else {
			fmt.Printf("req body: %s\n", b)
		}
		w.WriteHeader(http.StatusBadRequest)
	})

	cr := cors.New(cors.Options{
		AllowedOrigins: []string{"*.vmware.com"},
	})

	s := &http.Server{
		Handler:      cr.Handler(r),
		Addr:         fmt.Sprintf("0.0.0.0:%d", opt.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Printf("Listening on %s\n", s.Addr)
	log.Fatal(s.ListenAndServe())
}
