package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"lab4/internal/clinic"
	"lab4/internal/load"
	"lab4/internal/samples"
)

const (
	gracePeriod  = 10 * time.Second
	vusInterval  = time.Second
	metricsAddr  = ":9102"
	scenarioFile = "/config/scenarios.json"
	resultsDir   = "/results"
)

type counter struct {
	total, failed, invalid atomic.Int64
}

func (c *counter) Observe(r clinic.Result) {
	c.total.Add(1)
	if !r.Expected {
		c.failed.Add(1)
	}
	if r.Invalid {
		c.invalid.Add(1)
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	scale, err := strconv.ParseFloat(env("TIME_SCALE", "1"), 64)
	if err != nil {
		return fmt.Errorf("TIME_SCALE: %w", err)
	}
	catalog, err := load.LoadCatalog(scenarioFile)
	if err != nil {
		return err
	}
	plan, err := catalog.Get(env("SCENARIO", ""))
	if err != nil {
		return err
	}

	start := time.Now()
	writer, err := samples.Create(resultsDir, samples.Meta{StartUnixMs: start.UnixMilli(), Scenario: plan.Name, Scale: scale})
	if err != nil {
		return err
	}

	engine := &load.Engine{Schedule: load.Schedule(plan.Stages(scale)), Grace: gracePeriod}
	stats := &counter{}
	metrics := newLiveMetrics(engine.Active)
	api := clinic.NewClient(env("API_URL", "http://api:8080"), clinic.Observers{writer, metrics, stats})
	engine.NewSession = func(id int) load.Session {
		return clinic.NewPatient(api, id, plan.ErrorRate)
	}

	go serveMetrics(metrics)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	sampling, stopSampling := context.WithCancel(ctx)
	go recordVUs(sampling, writer, engine)

	log.Printf("scenario %s (%s), scale %g, duration %s", plan.Name, plan.Title, scale, engine.Schedule.Total())
	engine.Run(ctx)
	stopSampling()
	writer.RecordVUs(time.Now(), 0)

	if err := writer.Close(); err != nil {
		return err
	}
	elapsed := time.Since(start).Seconds()
	log.Printf("done in %.0fs: %d requests (%.1f rps), unexpected %d, user errors %d",
		elapsed, stats.total.Load(), float64(stats.total.Load())/elapsed, stats.failed.Load(), stats.invalid.Load())
	return ctx.Err()
}

func recordVUs(ctx context.Context, writer *samples.Writer, engine *load.Engine) {
	ticker := time.NewTicker(vusInterval)
	defer ticker.Stop()
	for {
		writer.RecordVUs(time.Now(), engine.Active())
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func serveMetrics(handler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)
	if err := http.ListenAndServe(metricsAddr, mux); err != nil {
		log.Printf("metrics server: %v", err)
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
