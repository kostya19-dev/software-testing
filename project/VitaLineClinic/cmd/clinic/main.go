package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vitaline-clinic/internal/bootstrap"
	"vitaline-clinic/internal/delivery/cli"
	guidelivery "vitaline-clinic/internal/delivery/gui"
	httpdelivery "vitaline-clinic/internal/delivery/http"
)

const (
	modeAPI = "api"
	modeGUI = "gui"
	modeCLI = "cli"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	mode := modeCLI
	if len(args) > 0 && isMode(args[0]) {
		mode = args[0]
		args = args[1:]
	}

	fs := flag.NewFlagSet("clinic "+mode, flag.ContinueOnError)
	configPath := fs.String("config", "", "путь к конфиг-файлу")
	debug := fs.Bool("debug", false, "включить debug-логирование")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	ctx := context.Background()
	app, err := bootstrap.New(ctx, bootstrap.Options{
		ConfigPath:   *configPath,
		Debug:        *debug,
		SkipDatabase: mode == modeGUI,
	})
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "close resources: %v\n", closeErr)
		}
	}()

	switch mode {
	case modeAPI:
		return runAPI(app)
	case modeGUI:
		return runGUI(app)
	case modeCLI:
		return runCLI(ctx, app, fs.Args())
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

func isMode(value string) bool {
	return value == modeAPI || value == modeGUI || value == modeCLI
}

func runAPI(app *bootstrap.App) error {
	server := httpdelivery.NewServer(
		app.APIAddress,
		app.AppName,
		app.Health,
		httpdelivery.Services{
			Patients:     app.UseCases.Patients(),
			Appointments: app.UseCases.Appointments(),
			Schedule:     app.UseCases.Schedule(),
			Visits:       app.UseCases.Visits(),
			Emergencies:  app.UseCases.Emergencies(),
		},
		app.Logger,
	)
	return serve(app, server)
}

func runGUI(app *bootstrap.App) error {
	services := guidelivery.NewBackendServices(app.GUIAPIBaseURL, app.Logger)
	server := guidelivery.NewServer(
		app.GUIAddress,
		app.AppName,
		services,
		app.Logger,
	)
	return serve(app, server)
}

func runCLI(ctx context.Context, app *bootstrap.App, args []string) error {
	commandApp := cli.NewCommandApp(
		app.AppName,
		app.Logger,
		app.UseCases.Patients(),
		app.UseCases.Appointments(),
		app.UseCases.Visits(),
		app.UseCases.Schedule(),
		os.Stdout,
	)
	if err := commandApp.Run(ctx, args); err != nil {
		app.Logger.Error("cli command failed", "error", err)
		return fmt.Errorf("error: %w", err)
	}
	return nil
}

type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

func serve(app *bootstrap.App, srv server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		app.Logger.Info("http shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.Error("http server stopped with error", "error", err)
			return fmt.Errorf("http server failed: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error("http server shutdown failed", "error", err)
		return fmt.Errorf("http shutdown failed: %w", err)
	}
	return nil
}
