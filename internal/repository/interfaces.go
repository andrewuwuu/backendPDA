package repository

import (
    "context"

    "pda-monitor/internal/domain"
)

type StationRepository interface {
    Upsert(ctx context.Context, station *domain.Station) error
    UpsertBatch(ctx context.Context, stations []domain.Station) error
    GetAll(ctx context.Context) ([]domain.Station, error)
    GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.Station, error)
    UpdateStatus(ctx context.Context, namaLokasi, status string) error
}

type FormulaRepository interface {
    GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.FormulaParams, error)
    GetAll(ctx context.Context) ([]domain.FormulaParams, error)
    Create(ctx context.Context, params *domain.FormulaParams) error
    Update(ctx context.Context, params *domain.FormulaParams) error
    Delete(ctx context.Context, namaLokasi string) error
}

type ReadingRepository interface {
    InsertReading(ctx context.Context, reading *domain.HourlyReading) error
    BulkInsertReadings(ctx context.Context, readings []domain.HourlyReading) error
    GetCurrentHourReadings(ctx context.Context) ([]domain.HourlyReading, error)
    GetCurrentHourByStation(ctx context.Context, namaLokasi string) ([]domain.HourlyReading, error)
    GetHourlySummary(ctx context.Context) ([]domain.HourlySummary, error)
    GetLatestReadingPerStation(ctx context.Context) ([]domain.HourlyReading, error)
    CleanupOldReadings(ctx context.Context, hoursToKeep int) (int64, error)
}