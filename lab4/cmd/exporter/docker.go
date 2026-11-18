package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Container struct {
	ID     string            `json:"Id"`
	Labels map[string]string `json:"Labels"`
}

func (c Container) Service() string {
	return c.Labels["com.docker.compose.service"]
}

type Stats struct {
	CPU struct {
		Usage struct {
			Total uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		Throttling struct {
			ThrottledTime uint64 `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	Memory struct {
		Usage uint64            `json:"usage"`
		Limit uint64            `json:"limit"`
		Stats map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
	Block struct {
		Entries []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
	Networks map[string]struct {
		Received    uint64 `json:"rx_bytes"`
		Transmitted uint64 `json:"tx_bytes"`
	} `json:"networks"`
	Pids struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

type DockerClient struct {
	http *http.Client
}

func NewDockerClient(socket string) *DockerClient {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
		MaxIdleConnsPerHost: 16,
	}
	return &DockerClient{http: &http.Client{Transport: transport, Timeout: 5 * time.Second}}
}

func (d *DockerClient) Containers(ctx context.Context, project string) ([]Container, error) {
	filters, err := json.Marshal(map[string][]string{"label": {"com.docker.compose.project=" + project}})
	if err != nil {
		return nil, err
	}
	var containers []Container
	err = d.get(ctx, "/containers/json?filters="+url.QueryEscape(string(filters)), &containers)
	return containers, err
}

func (d *DockerClient) Stats(ctx context.Context, id string) (Stats, error) {
	var stats Stats
	err := d.get(ctx, "/containers/"+id+"/stats?stream=false&one-shot=true", &stats)
	return stats, err
}

func (d *DockerClient) get(ctx context.Context, path string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker"+path, nil)
	if err != nil {
		return err
	}
	response, err := d.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("docker api %s: %s", path, response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}
