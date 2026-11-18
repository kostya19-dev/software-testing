package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Exporter struct {
	docker  *DockerClient
	project string
}

func (e *Exporter) Collect(ctx context.Context) (Snapshot, error) {
	containers, err := e.docker.Containers(ctx, e.project)
	if err != nil {
		return nil, err
	}

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		snapshot = make(Snapshot, len(containers))
	)
	for _, container := range containers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, err := e.docker.Stats(ctx, container.ID)
			if err != nil {
				log.Printf("stats %s: %v", container.Service(), err)
				return
			}
			mu.Lock()
			snapshot[container.Service()] = stats
			mu.Unlock()
		}()
	}
	wg.Wait()
	return snapshot, nil
}

func (e *Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 900*time.Millisecond)
	defer cancel()

	snapshot, err := e.Collect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	if err := snapshot.Render(w); err != nil {
		log.Printf("write metrics: %v", err)
	}
}

func main() {
	project := os.Getenv("STAND_PROJECT")
	if project == "" {
		log.Fatal("STAND_PROJECT is required")
	}
	exporter := &Exporter{docker: NewDockerClient("/var/run/docker.sock"), project: project}

	mux := http.NewServeMux()
	mux.Handle("/metrics", exporter)
	log.Printf("exporting containers of project %s on :9101", project)
	log.Fatal(http.ListenAndServe(":9101", mux))
}
