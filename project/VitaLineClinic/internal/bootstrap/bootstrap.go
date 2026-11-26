package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"vitaline-clinic/internal/appconfig"
	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/integration/yookassa"
	"vitaline-clinic/internal/logging"
	mongostorage "vitaline-clinic/internal/storage/mongodb"
	"vitaline-clinic/internal/storage/postgres"
	"vitaline-clinic/internal/tracing"
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
	Payments() usecase.IPaymentService
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
	payments     usecase.IPaymentService
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

// Payments returns nil when payments.mode is "disabled".
func (s *useCases) Payments() usecase.IPaymentService {
	return s.payments
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
	Tracer        *tracing.Tracer

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

	app := &App{
		ConfigPath:    configPath,
		AppName:       cfg.App.Name,
		APIAddress:    cfg.API.Address,
		GUIAddress:    cfg.GUI.Address,
		GUIAPIBaseURL: cfg.GUI.APIBaseURL,
		Logger:        logs.Logger,
		closers:       []closer{logs},
	}

	if opts.SkipDatabase {
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

	if err := app.connect(ctx, cfg, logs); err != nil {
		_ = logs.Close()
		return nil, err
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
		"tracing", cfg.Tracing.Enabled,
		"log_extended", cfg.Logging.Extended,
		"appointment_min_lead_time_minutes", cfg.Business.AppointmentMinLeadTimeMinutes,
	)

	return app, nil
}

// connect starts tracing, opens the configured database and wires use cases.
func (a *App) connect(ctx context.Context, cfg appconfig.Config, logs *logging.Instance) error {
	tracer, err := tracing.Start(ctx, cfg.Tracing)
	if err != nil {
		return err
	}

	store, err := openStorage(ctx, cfg.Database, tracer, logs.Logger)
	if err != nil {
		return err
	}

	a.Tracer = tracer
	a.Health = store.health
	a.closers = append(a.closers, tracer, store.closer)

	payments, err := newPaymentService(store, cfg.Payments)
	if err != nil {
		return err
	}
	cases := newUseCases(store, cfg.Business)
	cases.payments = payments
	a.UseCases = cases
	logs.Logger.Info("payments configured",
		"mode", cfg.Payments.Mode,
		"provider_url", cfg.Payments.Endpoint().BaseURL,
		"appointment_price", cfg.Payments.AppointmentPrice,
	)
	return nil
}

// newPaymentService selects the YooKassa endpoint by payments.mode: the mock server
// or the real API. Both are served by the same client and the same contract.
func newPaymentService(store storage, cfg appconfig.PaymentsConfig) (usecase.IPaymentService, error) {
	if cfg.Mode == appconfig.PaymentsDisabled {
		return nil, nil
	}
	price, err := domain.ParseMoney(cfg.AppointmentPrice, cfg.Currency)
	if err != nil {
		return nil, fmt.Errorf("payments.appointment_price: %w", err)
	}
	endpoint := cfg.Endpoint()
	name := "yookassa"
	if cfg.Mode == appconfig.PaymentsMock {
		name = "yookassa-mock"
	}
	gateway, err := yookassa.New(yookassa.Config{
		Name:      name,
		BaseURL:   endpoint.BaseURL,
		ShopID:    endpoint.ShopID,
		SecretKey: endpoint.SecretKey,
		Timeout:   time.Duration(cfg.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return usecase.NewPaymentService(store.payments, store.appointments, gateway, usecase.PaymentSettings{
		Price:     price,
		ReturnURL: cfg.ReturnURL,
	}), nil
}

// storage is the set of repositories backed by one database connection.
type storage struct {
	patients     usecase.PatientRepository
	appointments usecase.AppointmentRepository
	schedule     usecase.ScheduleRepository
	visits       usecase.VisitRepository
	emergencies  usecase.EmergencyRepository
	payments     usecase.PaymentRepository
	health       HealthChecker
	closer       closer
}

func openStorage(ctx context.Context, cfg appconfig.DatabaseConfig, tracer *tracing.Tracer, logger *slog.Logger) (storage, error) {
	switch cfg.Driver {
	case "postgres":
		return openPostgres(ctx, cfg)
	case "mongodb":
		return openMongo(ctx, cfg, tracer, logger)
	default:
		return storage{}, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}

func openPostgres(ctx context.Context, cfg appconfig.DatabaseConfig) (storage, error) {
	db, err := postgres.Open(ctx, cfg.URL)
	if err != nil {
		return storage{}, err
	}
	return storage{
		patients:     postgres.NewPatientRepository(db),
		appointments: postgres.NewAppointmentRepository(db),
		schedule:     postgres.NewScheduleRepository(db),
		visits:       postgres.NewVisitRepository(db),
		emergencies:  postgres.NewEmergencyRepository(db),
		payments:     postgres.NewPaymentRepository(db),
		health:       &databaseHealthChecker{ping: db.PingContext},
		closer:       db,
	}, nil
}

func openMongo(ctx context.Context, cfg appconfig.DatabaseConfig, tracer *tracing.Tracer, logger *slog.Logger) (storage, error) {
	store, err := mongostorage.Open(ctx, cfg.URL, cfg.Name,
		tracer.CommandMonitor(),
		logging.CommandMonitor(logger),
	)
	if err != nil {
		return storage{}, err
	}
	db := store.Database()
	return storage{
		patients:     mongostorage.NewPatientRepository(db),
		appointments: mongostorage.NewAppointmentRepository(db),
		schedule:     mongostorage.NewScheduleRepository(db),
		visits:       mongostorage.NewVisitRepository(db),
		emergencies:  mongostorage.NewEmergencyRepository(db),
		payments:     mongostorage.NewPaymentRepository(db),
		health:       &databaseHealthChecker{ping: store.Ping},
		closer:       store,
	}, nil
}

func newUseCases(store storage, business appconfig.BusinessConfig) *useCases {
	minLeadTime := time.Duration(business.AppointmentMinLeadTimeMinutes) * time.Minute
	return &useCases{
		patients: usecase.NewPatientService(store.patients),
		appointments: usecase.NewAppointmentService(
			store.appointments,
			usecase.WithAppointmentMinLeadTime(minLeadTime),
		),
		schedule:    usecase.NewScheduleService(store.schedule),
		visits:      usecase.NewVisitService(store.visits, store.appointments),
		emergencies: usecase.NewEmergencyService(store.emergencies, store.appointments),
	}
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
