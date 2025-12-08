package service

import (
    "context"
    "fmt"
    "math"
    "sync"
    "time"

    "pda-monitor/internal/domain"
    "pda-monitor/internal/repository"
)

type DebitCalculator struct {
    formulaRepo repository.FormulaRepository
    cache       map[string]*domain.FormulaParams
    mu          sync.RWMutex
}

func NewDebitCalculator(repo repository.FormulaRepository) *DebitCalculator {
    return &DebitCalculator{
        formulaRepo: repo,
        cache:       make(map[string]*domain.FormulaParams),
    }
}

func (dc *DebitCalculator) Calculate(ctx context.Context, record domain.PDARecord) (*domain.DebitResult, error) {
    result := &domain.DebitResult{
        NamaLokasi:   record.NamaLokasi,
        NamaAlat:     record.NamaAlat,
        WLevel:       record.WLevel,
        TMA:          record.TMA,
        Status:       record.Status,
        Sungai:       record.Sungai,
        Lat:          record.Lat,
        Lng:          record.Lng,
        RecordedAt:   record.RecordedAt,
        CalculatedAt: time.Now(),
    }

    params, err := dc.getParams(ctx, record.NamaLokasi)
    if err != nil {
        result.IsValid = false
        result.Debit = 0
        return result, nil
    }

    if !params.Validate(record.TMA) {
        result.IsValid = false
        result.Debit = 0
        return result, nil
    }

    base := record.TMA - params.H0

    if base < 0 {
        result.IsValid = false
        result.Debit = 0
        return result, nil
    }

    result.Debit = params.C * math.Pow(base, params.B)
    result.IsValid = true

    return result, nil
}

func (dc *DebitCalculator) CalculateBatch(ctx context.Context, records []domain.PDARecord) ([]domain.DebitResult, error) {
    results := make([]domain.DebitResult, 0, len(records))

    for _, record := range records {
        result, err := dc.Calculate(ctx, record)
        if err != nil {
            return nil, err
        }
        results = append(results, *result)
    }

    return results, nil
}

func (dc *DebitCalculator) getParams(ctx context.Context, namaLokasi string) (*domain.FormulaParams, error) {
    dc.mu.RLock()
    if params, ok := dc.cache[namaLokasi]; ok {
        dc.mu.RUnlock()
        return params, nil
    }
    dc.mu.RUnlock()

    params, err := dc.formulaRepo.GetByNamaLokasi(ctx, namaLokasi)
    if err != nil {
        return nil, fmt.Errorf("formula not found for %s: %w", namaLokasi, err)
    }

    dc.mu.Lock()
    dc.cache[namaLokasi] = params
    dc.mu.Unlock()

    return params, nil
}

func (dc *DebitCalculator) RefreshCache(ctx context.Context) error {
    params, err := dc.formulaRepo.GetAll(ctx)
    if err != nil {
        return err
    }

    dc.mu.Lock()
    defer dc.mu.Unlock()

    dc.cache = make(map[string]*domain.FormulaParams)
    for i := range params {
        dc.cache[params[i].NamaLokasi] = &params[i]
    }

    return nil
}

func (dc *DebitCalculator) InvalidateCache(namaLokasi string) {
    dc.mu.Lock()
    defer dc.mu.Unlock()
    delete(dc.cache, namaLokasi)
}