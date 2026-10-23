package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"vitaline-clinic/internal/appconfig"
	"vitaline-clinic/internal/logging"
	mongostorage "vitaline-clinic/internal/storage/mongodb"
	"vitaline-clinic/internal/storage/postgres"
	"vitaline-clinic/internal/usecase"
)

type Options struct {
	ConfigPath   string
	Debug        bool
	SkipDatabase bool
}

type UseCases interface {
	Patients() usecase.IPatientService
	Appointments() usecase.IAppointmentService
	Schedule() usecase.IScheduleService
	Visits() usecase.IVisitService
	Emergencies() usecase.IEmergencyService
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type closer interface {
	Close() error
}

type useCases struct {
	patients     usecase.IPatientService
	appointments usecase.IAppointmentService
	schedule     usecase.IScheduleService
	visits       usecase.IVisitService
	emergencies  usecase.IEmergencyService
}

func (s *useCases) Patients() usecase.IPatientService {
	return s.patients
}

func (s *useCases) Appointments() usecase.IAppointmentService {
	return s.appointments
}

func (s *useCases) Schedule() usecase.IScheduleService {
	return s.schedule
}

func (s *useCases) Visits() usecase.IVisitService {
	return s.visits
}

func (s *useCases) Emergencies() usecase.IEmergencyService {
	return s.emergencies
}

type databaseHealthChecker struct {
	ping func(context.Context) error
}

func (h *databaseHealthChecker) Ping(ctx context.Context) error {
	return h.ping(ctx)
}

type App struct {
	ConfigPath    string
	AppName       string
	APIAddress    string
	GUIAddress    string
	GUIAPIBaseURL string
	Logger        *slog.Logger
	UseCases      UseCases
	Health        HealthChecker

	closers []closer
}

func New(ctx context.Context, opts Options) (*App, error) {
	cfg, configPath, err := appconfig.LoadWithOptions(opts.ConfigPath, appconfig.LoadOptions{
		RequireDatabase: !opts.SkipDatabase,
	})
	if err != nil {
		return nil, err
	}

	logs, err := logging.New(cfg.Logging, opts.Debug)
	if err != nil {
		return nil, err
	}

	if opts.SkipDatabase {
		app := &App{
			ConfigPath:    configPath,
			AppName:       cfg.App.Name,
			APIAddress:    cfg.API.Address,
			GUIAddress:    cfg.GUI.Address,
			GUIAPIBaseURL: cfg.GUI.APIBaseURL,
			Logger:        logs.Logger,
			closers:       []closer{logs},
		}
		app.Logger.Info(
			"application bootstrap completed",
			"config_path", app.ConfigPath,
			"app_name", cfg.App.Name,
			"environment", cfg.App.Environment,
			"debug", opts.Debug,
			"http_address", cfg.API.Address,
			"gui_address", cfg.GUI.Address,
			"gui_api_base_url", cfg.GUI.APIBaseURL,
		)
		return app, nil
	}

	var patientRepo usecase.PatientRepository
	var appointmentRepo usecase.AppointmentRepository
	var scheduleRepo usecase.ScheduleRepository
	var visitRepo usecase.VisitRepository
	var emergencyRepo usecase.EmergencyRepository
	var health HealthChecker
	closers := []closer{logs}

	switch cfg.Database.Driver {
	case "postgres":
		db, err := postgres.Open(ctx, cfg.Database.URL)
		if err != nil {
			_ = logs.Close()
			return nil, err
		}
		patientRepo = postgres.NewPatientRepository(db)
		appointmentRepo = postgres.NewAppointmentRepository(db)
		scheduleRepo = postgres.NewScheduleRepository(db)
		visitRepo = postgres.NewVisitRepository(db)
		emergencyRepo = postgres.NewEmergencyRepository(db)
		health = &databaseHealthChecker{ping: db.PingContext}
		closers = append(closers, db)
	case "mongodb":
		store, err := mongostorage.Open(ctx, cfg.Database.URL, cfg.Database.Name)
		if err != nil {
			_ = logs.Close()
			return nil, err
		}
		patientRepo = mongostorage.NewPatientRepository(store.Database())
		appointmentRepo = mongostorage.NewAppointmentRepository(store.Database())
		scheduleRepo = mongostorage.NewScheduleRepository(store.Database())
		visitRepo = mongostorage.NewVisitRepository(store.Database())
		emergencyRepo = mongostorage.NewEmergencyRepository(store.Database())
		health = &databaseHealthChecker{ping: store.Ping}
		closers = append(closers, store)
	default:
		_ = logs.Close()
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	minLeadTime := time.Duration(cfg.Business.AppointmentMinLeadTimeMinutes) * time.Minute

	patientSvc := usecase.NewPatientService(patientRepo)
	appointmentSvc := usecase.NewAppointmentService(
		appointmentRepo,
		usecase.WithAppointmentMinLeadTime(minLeadTime),
	)
	scheduleSvc := usecase.NewScheduleService(scheduleRepo)
	visitSvc := usecase.NewVisitService(visitRepo, appointmentRepo)
	emergencySvc := usecase.NewEmergencyService(emergencyRepo, appointmentRepo)

	app := &App{
		ConfigPath:    configPath,
		AppName:       cfg.App.Name,
		APIAddress:    cfg.API.Address,
		GUIAddress:    cfg.GUI.Address,
		GUIAPIBaseURL: cfg.GUI.APIBaseURL,
		Logger:        logs.Logger,
		UseCases: &useCases{
			patients:     patientSvc,
			appointments: appointmentSvc,
			schedule:     scheduleSvc,
			visits:       visitSvc,
			emergencies:  emergencySvc,
		},
		Health:  health,
		closers: closers,
	}

	app.Logger.Info(
		"application bootstrap completed",
		"config_path", app.ConfigPath,
		"app_name", cfg.App.Name,
		"environment", cfg.App.Environment,
		"debug", opts.Debug,
		"http_address", cfg.API.Address,
		"gui_address", cfg.GUI.Address,
		"database_driver", cfg.Database.Driver,
		"database_name", cfg.Database.Name,
		"appointment_min_lead_time_minutes", cfg.Business.AppointmentMinLeadTimeMinutes,
	)

	return app, nil
}

func (a *App) Close() error {
	var result error

	for _, c := range a.closers {
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil && result == nil {
			result = fmt.Errorf("close resource: %w", err)
		}
	}

	return result
}
