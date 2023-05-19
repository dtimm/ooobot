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

func (o Out) String() string {
	start := o.Start.Format("2006-01-02")
	end := o.End.Format("2006-01-02")

	if start == end {
		return fmt.Sprintf("<@%s> out of the office on %s.", o.User, start)
	}

	return fmt.Sprintf("<@%s> out of the office from %s to %s.", o.User, o.Start.Format("2006-01-02"), o.End.Format("2006-01-02"))
}

type Ooobot struct {
	sync.Mutex
	out      map[string][]Out
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
		out:                     make(map[string][]Out),
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

	go func() {
		outText := o.WhosOut(values.Get("channel_id"))
		s := strings.NewReader(fmt.Sprintf(`{"text": "%s"}`, o.MakeItFunny(outText)))
		u := values.Get("response_url")
		http.Post(u, "application/json", s)
	}()

	w.WriteHeader(http.StatusOK)
}

func (o *Ooobot) WhosOut(c string) string {
	outList := o.GetOut(time.Now())
	sb := strings.Builder{}
	for _, out := range outList {
		if out.Channel != c {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(out.String())
	}
	if sb.Len() == 0 {
		sb.WriteString("No one is currently out of office.")
	}

	return sb.String()
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

	o.addRange(s, e, out)

	fmt.Printf("added <@%s> out from %s to %s\n", user, start, end)

	return nil
}

func (o *Ooobot) addRange(s, e time.Time, out Out) {
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		if _, ok := o.out[d.Format("2006-01-02")]; !ok {
			o.out[d.Format("2006-01-02")] = []Out{out}
		} else {
			if o.alreadyOut(out.User, d) {
				continue
			}
			o.out[d.Format("2006-01-02")] = append(o.out[d.Format("2006-01-02")], out)
		}
	}
}

func (o *Ooobot) alreadyOut(user string, t time.Time) bool {
	outs, ok := o.out[t.Format("2006-01-02")]
	if !ok {
		return false
	}
	for _, out := range outs {
		if out.User == user {
			return true
		}
	}

	return false
}

func (o *Ooobot) GetOut(t time.Time) []Out {
	o.Lock()
	defer o.Unlock()

	var r []Out
	r = append(r, o.out[t.Format("2006-01-02")]...)

	return r
}

func (o *Ooobot) MakeItFunny(s string) string {
	fmt.Printf("making '%s' funny\n", s)
	resp, err := o.ChatCompletionRequester.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Temperature: 0.7,
			Model:       openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: "This is a bot that makes up creative and humerous reasons for people being out of the office. Each out-of-office message should be converted to a single creative and humerous reason.",
			}, {
				Role:    openai.ChatMessageRoleUser,
				Content: s,
			}},
		},
	)

	if err != nil {
		return s
	}

	fmt.Printf("Here it is, but funny: %s\n", resp.Choices[0].Message.Content)
	return resp.Choices[0].Message.Content
}

func parseText(t string) (string, string, error) {
	s := strings.Split(t, " ")
	if len(s) == 2 {
		return s[0], s[1], nil
	} else if len(s) == 1 {
		return s[0], s[0], nil
	} else if len(s) == 0 {
		today := time.Now().Format("2006-01-02")
		return today, today, nil
	}

	return "", "", fmt.Errorf("invalid text: %s", t)
}
