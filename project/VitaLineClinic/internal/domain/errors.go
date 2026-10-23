package domain

import "errors"

var (
	ErrNotFound            = errors.New("сущность не найдена")
	ErrSlotBusy            = errors.New("выбранный временной слот уже занят")
	ErrInvalidStatus       = errors.New("недопустимый статус для операции")
	ErrPastDate            = errors.New("нельзя записаться в прошлое")
	ErrAppointmentLeadTime = errors.New("запись должна быть создана заранее согласно настройкам приложения")
	ErrValidation          = errors.New("ошибка валидации")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "валидация: поле '" + e.Field + "' — " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}
