package ooobot

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

type Ooobot struct {
	sync.Mutex
	out      []Out
	timezone *time.Location
}

type Out struct {
	SlackName string
	Start     time.Time
	End       time.Time
}

type request struct {
	SlackName string `json:"slack_name"`
	FirstDate string `json:"start_date"`
	LastDate  string `json:"end_date"`
}

func New() *Ooobot {
	pacificTime, _ := time.LoadLocation("America/Los_Angeles")
	return &Ooobot{
		timezone: pacificTime,
	}
}

func (o *Ooobot) HandleSlackRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req request
	err = json.Unmarshal(b, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = o.AddOut(req.SlackName, req.FirstDate, req.LastDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (o *Ooobot) AddOut(name, start, end string) error {
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
		SlackName: name,
		Start:     s,
		End:       e.Add(time.Hour*23 + time.Minute*59 + time.Second*59),
	}

	o.out = append(o.out, out)

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
