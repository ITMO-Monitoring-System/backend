package visits

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestReportPeriodValidation(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	for _, query := range []string{
		"date_from=2026-10-26&date_to=2026-10-25", "date_from=2024-01-01&date_to=2026-01-01",
		"date_from=invalid", "date_to=invalid", "gap_seconds=4", "gap_seconds=601", "gap_seconds=abc",
	} {
		t.Run(query, func(t *testing.T) {
			if _, err := parseTeacherReportFilter(httptest.NewRequest("GET", "/?"+query, nil), now); err == nil {
				t.Fatal("invalid filter was accepted")
			}
		})
	}
	f, err := parseTeacherReportFilter(httptest.NewRequest("GET", "/?date_from=2026-10-25&date_to=2026-10-25&gap_seconds=300", nil), now)
	if err != nil {
		t.Fatal(err)
	}
	if f.DateTo.Sub(f.DateFrom) != 24*time.Hour || f.GapSeconds != 300 {
		t.Fatalf("last day is not fully included: %+v", f)
	}
	f, err = parseTeacherReportFilter(httptest.NewRequest("GET", "/?date_from=2026-10-25T00:00:00%2B03:00&date_to=2026-10-26T00:00:00%2B03:00", nil), now)
	if err != nil || f.DateFrom.UTC().Hour() != 21 || f.DateTo.Sub(f.DateFrom) != 24*time.Hour {
		t.Fatalf("RFC3339 timezone bounds: %+v %v", f, err)
	}
}
