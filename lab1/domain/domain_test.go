package domain_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/lab1/testutil"
)

func TestPatientPhoneBoundaries(t *testing.T) {
	for _, n := range []int{6, 7, 15, 16} {
		t.Run("digits_"+strconv.Itoa(n), func(t *testing.T) {
			p := testutil.NewPatientBuilder().WithPhone(strings.Repeat("1", n)).Build()
			err := p.Validate()
			if (err == nil) != (n >= 7 && n <= 15) {
				t.Fatalf("length %d: %v", n, err)
			}
			if err != nil {
				var validation *domain.ValidationError
				if !errors.As(err, &validation) || validation.Field != "Phone" {
					t.Fatalf("typed error: %v", err)
				}
			}
		})
	}
}

func TestPatientValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := testutil.Patient()
		err := p.Validate()
		if err != nil {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("FullName", func(t *testing.T) {
		p := testutil.Patient()
		p.FullName = " "
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("Phone", func(t *testing.T) {
		p := testutil.Patient()
		p.Phone = "bad"
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("Email", func(t *testing.T) {
		p := testutil.Patient()
		p.Email = "bad"
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
}

func TestScheduleValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := testutil.Schedule()
		err := p.Validate()
		if err != nil {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("DoctorID", func(t *testing.T) {
		p := testutil.Schedule()
		p.DoctorID = uuid.Nil
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("StartTime", func(t *testing.T) {
		p := testutil.Schedule()
		p.StartTime = time.Time{}
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("missing_end_time", func(t *testing.T) {
		p := testutil.Schedule()
		p.EndTime = time.Time{}
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
	t.Run("equal_start_and_end", func(t *testing.T) {
		p := testutil.Schedule()
		p.EndTime = testutil.Date
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
}

func TestAppointmentCancelTransitions(t *testing.T) {
	t.Run("Created", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = domain.StatusCreated
		err := p.Cancel()
		if err != nil || p.Status != domain.StatusCancelled {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Cancelled", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = domain.StatusCancelled
		err := p.Cancel()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.StatusCancelled {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Completed", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = domain.StatusCompleted
		err := p.Cancel()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.StatusCompleted {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
}

func TestAppointmentCompleteTransitions(t *testing.T) {
	t.Run("Created", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = domain.StatusCreated
		err := p.Complete()
		if err != nil || p.Status != domain.StatusCompleted {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Cancelled", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = domain.StatusCancelled
		err := p.Complete()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.StatusCancelled {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Completed", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = domain.StatusCompleted
		err := p.Complete()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.StatusCompleted {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
}

func TestRescheduleOfferAcceptTransitions(t *testing.T) {
	t.Run("Pending", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = domain.OfferStatusPending
		err := p.Accept()
		if err != nil || p.Status != domain.OfferStatusAccepted {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Accepted", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = domain.OfferStatusAccepted
		err := p.Accept()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.OfferStatusAccepted {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Declined", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = domain.OfferStatusDeclined
		err := p.Accept()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.OfferStatusDeclined {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
}

func TestRescheduleOfferDeclineTransitions(t *testing.T) {
	t.Run("Pending", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = domain.OfferStatusPending
		err := p.Decline()
		if err != nil || p.Status != domain.OfferStatusDeclined {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Accepted", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = domain.OfferStatusAccepted
		err := p.Decline()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.OfferStatusAccepted {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
	t.Run("Declined", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = domain.OfferStatusDeclined
		err := p.Decline()
		if !errors.Is(err, domain.ErrInvalidStatus) || p.Status != domain.OfferStatusDeclined {
			t.Fatalf("status=%s err=%v", p.Status, err)
		}
	})
}

func TestVisitCompleteTransitions(t *testing.T) {
	for _, tc := range []struct {
		name, diagnosis, recommendations string
		completed                        bool
		want                             error
	}{
		{"valid", "diagnosis", "rest", false, nil}, {"completed", "diagnosis", "rest", true, domain.ErrInvalidStatus},
		{"empty_diagnosis", " ", "rest", false, domain.ErrValidation}, {"empty_recommendations", "diagnosis", " ", false, domain.ErrValidation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := testutil.Visit()
			if tc.completed {
				v.Status = domain.VisitStatusCompleted
			}
			err := v.Complete(tc.diagnosis, tc.recommendations)
			if !errors.Is(err, tc.want) {
				t.Fatalf("%v", err)
			}
			if err == nil && (v.Diagnosis != tc.diagnosis || v.Recommendations != tc.recommendations || v.Status != domain.VisitStatusCompleted) {
				t.Fatalf("%+v", v)
			}
		})
	}
}

func TestAppointmentValidate(t *testing.T) {
	t.Run("invalid_status", func(t *testing.T) {
		p := testutil.Appointment()
		p.Status = "INVALID"
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
}

func TestVisitValidate(t *testing.T) {
	t.Run("invalid_status", func(t *testing.T) {
		p := testutil.Visit()
		p.Status = "INVALID"
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
}

func TestEmergencyRequestValidate(t *testing.T) {
	t.Run("invalid_status", func(t *testing.T) {
		p := testutil.EmergencyRequest()
		p.Status = "INVALID"
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
}

func TestRescheduleOfferValidate(t *testing.T) {
	t.Run("invalid_status", func(t *testing.T) {
		p := testutil.RescheduleOffer()
		p.Status = "INVALID"
		err := p.Validate()
		if !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("unexpected validation: %v", err)
		}
	})
}
