package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPatientJourney(t *testing.T) {
	base := os.Getenv("API_URL")
	if os.Getenv("RUN_ID") == "" || base != "http://api-"+os.Getenv("RUN_ID")+":8080" || os.Getenv("LAB2_STAND") != "persistent" {
		t.Fatal("Use make test to start the Lab 2 stand")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	request := func(method, path string, input any, status int, output any) {
		t.Helper()
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(method, base+path, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s %s -> %d", method, path, response.StatusCode)
		if response.StatusCode != status {
			t.Fatalf("expected HTTP %d, got %d: %s", status, response.StatusCode, body)
		}
		if output != nil {
			if err := json.Unmarshal(body, output); err != nil {
				t.Fatal(err)
			}
		}
	}
	var p struct{ ID string }
	request("POST", "/api/patients", map[string]any{"full_name": "Пациент ЛР2", "phone": "+79001234567", "email": uuid.NewString() + "@example.test", "birth_date": "1990-01-01T00:00:00Z"}, 201, &p)
	if p.ID == "" {
		t.Fatal("patient ID missing")
	}
	if os.Getenv("LAB2_E2E_FAIL") == "1" {
		t.Fatal("Учебный сбой после регистрации пациента: проверка отчёта и отладки")
	}
	doctor := uuid.NewString()
	date := time.Now().UTC().Truncate(24 * time.Hour).Add(34 * time.Hour)
	var appointment struct{ ID, Status string }
	input := map[string]any{"patient_id": p.ID, "doctor_id": doctor, "date": date}
	request("POST", "/api/appointments", input, 201, &appointment)
	if appointment.ID == "" || appointment.Status != "CREATED" {
		t.Fatalf("incorrect booking: %+v", appointment)
	}
	request("POST", "/api/appointments", input, 409, nil)
	var visit struct{ ID, Status string }
	request("POST", "/api/visits", map[string]string{"appointment_id": appointment.ID, "complaints": "Головная боль"}, 201, &visit)
	if visit.ID == "" || visit.Status != "IN_PROGRESS" {
		t.Fatalf("incorrect visit: %+v", visit)
	}
	request("POST", "/api/visits/"+visit.ID+"/complete", map[string]string{"diagnosis": "Тестовый диагноз", "recommendations": "Контрольный визит"}, 200, &visit)
	if visit.Status != "COMPLETED" {
		t.Fatal("visit was not completed")
	}
	var history []struct{ ID, AppointmentID, Status, Diagnosis, Recommendations string }
	request("GET", "/api/visits?patient_id="+p.ID, nil, 200, &history)
	if len(history) != 1 || history[0].ID != visit.ID || history[0].AppointmentID != appointment.ID || history[0].Diagnosis != "Тестовый диагноз" || history[0].Recommendations != "Контрольный визит" || history[0].Status != "COMPLETED" {
		t.Fatalf("incorrect medical history: %+v", history)
	}
	var bookings []struct{ ID, Status string }
	request("GET", "/api/appointments?patient_id="+p.ID, nil, 200, &bookings)
	if len(bookings) != 1 || bookings[0].ID != appointment.ID || bookings[0].Status != "COMPLETED" {
		t.Fatalf("incorrect appointments: %+v", bookings)
	}
}
