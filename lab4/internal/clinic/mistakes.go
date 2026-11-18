package clinic

import (
	"context"
	"net/http"
	"time"
)

const unknownID = "00000000-0000-4000-8000-0000000dead0"

type Identity struct {
	PatientID string
	DoctorID  string
}

type mistake func(ctx context.Context, c *Client, who Identity)

var mistakes = map[Operation]mistake{
	Register: func(ctx context.Context, c *Client, _ Identity) {
		body := Registration{FullName: "Нагрузочный Пациент", Phone: "+79001234567", Email: "не-адрес", BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}
		c.wrong(ctx, Register, http.MethodPost, "/api/patients", body, http.StatusBadRequest)
	},
	Login: func(ctx context.Context, c *Client, _ Identity) {
		c.wrong(ctx, Login, http.MethodGet, "/api/patients/"+unknownID, nil, http.StatusNotFound)
	},
	BrowseSchedule: func(ctx context.Context, c *Client, who Identity) {
		c.wrong(ctx, BrowseSchedule, http.MethodGet, "/api/schedule/free?doctor_id="+who.DoctorID+"&date=", nil, http.StatusBadRequest)
	},
	Book: func(ctx context.Context, c *Client, who Identity) {
		body := appointmentRequest{PatientID: who.PatientID, DoctorID: who.DoctorID, Date: time.Now().Add(-24 * time.Hour)}
		c.wrong(ctx, Book, http.MethodPost, "/api/appointments", body, http.StatusConflict)
	},
	MyAppointments: func(ctx context.Context, c *Client, _ Identity) {
		c.wrong(ctx, MyAppointments, http.MethodPost, "/api/appointments/"+unknownID+"/cancel", nil, http.StatusNotFound)
	},
	Cancel: func(ctx context.Context, c *Client, _ Identity) {
		c.wrong(ctx, Cancel, http.MethodPost, "/api/appointments/"+unknownID+"/cancel", nil, http.StatusNotFound)
	},
	StartVisit: func(ctx context.Context, c *Client, _ Identity) {
		body := visitRequest{AppointmentID: unknownID, Complaints: "Ошибка ввода"}
		c.wrong(ctx, StartVisit, http.MethodPost, "/api/visits", body, http.StatusNotFound)
	},
	CompleteVisit: func(ctx context.Context, c *Client, _ Identity) {
		body := conclusion{Diagnosis: "Ошибка ввода", Recommendations: "Нет"}
		c.wrong(ctx, CompleteVisit, http.MethodPost, "/api/visits/"+unknownID+"/complete", body, http.StatusNotFound)
	},
	VisitHistory: func(ctx context.Context, c *Client, _ Identity) {
		c.wrong(ctx, VisitHistory, http.MethodGet, "/api/visits?patient_id=", nil, http.StatusBadRequest)
	},
}

func (c *Client) Mistake(ctx context.Context, operation Operation, who Identity) {
	mistakes[operation](ctx, c, who)
}

func (c *Client) wrong(ctx context.Context, operation Operation, method, path string, body any, status int) {
	c.send(ctx, call{operation: operation, method: method, path: path, body: body, expect: []int{status}, invalid: true})
}
