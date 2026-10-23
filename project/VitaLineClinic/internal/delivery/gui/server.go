package gui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

const (
	defaultDoctorID  = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	defaultPatientID = "22222222-2222-2222-2222-222222222222"
)

type Services struct {
	Patients     usecase.IPatientService
	Appointments usecase.IAppointmentService
	Schedule     usecase.IScheduleService
	Visits       usecase.IVisitService
	Emergencies  usecase.IEmergencyService
}

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	appName    string
	services   Services
	presenter  *Presenter
}

func NewServer(address, appName string, services Services, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	srv := &Server{
		logger:    logger,
		appName:   appName,
		services:  services,
		presenter: NewPresenter(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleRoot)
	mux.HandleFunc("/patients", srv.handlePatients)
	mux.HandleFunc("/patients/register", srv.handlePatientRegister)
	mux.HandleFunc("/patients/delete", srv.handlePatientDelete)
	mux.HandleFunc("/appointments", srv.handleAppointments)
	mux.HandleFunc("/appointments/create", srv.handleAppointmentCreate)
	mux.HandleFunc("/appointments/cancel", srv.handleAppointmentCancel)
	mux.HandleFunc("/schedule", srv.handleSchedule)
	mux.HandleFunc("/schedule/create", srv.handleScheduleCreate)
	mux.HandleFunc("/schedule/delete", srv.handleScheduleDelete)
	mux.HandleFunc("/visits", srv.handleVisits)
	mux.HandleFunc("/visits/start", srv.handleVisitStart)
	mux.HandleFunc("/visits/complete", srv.handleVisitComplete)
	mux.HandleFunc("/emergencies", srv.handleEmergencies)
	mux.HandleFunc("/emergencies/create", srv.handleEmergencyCreate)
	mux.HandleFunc("/emergencies/offers/accept", srv.handleRescheduleOfferAccept)
	mux.HandleFunc("/emergencies/offers/decline", srv.handleRescheduleOfferDecline)

	srv.httpServer = &http.Server{
		Addr:    address,
		Handler: srv.logRequests(mux),
	}
	return srv
}

func (s *Server) ListenAndServe() error {
	s.logger.Info("gui server starting", "address", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("gui server shutting down")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/patients", http.StatusSeeOther)
}

func (s *Server) handlePatients(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodGet) {
		return
	}

	patients, err := s.services.Patients.GetAll(r.Context())
	if err != nil {
		s.renderError(w, r, "patients", "Пациенты", err)
		return
	}

	vm := s.baseView(r, "patients", "Пациенты")
	vm.Patients = s.presenter.Patients(patients)

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("email"))
	}
	vm.PatientSearch = query
	vm.SearchEmail = query
	if query != "" {
		vm.Patients = filterPatientRows(vm.Patients, query)
		vm.Notice = fmt.Sprintf("Найдено пациентов: %d", len(vm.Patients))
	}

	s.render(w, "patients", vm)
}

func (s *Server) handlePatientRegister(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := r.ParseForm(); err != nil {
		s.redirect(w, r, "/patients", "", err)
		return
	}

	birthDate, err := parseDate(r.FormValue("birth-date"))
	if err != nil {
		s.redirect(w, r, "/patients", "", err)
		return
	}

	patient, err := s.services.Patients.Register(r.Context(), usecase.CreatePatientDTO{
		FullName:  strings.TrimSpace(r.FormValue("full-name")),
		Phone:     strings.TrimSpace(r.FormValue("phone")),
		Email:     strings.TrimSpace(r.FormValue("email")),
		BirthDate: birthDate,
	})
	if err != nil {
		s.redirect(w, r, "/patients", "", err)
		return
	}

	s.redirect(w, r, "/patients", "Пациент зарегистрирован: "+patient.ID.String(), nil)
}

func (s *Server) handlePatientDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := parseUUID(r.FormValue("id"), "id")
	if err != nil {
		s.redirect(w, r, "/patients", "", err)
		return
	}
	if err := s.services.Patients.Delete(r.Context(), id); err != nil {
		s.redirect(w, r, "/patients", "", err)
		return
	}
	s.redirect(w, r, "/patients", "Пациент удалён", nil)
}

