package clinic

import (
	"crypto/rand"
	"fmt"
	"time"

	"lab4/internal/load"
)

const (
	seededDoctor     = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	slotLength       = 30 * time.Minute
	cancelShare      = 0.3
	visitsPerPatient = 5

	thinkMin = 300 * time.Millisecond
	thinkMax = 1000 * time.Millisecond
	restMin  = 1000 * time.Millisecond
	restMax  = 2000 * time.Millisecond
)

type Patient struct {
	api        *Client
	errorRate  float64
	who        Identity
	firstDay   time.Time
	slot       int
	visitsLeft int
}

func NewPatient(api *Client, vu int, errorRate float64) *Patient {
	return &Patient{
		api:       api,
		errorRate: errorRate,
		who:       Identity{DoctorID: fmt.Sprintf("d0c70000-0000-4000-8000-%012d", vu)},
		firstDay:  time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour),
	}
}

func (p *Patient) Iterate(vu *load.VU) {
	if p.visitsLeft > 0 || p.signUp(vu) {
		p.visitClinic(vu)
		p.visitsLeft--
	}
	vu.Pause(restMin, restMax)
}

func (p *Patient) signUp(vu *load.VU) bool {
	p.who.PatientID = p.create(vu, Register, func() string {
		return p.api.Register(vu.Context(), newRegistration())
	})
	if p.who.PatientID == "" {
		return false
	}
	p.visitsLeft = visitsPerPatient
	return p.think(vu)
}

func (p *Patient) visitClinic(vu *load.VU) {
	ctx := vu.Context()
	patientID := p.who.PatientID

	p.step(vu, Login, func() bool { return p.api.OpenProfile(ctx, patientID) })
	if !p.think(vu) {
		return
	}
	p.step(vu, BrowseSchedule, func() bool { return p.api.FreeSlots(ctx, seededDoctor, p.firstDay) })
	if !p.think(vu) {
		return
	}
	appointmentID := p.create(vu, Book, func() string {
		return p.api.Book(ctx, patientID, p.who.DoctorID, p.nextSlot())
	})
	if !p.think(vu) {
		return
	}
	p.step(vu, MyAppointments, func() bool { return p.api.Appointments(ctx, patientID) })
	if appointmentID != "" && !(p.think(vu) && p.settle(vu, appointmentID)) {
		return
	}
	if p.think(vu) {
		p.step(vu, VisitHistory, func() bool { return p.api.History(ctx, patientID) })
	}
}

func (p *Patient) settle(vu *load.VU, appointmentID string) bool {
	ctx := vu.Context()
	if vu.Chance(cancelShare) {
		p.step(vu, Cancel, func() bool { return p.api.Cancel(ctx, appointmentID) })
		return true
	}
	visitID := p.create(vu, StartVisit, func() string {
		return p.api.StartVisit(ctx, appointmentID, "Головная боль")
	})
	if visitID == "" {
		return true
	}
	if !p.think(vu) {
		return false
	}
	p.step(vu, CompleteVisit, func() bool { return p.api.CompleteVisit(ctx, visitID, "ОРВИ", "Обильное питьё") })
	return true
}

func (p *Patient) step(vu *load.VU, operation Operation, action func() bool) bool {
	if p.mistaken(vu, operation) {
		return false
	}
	return action()
}

func (p *Patient) create(vu *load.VU, operation Operation, action func() string) string {
	if p.mistaken(vu, operation) {
		return ""
	}
	return action()
}

func (p *Patient) mistaken(vu *load.VU, operation Operation) bool {
	if !vu.Chance(p.errorRate) {
		return false
	}
	p.api.Mistake(vu.Context(), operation, p.who)
	return true
}

func (p *Patient) think(vu *load.VU) bool {
	return vu.Pause(thinkMin, thinkMax)
}

func (p *Patient) nextSlot() time.Time {
	at := p.firstDay.Add(time.Duration(p.slot) * slotLength)
	p.slot++
	return at
}

func newRegistration() Registration {
	return Registration{
		FullName:  "Нагрузочный Пациент",
		Phone:     "+79001234567",
		Email:     rand.Text() + "@load.test",
		BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
