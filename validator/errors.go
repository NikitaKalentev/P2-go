package validator

import "fmt"

var errs = []error{}

// Функция для обновления списка с ошибками
func pushErr(err error) {
	errs = append(errs, err)
}

// Несоответствие типа данных
func NewTypeMismatchError(key, mustBe string, line int) error {
	return fmt.Errorf("%s:%d %s must be %s", path, line, key, mustBe)
}

// Обязательное поле отсутствует
func NewRequiredFieldError(key string) error {
	return fmt.Errorf("%s is required", key)
}

// Обязательное поле отсутствует с номером строки
func NewRequiredFieldErrorWithLine(key string, line int) error {
	return fmt.Errorf("%s:%d %s is required", path, line, key)
}

// Числовое значение за пределами разрешённых значений
func NewValueOutOfRangeError(key string, line int) error {
	return fmt.Errorf("%s:%d %s value out of range", path, line, key)
}

// Неправильный формат строки
func NewInvalidFormatError(key, value string, line int) error {
	return fmt.Errorf("%s:%d %s has invalid format '%s'", path, line, key, value)
}

// Неправильное значение в поле с ограниченным набором разрешённых значений
func NewUnsupportedValueError(key, value string, line int) error {
	return fmt.Errorf("%s:%d %s has unsupported value '%s'", path, line, key, value)