func (s *Server) handleAppointments(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodGet) {
		return
	}
	vm := s.baseView(r, "appointments", "Записи")
	vm.DoctorID = valueOrDefault(r.URL.Query().Get("doctor-id"), defaultDoctorID)
	vm.PatientSearch = strings.TrimSpace(r.URL.Query().Get("patient-q"))
	vm.DoctorSearch = strings.TrimSpace(r.URL.Query().Get("doctor-q"))
	vm.DateTime = defaultDateTime()

	patients, err := s.services.Patients.GetAll(r.Context())
	if err != nil {
		s.renderError(w, r, "appointments", "Записи", err)
		return
	}
	appointments, err := s.services.Appointments.GetAll(r.Context())
	if err != nil {
		s.renderError(w, r, "appointments", "Записи", err)
		return
	}

	vm.Patients = s.presenter.Patients(patients)
	vm.Appointments = decorateAppointments(s.presenter.Appointments(appointments), vm.Patients)
	vm.Appointments = filterAppointmentRows(vm.Appointments, vm.PatientSearch, vm.DoctorSearch)
	vm.Doctors = filterDoctors(vm.DoctorSearch)
	vm.DoctorOptions = allDoctors()
	s.render(w, "appointments", vm)
}

func (s *Server) handleAppointmentCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	patientID, err := parseUUID(r.FormValue("patient-id"), "patient-id")
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	doctorID, err := parseUUID(r.FormValue("doctor-id"), "doctor-id")
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	date, err := parseDateTime(r.FormValue("date-time"))
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	app, err := s.services.Appointments.Create(r.Context(), usecase.CreateAppointmentDTO{
		PatientID: patientID,
		DoctorID:  doctorID,
		Date:      date,
	})
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	s.redirect(w, r, "/appointments", "Запись создана: "+app.ID.String(), nil)
}

func (s *Server) handleAppointmentCancel(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := parseUUID(r.FormValue("id"), "id")
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	if err := s.services.Appointments.Cancel(r.Context(), id); err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	s.redirect(w, r, "/appointments", "Запись отменена", nil)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodGet) {
		return
	}
	vm := s.baseView(r, "schedule", "Расписание")
	vm.DoctorID = valueOrDefault(r.URL.Query().Get("doctor-id"), defaultDoctorID)
	vm.DoctorSearch = strings.TrimSpace(r.URL.Query().Get("doctor-q"))
	vm.Doctors = filterDoctors(vm.DoctorSearch)
	vm.DoctorOptions = allDoctors()
	if strings.TrimSpace(r.URL.Query().Get("doctor-id")) == "" && vm.DoctorSearch != "" && len(vm.Doctors) > 0 {
		vm.DoctorID = vm.Doctors[0].ID
	}
	vm.Date = valueOrDefault(r.URL.Query().Get("date"), defaultDate())

	doctorID, err := parseUUID(vm.DoctorID, "doctor-id")
	if err != nil {
		vm.Error = userError(err)
		s.render(w, "schedule", vm)
		return
	}

	slots, err := s.services.Schedule.GetByDoctor(r.Context(), doctorID)
	if err != nil {
		s.renderError(w, r, "schedule", "Расписание", err)
		return
	}
	vm.Slots = decorateSlots(s.presenter.ScheduleSlots(slots))

	date, err := parseDate(vm.Date)
	if err == nil {
		free, err := s.services.Schedule.GetFreeSlots(r.Context(), doctorID, date)
		if err == nil {
			vm.FreeSlots = decorateSlots(s.presenter.ScheduleSlots(free))
		}
	}

	s.render(w, "schedule", vm)
}

