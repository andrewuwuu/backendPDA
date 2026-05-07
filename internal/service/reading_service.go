package service

import (
	"context"
	"fmt"
	"time"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/repository"
	"pda-monitor/internal/timeutil"
)

type ReadingService struct {
	readingRepo repository.ReadingRepository
	calculator  *DebitCalculator
}

func NewReadingService(
	readingRepo repository.ReadingRepository,
	calculator *DebitCalculator,
) *ReadingService {
	return &ReadingService{
		readingRepo: readingRepo,
		calculator:  calculator,
	}
}

func (s *ReadingService) ProcessAndStoreReadings(ctx context.Context, records []domain.PDARecord) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	results, err := s.calculator.CalculateBatch(ctx, records)
	if err != nil {
		return 0, fmt.Errorf("calculate batch debit: %w", err)
	}

	readings := make([]domain.HourlyReading, 0, len(records))
	for i, record := range records {
		result := results[i]
		var debit *float64
		if result.IsValid {
			debit = &result.Debit
		}

		readings = append(readings, domain.HourlyReading{
			NamaLokasi: record.NamaLokasi,
			HourBucket: timeutil.TruncateToJakartaHour(record.RecordedAt),
			RecordedAt: record.RecordedAt,
			WLevel:     record.WLevel,
			TMA:        record.TMA,
			Debit:      debit,
			IsValid:    result.IsValid,
			Rain:       record.Rain,
		})
	}

	if err := s.readingRepo.BulkInsertReadings(ctx, readings); err != nil {
		return 0, err
	}

	return len(readings), nil
}

func (s *ReadingService) GetCurrentHourData(ctx context.Context) ([]domain.HourlyReading, error) {
	return s.readingRepo.GetCurrentHourReadings(ctx)
}

func (s *ReadingService) GetCurrentHourByStation(ctx context.Context, namaLokasi string) ([]domain.HourlyReading, error) {
	return s.readingRepo.GetCurrentHourByStation(ctx, namaLokasi)
}

func (s *ReadingService) GetCurrentHourSummary(ctx context.Context) ([]domain.HourlySummary, error) {
	return s.readingRepo.GetHourlySummary(ctx)
}

func (s *ReadingService) GetLatestReadings(ctx context.Context) ([]domain.HourlyReading, error) {
	return s.readingRepo.GetLatestReadingPerStation(ctx)
}

func (s *ReadingService) CleanupOldData(ctx context.Context, hoursToKeep int) (int64, error) {
	return s.readingRepo.CleanupOldReadings(ctx, hoursToKeep)
}

func (s *ReadingService) GetDailyTMASummary(ctx context.Context, date time.Time) (map[string]domain.TMARangeSummary, error) {
	summaries, err := s.readingRepo.GetTMARangeForDay(ctx, date, 7, 17)
	if err != nil {
		return nil, err
	}

	result := make(map[string]domain.TMARangeSummary)
	for _, summary := range summaries {
		result[summary.NamaLokasi] = summary
	}

	return result, nil
}

func (s *ReadingService) GetDebitSnapshots(ctx context.Context, date time.Time, hours []int) (map[string]map[int]*float64, error) {
	snapshots, err := s.readingRepo.GetDebitAtHours(ctx, date, hours)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[int]*float64)
	for _, snap := range snapshots {
		if result[snap.NamaLokasi] == nil {
			result[snap.NamaLokasi] = make(map[int]*float64)
		}
		result[snap.NamaLokasi][snap.Hour] = snap.Debit
	}

	return result, nil
}

func (s *ReadingService) GetReadingsByTimeRange(ctx context.Context, namaLokasi string, from, to time.Time) ([]domain.HourlyReading, error) {
	return s.readingRepo.GetReadingsByTimeRange(ctx, namaLokasi, from, to)
}

func (s *ReadingService) GetAllReadingsByTimeRange(ctx context.Context, from, to time.Time) ([]domain.HourlyReading, error) {
	return s.readingRepo.GetAllReadingsByTimeRange(ctx, from, to)
}

func (s *ReadingService) RecalculateDebit(ctx context.Context, namaLokasi string) error {
	readings, err := s.readingRepo.GetCurrentHourByStation(ctx, namaLokasi)
	if err != nil {
		return err
	}

	for i := range readings {
		debit, valid, _ := s.calculator.CalculateWithTMA(ctx, readings[i].NamaLokasi, readings[i].TMA)
		if valid {
			readings[i].Debit = &debit
			readings[i].IsValid = true
		} else {
			readings[i].Debit = nil
			readings[i].IsValid = false
		}
	}

	return s.readingRepo.BulkInsertReadings(ctx, readings)
}
