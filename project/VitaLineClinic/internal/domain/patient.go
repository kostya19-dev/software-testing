package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegexp = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
)

type Patient struct {
	ID        uuid.UUID
	FullName  string
	Phone     string
	Email     string
	BirthDate time.Time
	CreatedAt time.Time
}

func (p *Patient) Validate() error {
	if strings.TrimSpace(p.FullName) == "" {
		return NewValidationError("FullName", "ФИО не может быть пустым")
	}
	if !phoneRegexp.MatchString(p.Phone) {
		return NewValidationError("Phone", "должен содержать 7–15 цифр; допускается ведущий «+»")
	}
	if !emailRegexp.MatchString(p.Email) {
		return NewValidationError("Email", "некорректный формат адреса электронной почты")
	}
	return nil
}
