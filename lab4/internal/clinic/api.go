package clinic

import (
	"context"
	"net/http"
	"time"
)

type Registration struct {
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	BirthDate time.Time `json:"birth_date"`
}

type appointmentRequest struct {
	PatientID string    `json:"patient_id"`
	DoctorID  string    `json:"doctor_id"`
	Date      time.Time `json:"date"`
}

type visitRequest struct {
	AppointmentID string `json:"appointment_id"`
	Complaints    string `json:"complaints"`
}

type conclusion struct {
	Diagnosis       string `json:"diagnosis"`
	Recommendations string `json:"recommendations"`
}

func (c *Client) Register(ctx context.Context, patient Registration) string {
	return c.created(ctx, call{operation: Register, method: http.MethodPost, path: "/api/patients", body: patient, expect: []int{201}})
}

func (c *Client) OpenProfile(ctx context.Context, patientID string) bool {
	return c.succeeded(ctx, call{operation: Login, method: http.MethodGet, path: "/api/patients/" + patientID, expect: []int{200}})
}

func (c *Client) FreeSlots(ctx context.Context, doctorID string, day time.Time) bool {
	path := "/api/schedule/free?doctor_id=" + doctorID + "&date=" + day.Format(time.DateOnly)
	return c.succeeded(ctx, call{operation: BrowseSchedule, method: http.MethodGet, path: path, expect: []int{200}})
}

func (c *Client) Book(ctx context.Context, patientID, doctorID string, at time.Time) string {
	body := appointmentRequest{PatientID: patientID, DoctorID: doctorID, Date: at}
	return c.created(ctx, call{operation: Book, method: http.MethodPost, path: "/api/appointments", body: body, expect: []int{201}})
}

func (c *Client) Appointments(ctx context.Context, patientID string) bool {
	path := "/api/appointments?patient_id=" + patientID
	return c.succeeded(ctx, call{operation: MyAppointments, method: http.MethodGet, path: path, expect: []int{200}})
}

func (c *Client) Cancel(ctx context.Context, appointmentID string) bool {
	path := "/api/appointments/" + appointmentID + "/cancel"
	return c.succeeded(ctx, call{operation: Cancel, method: http.MethodPost, path: path, expect: []int{200}})
}

func (c *Client) StartVisit(ctx context.Context, appointmentID, complaints string) string {
	body := visitRequest{AppointmentID: appointmentID, Complaints: complaints}
	return c.created(ctx, call{operation: StartVisit, method: http.MethodPost, path: "/api/visits", body: body, expect: []int{201}})
}

func (c *Client) CompleteVisit(ctx context.Context, visitID, diagnosis, recommendations string) bool {
	body := conclusion{Diagnosis: diagnosis, Recommendations: recommendations}
	path := "/api/visits/" + visitID + "/complete"
	return c.succeeded(ctx, call{operation: CompleteVisit, method: http.MethodPost, path: path, body: body, expect: []int{200}})
}

func (c *Client) History(ctx context.Context, patientID string) bool {
	path := "/api/visits?patient_id=" + patientID
	return c.succeeded(ctx, call{operation: VisitHistory, method: http.MethodGet, path: path, expect: []int{200}})
}
