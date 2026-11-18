package clinic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"slices"
	"time"
)

type Operation string

const (
	Register       Operation = "register"
	Login          Operation = "login"
	BrowseSchedule Operation = "browse_schedule"
	Book           Operation = "book_appointment"
	MyAppointments Operation = "my_appointments"
	Cancel         Operation = "cancel_appointment"
	StartVisit     Operation = "start_visit"
	CompleteVisit  Operation = "complete_visit"
	VisitHistory   Operation = "visit_history"
)

const requestTimeout = 10 * time.Second

type Result struct {
	Operation Operation
	Status    int
	Latency   time.Duration
	At        time.Time
	Expected  bool
	Invalid   bool
}

type Observer interface {
	Observe(Result)
}

type Client struct {
	base     string
	http     *http.Client
	observer Observer
}

func NewClient(base string, observer Observer) *Client {
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        4096,
		MaxIdleConnsPerHost: 4096,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Client{base: base, http: &http.Client{Transport: transport, Timeout: requestTimeout}, observer: observer}
}

type call struct {
	operation Operation
	method    string
	path      string
	body      any
	expect    []int
	invalid   bool
}

func (c *Client) send(ctx context.Context, rq call) (int, []byte) {
	var payload io.Reader
	if rq.body != nil {
		data, err := json.Marshal(rq.body)
		if err != nil {
			panic(err)
		}
		payload = bytes.NewReader(data)
	}

	start := time.Now()
	status, body := c.exchange(ctx, rq.method, rq.path, payload)
	end := time.Now()

	c.observer.Observe(Result{
		Operation: rq.operation,
		Status:    status,
		Latency:   end.Sub(start),
		At:        end,
		Expected:  slices.Contains(rq.expect, status),
		Invalid:   rq.invalid,
	})
	return status, body
}

func (c *Client) exchange(ctx context.Context, method, path string, payload io.Reader) (int, []byte) {
	request, err := http.NewRequestWithContext(ctx, method, c.base+path, payload)
	if err != nil {
		return 0, nil
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return 0, nil
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil
	}
	return response.StatusCode, body
}

func (c *Client) succeeded(ctx context.Context, rq call) bool {
	status, _ := c.send(ctx, rq)
	return status == rq.expect[0]
}

func (c *Client) created(ctx context.Context, rq call) string {
	status, body := c.send(ctx, rq)
	if status != rq.expect[0] {
		return ""
	}
	var entity struct {
		ID string
	}
	if json.Unmarshal(body, &entity) != nil {
		return ""
	}
	return entity.ID
}

type Observers []Observer

func (o Observers) Observe(r Result) {
	for _, observer := range o {
		observer.Observe(r)
	}
}
