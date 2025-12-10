package service

import (
    "context"
    "fmt"
    "math"
    "sort"
    "sync"
    "time"

    "pda-monitor/internal/domain"
    "pda-monitor/internal/repository"
)

type DebitCalculator struct {
    formulaRepo repository.FormulaRepository
    cache       map[string][]domain.FormulaParams // Changed to slice for multiple formulas
    mu          sync.RWMutex
}

func NewDebitCalculator(repo repository.FormulaRepository) *DebitCalculator {
    return &DebitCalculator{
        formulaRepo: repo,
        cache:       make(map[string][]domain.FormulaParams),
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

    formulas, err := dc.getParams(ctx, record.NamaLokasi)
    if err != nil || len(formulas) == 0 {
        result.IsValid = false
        result.Debit = 0
        return result, nil
    }

    var matchedFormula *domain.FormulaParams
    for i := range formulas {
        if formulas[i].Validate(record.TMA) {
            matchedFormula = &formulas[i]
            break
        }
    }

    if matchedFormula == nil {
        result.IsValid = false
        result.Debit = 0
        return result, nil
    }

    base := record.TMA - matchedFormula.H0

    if base < 0 {
        result.IsValid = false
        result.Debit = 0
        return result, nil
    }

    result.Debit = matchedFormula.C * math.Pow(base, matchedFormula.B)
    result.IsValid = true

    return result, nil
}

func (dc *DebitCalculator) CalculateWithTMA(ctx context.Context, namaLokasi string, tma float64) (float64, bool, error) {
    formulas, err := dc.getParams(ctx, namaLokasi)
    if err != nil || len(formulas) == 0 {
        return 0, false, nil
    }

    var matchedFormula *domain.FormulaParams
    for i := range formulas {
        if formulas[i].Validate(tma) {
            matchedFormula = &formulas[i]
            break
        }
    }

    if matchedFormula == nil {
        return 0, false, nil
    }

    base := tma - matchedFormula.H0
    if base < 0 {
        return 0, false, nil
    }

    debit := matchedFormula.C * math.Pow(base, matchedFormula.B)
    return debit, true, nil
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

func (dc *DebitCalculator) getParams(ctx context.Context, namaLokasi string) ([]domain.FormulaParams, error) {
    dc.mu.RLock()
    if formulas, ok := dc.cache[namaLokasi]; ok {
        dc.mu.RUnlock()
        return formulas, nil
    }
    dc.mu.RUnlock()

    formulas, err := dc.formulaRepo.GetAllByNamaLokasi(ctx, namaLokasi)
    if err != nil {
        return nil, fmt.Errorf("formulas not found for %s: %w", namaLokasi, err)
    }

    if len(formulas) == 0 {
        return nil, fmt.Errorf("no formulas found for %s", namaLokasi)
    }

    sort.Slice(formulas, func(i, j int) bool {
        return formulas[i].Priority > formulas[j].Priority
    })

    dc.mu.Lock()
    dc.cache[namaLokasi] = formulas
    dc.mu.Unlock()

    return formulas, nil
}

func (dc *DebitCalculator) RefreshCache(ctx context.Context) error {
    params, err := dc.formulaRepo.GetAll(ctx)
    if err != nil {
        return err
    }

    dc.mu.Lock()
    defer dc.mu.Unlock()

    dc.cache = make(map[string][]domain.FormulaParams)
    for _, p := range params {
        dc.cache[p.NamaLokasi] = append(dc.cache[p.NamaLokasi], p)
    }

    for namaLokasi := range dc.cache {
        formulas := dc.cache[namaLokasi]
        sort.Slice(formulas, func(i, j int) bool {
            return formulas[i].Priority > formulas[j].Priority
        })
        dc.cache[namaLokasi] = formulas
    }

    return nil
}

func (dc *DebitCalculator) InvalidateCache(namaLokasi string) {
    dc.mu.Lock()
    defer dc.mu.Unlock()
    delete(dc.cache, namaLokasi)
}

func (dc *DebitCalculator) GetFormulasForStation(ctx context.Context, namaLokasi string) ([]domain.FormulaParams, error) {
    return dc.getParams(ctx, namaLokasi)
}