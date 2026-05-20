package consent

// ConsentItem — одна запись журнала согласий в ответе API.
type ConsentItem struct {
	Type       string  `json:"type"`
	DocVersion string  `json:"doc_version"`
	AcceptedAt string  `json:"accepted_at"`
	RevokedAt  *string `json:"revoked_at,omitempty"`
	Active     bool    `json:"active"`
}

// ConsentsResponse — список всех согласий пользователя.
type ConsentsResponse struct {
	ISU      string        `json:"isu"`
	Consents []ConsentItem `json:"consents"`
}
