package gui

import "strings"

var clinicDoctors = []DoctorRow{
	{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		FullName:  "Павлова Елена Сергеевна",
		Specialty: "Терапевт",
		Cabinet:   "101",
	},
	{
		ID:        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		FullName:  "Орлов Андрей Михайлович",
		Specialty: "Кардиолог",
		Cabinet:   "202",
	},
	{
		ID:        "cccccccc-cccc-cccc-cccc-cccccccccccc",
		FullName:  "Белова Наталья Игоревна",
		Specialty: "Невролог",
		Cabinet:   "305",
	},
	{
		ID:        "dddddddd-dddd-dddd-dddd-dddddddddddd",
		FullName:  "Громов Павел Олегович",
		Specialty: "Эндокринолог",
		Cabinet:   "410",
	},
	{
		ID:        "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		FullName:  "Ким Виктория Романовна",
		Specialty: "Педиатр",
		Cabinet:   "118",
	},
	{
		ID:        "ffffffff-ffff-ffff-ffff-ffffffffffff",
		FullName:  "Савин Илья Максимович",
		Specialty: "Ортопед",
		Cabinet:   "214",
	},
}

func allDoctors() []DoctorRow {
	rows := make([]DoctorRow, len(clinicDoctors))
	copy(rows, clinicDoctors)
	return rows
}

func filterDoctors(query string) []DoctorRow {
	query = strings.TrimSpace(query)
	if query == "" {
		return allDoctors()
	}
	rows := make([]DoctorRow, 0, len(clinicDoctors))
	for _, doctor := range clinicDoctors {
		if doctorMatches(doctor, query) {
			rows = append(rows, doctor)
		}
	}
	return rows
}

func doctorByID(id string) (DoctorRow, bool) {
	for _, doctor := range clinicDoctors {
		if strings.EqualFold(doctor.ID, strings.TrimSpace(id)) {
			return doctor, true
		}
	}
	return DoctorRow{}, false
}

func doctorMatches(doctor DoctorRow, query string) bool {
	return containsFold(doctor.ID, query) ||
		containsFold(doctor.FullName, query) ||
		containsFold(doctor.Specialty, query) ||
		containsFold(doctor.Cabinet, query)
}

func decorateAppointments(rows []AppointmentRow, patients []PatientRow) []AppointmentRow {
	patientByID := make(map[string]PatientRow, len(patients))
	for _, patient := range patients {
		patientByID[patient.ID] = patient
	}
	for i := range rows {
		if patient, ok := patientByID[rows[i].PatientID]; ok {
			rows[i].PatientName = patient.FullName
			rows[i].PatientEmail = patient.Email
		}
		if doctor, ok := doctorByID(rows[i].DoctorID); ok {
			rows[i].DoctorName = doctor.FullName
			rows[i].DoctorSpecialty = doctor.Specialty
		}
	}
	return rows
}

func decorateSlots(rows []ScheduleSlotRow) []ScheduleSlotRow {
	for i := range rows {
		if doctor, ok := doctorByID(rows[i].DoctorID); ok {
			rows[i].DoctorName = doctor.FullName
			rows[i].DoctorSpecialty = doctor.Specialty
		}
	}
	return rows
}

func decorateEmergencyRequests(rows []EmergencyRequestRow, patients []PatientRow) []EmergencyRequestRow {
	patientByID := make(map[string]PatientRow, len(patients))
	for _, patient := range patients {
		patientByID[patient.ID] = patient
	}
	for i := range rows {
		if patient, ok := patientByID[rows[i].PatientID]; ok {
			rows[i].PatientName = patient.FullName
			rows[i].PatientEmail = patient.Email
		}
		if doctor, ok := doctorByID(rows[i].DoctorID); ok {
			rows[i].DoctorName = doctor.FullName
			rows[i].DoctorSpecialty = doctor.Specialty
		}
	}
	return rows
}

func decorateRescheduleOffers(rows []RescheduleOfferRow, patients []PatientRow) []RescheduleOfferRow {
	patientByID := make(map[string]PatientRow, len(patients))
	for _, patient := range patients {
		patientByID[patient.ID] = patient
	}
	for i := range rows {
		if patient, ok := patientByID[rows[i].PatientID]; ok {
			rows[i].PatientName = patient.FullName
			rows[i].PatientEmail = patient.Email
		}
		if doctor, ok := doctorByID(rows[i].DoctorID); ok {
			rows[i].DoctorName = doctor.FullName
			rows[i].DoctorSpecialty = doctor.Specialty
		}
	}
	return rows
}

func filterPatientRows(rows []PatientRow, query string) []PatientRow {
	query = strings.TrimSpace(query)
	if query == "" {
		return rows
	}
	filtered := make([]PatientRow, 0, len(rows))
	for _, row := range rows {
		if containsFold(row.FullName, query) ||
			containsFold(row.Phone, query) ||
			containsFold(row.Email, query) ||
			containsFold(row.ID, query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterAppointmentRows(rows []AppointmentRow, patientQuery, doctorQuery string) []AppointmentRow {
	patientQuery = strings.TrimSpace(patientQuery)
	doctorQuery = strings.TrimSpace(doctorQuery)
	if patientQuery == "" && doctorQuery == "" {
		return rows
	}
	filtered := make([]AppointmentRow, 0, len(rows))
	for _, row := range rows {
		if patientQuery != "" && !appointmentPatientMatches(row, patientQuery) {
			continue
		}
		if doctorQuery != "" && !appointmentDoctorMatches(row, doctorQuery) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func appointmentPatientMatches(row AppointmentRow, query string) bool {
	return containsFold(row.PatientID, query) ||
		containsFold(row.PatientName, query) ||
		containsFold(row.PatientEmail, query)
}

func appointmentDoctorMatches(row AppointmentRow, query string) bool {
	return containsFold(row.DoctorID, query) ||
		containsFold(row.DoctorName, query) ||
		containsFold(row.DoctorSpecialty, query)
}

func containsFold(value, query string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(value, query)
}
