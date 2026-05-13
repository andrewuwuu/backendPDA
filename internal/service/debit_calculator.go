package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/repository"
)

type DebitCalculator struct {
	formulaRepo repository.FormulaRepository
	cache       map[string][]domain.FormulaParams
	mu          sync.RWMutex
}

func NewDebitCalculator(repo repository.FormulaRepository) *DebitCalculator {
	return &DebitCalculator{
		formulaRepo: repo,
		cache:       make(map[string][]domain.FormulaParams),
	}
}

// normalizeKey produces a canonical cache key from a station name.
func normalizeKey(namaLokasi string) string {
	return strings.TrimSpace(strings.ToLower(namaLokasi))
}

func (dc *DebitCalculator) Calculate(ctx context.Context, record domain.PDARecord) (*domain.DebitResult, error) {
	result := &domain.DebitResult{
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
	if err := dc.preloadMissingStations(ctx, records); err != nil {
		return nil, err
	}

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
	key := normalizeKey(namaLokasi)

	dc.mu.RLock()
	if formulas, ok := dc.cache[key]; ok {
		dc.mu.RUnlock()
		return formulas, nil
	}
	dc.mu.RUnlock()

	formulas, err := dc.formulaRepo.GetAllByNamaLokasi(ctx, namaLokasi)
	if err != nil {
		return nil, fmt.Errorf("formulas not found for %s: %w", namaLokasi, err)
	}

	sortFormulas(formulas)

	dc.mu.Lock()
	dc.cache[key] = formulas
	dc.mu.Unlock()

	if len(formulas) == 0 {
		return nil, fmt.Errorf("no formulas found for %s", namaLokasi)
	}

	return formulas, nil
}

func (dc *DebitCalculator) RefreshCache(ctx context.Context) error {
	cache, err := dc.loadAllParams(ctx)
	if err != nil {
		return err
	}

	dc.mu.Lock()
	dc.cache = cache
	dc.mu.Unlock()

	return nil
}

func (dc *DebitCalculator) InvalidateCache(namaLokasi string) {
	key := normalizeKey(namaLokasi)
	dc.mu.Lock()
	defer dc.mu.Unlock()
	delete(dc.cache, key)
}

func (dc *DebitCalculator) GetFormulasForStation(ctx context.Context, namaLokasi string) ([]domain.FormulaParams, error) {
	return dc.getParams(ctx, namaLokasi)
}

func (dc *DebitCalculator) preloadMissingStations(ctx context.Context, records []domain.PDARecord) error {
	if len(records) == 0 {
		return nil
	}

	missing := make(map[string]struct{})

	dc.mu.RLock()
	for _, record := range records {
		key := normalizeKey(record.NamaLokasi)
		if _, ok := dc.cache[key]; !ok {
			missing[key] = struct{}{}
		}
	}
	dc.mu.RUnlock()

	if len(missing) == 0 {
		return nil
	}

	cache, err := dc.loadAllParams(ctx)
	if err != nil {
		return err
	}

	for key := range missing {
		if _, ok := cache[key]; !ok {
			cache[key] = []domain.FormulaParams{}
		}
	}

	dc.mu.Lock()
	for key, formulas := range cache {
		dc.cache[key] = formulas
	}
	dc.mu.Unlock()

	return nil
}

func (dc *DebitCalculator) loadAllParams(ctx context.Context) (map[string][]domain.FormulaParams, error) {
	params, err := dc.formulaRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	cache := make(map[string][]domain.FormulaParams)
	for _, p := range params {
		key := normalizeKey(p.NamaLokasi)
		cache[key] = append(cache[key], p)
	}

	for key := range cache {
		sortFormulas(cache[key])
	}

	return cache, nil
}

func sortFormulas(formulas []domain.FormulaParams) {
	sort.Slice(formulas, func(i, j int) bool {
		return formulas[i].Priority > formulas[j].Priority
	})
}
