package domain

import "errors"

var ErrInvalidInput = errors.New("invalid input")
var ErrReportTooLarge = errors.New("Слишком большой отчёт. Сократите период и повторите запрос")
