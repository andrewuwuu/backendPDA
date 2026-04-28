package service

import (
	"context"
	"testing"
	"time"

	"pda-monitor/internal/domain"
)

type fakeReadingRepo struct {
	readings []domain.HourlyReading
	saved    []domain.HourlyReading
}

func (r *fakeReadingRepo) InsertReading(ctx context.Context, reading *domain.HourlyReading) error {
	r.saved = append(r.saved, *reading)
	return nil
}

func (r *fakeReadingRepo) BulkInsertReadings(ctx context.Context, readings []domain.HourlyReading) error {
	r.saved = append([]domain.HourlyReading(nil), readings...)
	return nil
}

func (r *fakeReadingRepo) GetCurrentHourReadings(ctx context.Context) ([]domain.HourlyReading, error) {
	return r.readings, nil
}

func (r *fakeReadingRepo) GetCurrentHourByStation(ctx context.Context, namaLokasi string) ([]domain.HourlyReading, error) {
	var result []domain.HourlyReading
	for _, reading := range r.readings {
		if reading.NamaLokasi == namaLokasi {
			result = append(result, reading)
		}
	}
	return result, nil
}

func (r *fakeReadingRepo) GetHourlySummary(ctx context.Context) ([]domain.HourlySummary, error) {
	return nil, nil
}

func (r *fakeReadingRepo) GetLatestReadingPerStation(ctx context.Context) ([]domain.HourlyReading, error) {
	return nil, nil
}

func (r *fakeReadingRepo) CleanupOldReadings(ctx context.Context, hoursToKeep int) (int64, error) {
	return 0, nil
}

func (r *fakeReadingRepo) GetTMARangeForDay(ctx context.Context, date time.Time, startHour, endHour int) ([]domain.TMARangeSummary, error) {
	return nil, nil
}

func (r *fakeReadingRepo) GetDebitAtHours(ctx context.Context, date time.Time, hours []int) ([]domain.HourlyDebitSnapshot, error) {
	return nil, nil
}

func (r *fakeReadingRepo) GetReadingsByTimeRange(ctx context.Context, namaLokasi string, from, to time.Time) ([]domain.HourlyReading, error) {
	return nil, nil
}

func (r *fakeReadingRepo) GetAllReadingsByTimeRange(ctx context.Context, from, to time.Time) ([]domain.HourlyReading, error) {
	return nil, nil
}

func TestReadingServiceRecalculateDebitUpdatesCurrentHourRows(t *testing.T) {
	formulas := &fakeFormulaRepo{formulas: map[string][]domain.FormulaParams{
		"pda-test": {
			{
				NamaLokasi:      "pda-test",
				C:               12,
				H0:              0.25,
				B:               2,
				TMAMin:          0,
				TMAMinInclusive: true,
				TMAMax:          2,
				TMAMaxInclusive: true,
				Priority:        1,
			},
		},
	}}
	readings := &fakeReadingRepo{readings: []domain.HourlyReading{
		{
			NamaLokasi: "pda-test",
			TMA:        1.25,
			WLevel:     125,
			RecordedAt: time.Now(),
		},
	}}

	service := NewReadingService(readings, NewDebitCalculator(formulas))
	if err := service.RecalculateDebit(context.Background(), "pda-test"); err != nil {
		t.Fatalf("RecalculateDebit returned error: %v", err)
	}

	if len(readings.saved) != 1 {
		t.Fatalf("saved readings = %d, want 1", len(readings.saved))
	}
	if !readings.saved[0].IsValid {
		t.Fatal("expected recalculated reading to be valid")
	}
	if readings.saved[0].Debit == nil {
		t.Fatal("expected recalculated reading to have debit")
	}
	if *readings.saved[0].Debit != 12 {
		t.Fatalf("debit = %v, want 12", *readings.saved[0].Debit)
	}
}
