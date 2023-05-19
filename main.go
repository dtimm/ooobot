package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dtimm/ooobot/pkg/ooobot"
	"github.com/sashabaranov/go-openai"
	"github.com/slack-go/slack"

	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"github.com/rs/cors"
)

type options struct {
	Port        int    `short:"p" long:"port" env:"PORT" description:"port to listen on" default:"8080"`
	OpenAIToken string `short:"t" long:"openai-api-token" env:"OPENAI_API_KEY" description:"OpenAI API token"`
	SlackToken  string `short:"s" long:"slack-oauth-token" env:"SLACK_OAUTH_TOKEN" description:"Slack OAuth token"`
}

var SLACK_OAUTH_TOKEN string

func main() {
	var opt options
	flags.Parse(&opt)

	SLACK_OAUTH_TOKEN = opt.SlackToken
	c := openai.NewClient(opt.OpenAIToken)
	o := ooobot.New(c)

	r := mux.NewRouter()
	r.HandleFunc("/v1/outofoffice", o.HandleOutRequest)
	r.HandleFunc("/v1/whosout", o.HandleWhosOutRequest)

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
		w.WriteHeader(http.StatusNotFound)
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

	go sendSlackMessages(o)

	log.Fatal(s.ListenAndServe())
}

func ItsTime() bool {
	pacificTime, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Now().In(pacificTime)
	nine := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, pacificTime)
	nineOFive := time.Date(now.Year(), now.Month(), now.Day(), 9, 5, 0, 0, pacificTime)
	if now.Before(nine) || now.After(nineOFive) {
		return false
	}
	return true
}

func sendSlackMessages(o *ooobot.Ooobot) {
	ticker := time.NewTicker(300 * time.Second)
	api := slack.New(SLACK_OAUTH_TOKEN)
	for range ticker.C {
		if !ItsTime() {
			continue
		}
		messages := generateMessages(o)
		for c, t := range messages {
			msg := o.MakeItFunny(t)
			_, _, _, err := api.SendMessage(c, slack.MsgOptionText(msg, false))
			if err != nil {
				fmt.Printf("error sending message to channel %s: %s\n", c, err)
			}
		}
	}
}

func generateMessages(o *ooobot.Ooobot) map[string]string {
	channels := make(map[string]bool)
	outList := o.GetOut(time.Now())
	for _, out := range outList {
		channels[out.Channel] = true
	}

	yeee := make(map[string]string)
	for c := range channels {
		yeee[c] = o.WhosOut(c)
	}
	return yeee
}
