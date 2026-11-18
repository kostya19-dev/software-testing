package stand

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Point struct {
	Time  float64
	Value float64
}

type Series map[string][]Point

type Client struct {
	base string
	http *http.Client
}

func NewPrometheus(base string) *Client {
	return &Client{base: base, http: &http.Client{Timeout: time.Minute}}
}

func (c *Client) WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := c.http.Get(c.base + "/-/ready")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("prometheus at %s is not ready after %s", c.base, timeout)
}

func (c *Client) Range(query string, start, end float64, step time.Duration, label string) (Series, error) {
	params := url.Values{
		"query": {query},
		"start": {strconv.FormatFloat(start, 'f', 3, 64)},
		"end":   {strconv.FormatFloat(end, 'f', 3, 64)},
		"step":  {strconv.FormatFloat(step.Seconds(), 'f', -1, 64)},
	}
	response, err := c.http.Get(c.base + "/api/v1/query_range?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var payload struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Values [][2]any          `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Status != "success" {
		return nil, fmt.Errorf("prometheus query %q: %s", query, payload.Error)
	}

	series := Series{}
	for _, item := range payload.Data.Result {
		points := make([]Point, 0, len(item.Values))
		for _, pair := range item.Values {
			t, _ := pair[0].(float64)
			raw, _ := pair[1].(string)
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue
			}
			points = append(points, Point{Time: t, Value: v})
		}
		series[item.Metric[label]] = points
	}
	return series, nil
}