func (s *Server) handleScheduleCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	doctorID, err := parseUUID(r.FormValue("doctor-id"), "doctor-id")
	if err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	date, err := parseDate(r.FormValue("date"))
	if err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	start, err := parseTimeOnDate(date, r.FormValue("start"))
	if err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	end, err := parseTimeOnDate(date, r.FormValue("end"))
	if err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	slot, err := s.services.Schedule.CreateSlot(r.Context(), usecase.CreateScheduleSlotDTO{
		DoctorID:  doctorID,
		Date:      date,
		StartTime: start,
		EndTime:   end,
		Cabinet:   strings.TrimSpace(r.FormValue("cabinet")),
	})
	if err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	s.redirect(w, r, "/schedule?doctor-id="+url.QueryEscape(doctorID.String())+"&date="+url.QueryEscape(date.Format(dateLayout)), "Слот создан: "+slot.ID.String(), nil)
}

func (s *Server) handleScheduleDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := parseUUID(r.FormValue("id"), "id")
	if err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	if err := s.services.Schedule.DeleteSlot(r.Context(), id); err != nil {
		s.redirect(w, r, "/schedule", "", err)
		return
	}
	s.redirect(w, r, "/schedule", "Слот удалён", nil)
}

func (s *Server) handleVisits(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodGet) {
		return
	}
	vm := s.baseView(r, "visits", "Приёмы")
	vm.PatientID = valueOrDefault(r.URL.Query().Get("patient-id"), defaultPatientID)

	appointments, err := s.services.Appointments.GetAll(r.Context())
	if err != nil {
		s.renderError(w, r, "visits", "Приёмы", err)
		return
	}
	vm.Appointments = s.presenter.Appointments(appointments)

	patientID, err := parseUUID(vm.PatientID, "patient-id")
	if err != nil {
		vm.Error = userError(err)
		s.render(w, "visits", vm)
		return
	}
	visits, err := s.services.Visits.GetByPatient(r.Context(), patientID)
	if err != nil {
		s.renderError(w, r, "visits", "Приёмы", err)
		return
	}
	vm.Visits = s.presenter.Visits(visits)
	s.render(w, "visits", vm)
}

func (s *Server) handleEmergencies(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodGet) {
		return
	}

	vm := s.baseView(r, "emergencies", "Срочные")
	vm.DateTime = defaultEmergencyDateTime()

	patients, err := s.services.Patients.GetAll(r.Context())
	if err != nil {
		s.renderError(w, r, "emergencies", "Срочные", err)
		return
	}
	requests, err := s.services.Emergencies.GetRequests(r.Context())
	if err != nil {
		s.renderError(w, r, "emergencies", "Срочные", err)
		return
	}
	offers, err := s.services.Emergencies.GetOffers(r.Context())
	if err != nil {
		s.renderError(w, r, "emergencies", "Срочные", err)
		return
	}

	vm.Patients = s.presenter.Patients(patients)
	vm.EmergencyRequests = decorateEmergencyRequests(s.presenter.EmergencyRequests(requests), vm.Patients)
	vm.RescheduleOffers = decorateRescheduleOffers(s.presenter.RescheduleOffers(offers), vm.Patients)
	s.render(w, "emergencies", vm)
}

func (s *Server) handleEmergencyCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}

	patientID, err := parseUUID(r.FormValue("patient-id"), "patient-id")
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	doctorID, err := parseUUID(r.FormValue("doctor-id"), "doctor-id")
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	requestedDate, err := parseDateTime(r.FormValue("requested-date"))
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	request, err := s.services.Emergencies.Create(r.Context(), usecase.CreateEmergencyRequestDTO{
		PatientID:     patientID,
		DoctorID:      doctorID,
		RequestedDate: requestedDate,
		Reason:        strings.TrimSpace(r.FormValue("reason")),
	})
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}

	s.redirect(w, r, "/emergencies", "Срочная заявка создана: "+request.ID.String()+" ("+string(request.Status)+")", nil)
}

func (s *Server) handleRescheduleOfferAccept(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := parseUUID(r.FormValue("id"), "id")
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	offer, err := s.services.Emergencies.AcceptOffer(r.Context(), id)
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	s.redirect(w, r, "/emergencies", "Перенос принят: "+offer.ID.String(), nil)
}

func (s *Server) handleRescheduleOfferDecline(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	id, err := parseUUID(r.FormValue("id"), "id")
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	offer, err := s.services.Emergencies.DeclineOffer(r.Context(), id)
	if err != nil {
		s.redirect(w, r, "/emergencies", "", err)
		return
	}
	s.redirect(w, r, "/emergencies", "Перенос отклонен: "+offer.ID.String(), nil)
}

