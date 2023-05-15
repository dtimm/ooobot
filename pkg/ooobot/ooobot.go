package ooobot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Out struct {
	Channel string
	User    string
	Start   time.Time
	End     time.Time
}
type Ooobot struct {
	sync.Mutex
	out      []Out
	timezone *time.Location
	ChatCompletionRequester
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ChatCompletionRequester
type ChatCompletionRequester interface {
	CreateChatCompletion(context.Context, openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

func New(r ChatCompletionRequester) *Ooobot {
	pacificTime, _ := time.LoadLocation("America/Los_Angeles")
	return &Ooobot{
		timezone:                pacificTime,
		ChatCompletionRequester: r,
	}
}

func (o *Ooobot) HandleOutRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error reading body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	values, err := url.ParseQuery(string(b))
	if err != nil {
		fmt.Fprintf(os.Stdout, "error parsing query: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t := values.Get("text")
	start, end, err := parseText(t)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error parsing text: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	channel := values.Get("channel_id")
	user := values.Get("user_id")

	err = o.AddOut(
		channel,
		user,
		start,
		end,
	)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error adding out: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	u := values.Get("response_url")
	s := strings.NewReader(fmt.Sprintf(`{"text": "set <@%s> out of office from %s to %s"}`, user, start, end))
	http.Post(u, "application/json", s)

	w.WriteHeader(http.StatusOK)
}

func (o *Ooobot) HandleWhosOutRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stdout, "error reading body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	values, err := url.ParseQuery(string(b))
	if err != nil {
		fmt.Fprintf(os.Stdout, "error parsing query: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	outList := o.GetOut(time.Now())
	sb := strings.Builder{}
	for _, out := range outList {
		if out.Channel != values.Get("channel_id") {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(outString(out))
	}
	if sb.Len() == 0 {
		sb.WriteString("No one is currently out of office.")
	}

	go func() {
		s := strings.NewReader(fmt.Sprintf(`{"text": "%s"}`, o.makeItFunny(sb.String())))
		u := values.Get("response_url")
		http.Post(u, "application/json", s)
	}()

	w.WriteHeader(http.StatusOK)
}

func (o *Ooobot) AddOut(channel, user, start, end string) error {
	o.Lock()
	defer o.Unlock()

	s, err := time.ParseInLocation("2006-01-02", start, o.timezone)
	if err != nil {
		return err
	}
	e, err := time.ParseInLocation("2006-01-02", end, o.timezone)
	if err != nil {
		return err
	}

	out := Out{
		Channel: channel,
		User:    user,
		Start:   s,
		End:     e.Add(time.Hour*23 + time.Minute*59 + time.Second*59),
	}

	o.out = append(o.out, out)

	fmt.Printf("added <@%s> out from %s to %s\n", user, start, end)

	return nil
}

func (o *Ooobot) GetOut(t time.Time) []Out {
	o.Lock()
	defer o.Unlock()

	var r []Out
	for _, out := range o.out {
		if t.After(out.Start) && t.Before(out.End) {
			r = append(r, out)
		}
	}

	return r
}

func outString(out Out) string {
	start := out.Start.Format("2006-01-02")
	end := out.End.Format("2006-01-02")

	if start == end {
		return fmt.Sprintf("<@%s> out of the office on %s.", out.User, start)
	}

	return fmt.Sprintf("<@%s> out of the office from %s to %s.", out.User, out.Start.Format("2006-01-02"), out.End.Format("2006-01-02"))
}

func (o *Ooobot) makeItFunny(s string) string {
	fmt.Printf("making '%s' funny\n", s)
	resp, err := o.ChatCompletionRequester.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Temperature: 0.7,
			Model:       openai.GPT4,
			Messages: []openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("Make up creative and humerous reasons for the following: %s", s),
			}},
		},
	)

	if err != nil {
		return s
	}
	return resp.Choices[0].Message.Content
}

func parseText(t string) (string, string, error) {
	s := strings.Split(t, " ")
	if len(s) == 2 {
		return s[0], s[1], nil
	} else if len(s) == 1 {
		return s[0], s[0], nil
	}

	return "", "", fmt.Errorf("invalid text: %s", t)
}
