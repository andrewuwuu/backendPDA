package service

import (
    "context"
    "log"
    "time"

    "pda-monitor/internal/domain"
    "pda-monitor/internal/repository"
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

    now := time.Now()
    hourBucket := truncateToHour(now)

    readings := make([]domain.HourlyReading, 0, len(records))

    for _, record := range records {
        result, err := s.calculator.Calculate(ctx, record)
        if err != nil {
            log.Printf("Error calculating debit for %s: %v", record.NamaLokasi, err)
            continue
        }

        var debit *float64
        if result.IsValid {
            debit = &result.Debit
        }

        readings = append(readings, domain.HourlyReading{
            NamaLokasi: record.NamaLokasi,
            HourBucket: hourBucket,
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

func truncateToHour(t time.Time) time.Time {
    return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}