func (s *Server) handleVisitStart(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	appointmentID, err := parseUUID(r.FormValue("appointment-id"), "appointment-id")
	if err != nil {
		s.redirect(w, r, "/visits", "", err)
		return
	}
	visit, err := s.services.Visits.Create(r.Context(), usecase.CreateVisitDTO{
		AppointmentID: appointmentID,
		Complaints:    strings.TrimSpace(r.FormValue("complaints")),
	})
	if err != nil {
		s.redirect(w, r, "/visits", "", err)
		return
	}
	s.redirect(w, r, "/visits", "Приём начат: "+visit.ID.String(), nil)
}

func (s *Server) handleVisitComplete(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	visitID, err := parseUUID(r.FormValue("visit-id"), "visit-id")
	if err != nil {
		s.redirect(w, r, "/visits", "", err)
		return
	}
	visit, err := s.services.Visits.Complete(r.Context(), visitID, usecase.UpdateVisitDTO{
		Diagnosis:       strings.TrimSpace(r.FormValue("diagnosis")),
		Recommendations: strings.TrimSpace(r.FormValue("recommendations")),
	})
	if err != nil {
		s.redirect(w, r, "/visits", "", err)
		return
	}
	s.redirect(w, r, "/visits", "Приём завершён: "+visit.ID.String(), nil)
}

func (s *Server) baseView(r *http.Request, active, title string) ViewModel {
	return ViewModel{
		AppName:       s.appName,
		Title:         title,
		Active:        active,
		Notice:        strings.TrimSpace(r.URL.Query().Get("notice")),
		Error:         strings.TrimSpace(r.URL.Query().Get("error")),
		Doctors:       allDoctors(),
		DoctorOptions: allDoctors(),
		DoctorID:      defaultDoctorID,
		PatientID:     defaultPatientID,
		PatientSearch: "",
		SearchEmail:   "",
		DoctorSearch:  "",
		Date:          defaultDate(),
		DateTime:      defaultDateTime(),
		AppointmentID: "",
	}
}

func (s *Server) render(w http.ResponseWriter, name string, vm ViewModel) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := parseTemplates().ExecuteTemplate(w, name, vm); err != nil {
		s.logger.Error("render gui template failed", "template", name, "error", err)
		http.Error(w, "render template", http.StatusInternalServerError)
	}
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, active, title string, err error) {
	vm := s.baseView(r, active, title)
	vm.Error = userError(err)
	s.render(w, active, vm)
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request, path string, notice string, err error) {
	values := url.Values{}
	if strings.TrimSpace(notice) != "" {
		values.Set("notice", notice)
	}
	if err != nil {
		values.Set("error", userError(err))
		s.logger.Error("gui action failed", "path", r.URL.Path, "error", err)
	} else {
		s.logger.Info("gui action completed", "path", r.URL.Path, "notice", notice)
	}
	target := path
	if encoded := values.Encode(); encoded != "" {
		sep := "?"
		if strings.Contains(target, "?") {
			sep = "&"
		}
		target += sep + encoded
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		s.logger.Info(
			"gui request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration", time.Since(start),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func parseUUID(value, field string) (uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return uuid.Nil, fmt.Errorf("флаг %s обязателен", field)
	}
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("разбор %s: %w", field, err)
	}
	return id, nil
}

func parseDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New("дата обязательна")
	}
	date, err := time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("разбор даты: %w", err)
	}
	return date, nil
}

func parseDateTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New("дата и время обязательны")
	}
	date, err := time.ParseInLocation(formTimeLayout, strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("разбор даты и времени: %w", err)
	}
	return date, nil
}

func parseTimeOnDate(date time.Time, value string) (time.Time, error) {
	clock, err := time.ParseInLocation(timeLayout, strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("разбор времени: %w", err)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), 0, 0, time.Local), nil
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func userError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, domain.ErrNotFound) {
		return "Запись не найдена"
	}
	return err.Error()
}
