package domain

import (
	"errors"
	"time"
)

// Типы согласий, которые фиксирует система.
const (
	ConsentPersonalData = "personal_data" // обработка обычных персональных данных
	ConsentBiometric    = "biometric"     // обработка биометрических персональных данных
)

// Версии текстов согласий. При изменении формулировок текста согласия
// необходимо поднять версию — тогда в журнале видно, на какую редакцию
// согласился пользователь. Должны совпадать с версиями во фронтенде
// (frontend/src/legal/consent.ts).
const (
	PersonalDataConsentVersion = "2026-05-20"
	BiometricConsentVersion    = "2026-05-20"
)

// ErrBiometricConsentRequired возвращается при попытке загрузить фотографии
// лица без действующего согласия на обработку биометрических данных.
var ErrBiometricConsentRequired = errors.New("biometric consent required")

// Consent — одна запись в журнале согласий пользователя.
type Consent struct {
	ID         int
	ISU        string
	Type       string
	DocVersion string
	AcceptedAt time.Time
	RevokedAt  *time.Time // nil — согласие действует
	IPAddress  string
	UserAgent  string
}
