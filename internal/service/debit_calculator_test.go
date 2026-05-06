package service

import (
	"context"
	"math"
	"testing"
	"time"

	"pda-monitor/internal/domain"
)

type fakeFormulaRepo struct {
	formulas             map[string][]domain.FormulaParams
	getAllCalls          int
	getByNamaLokasiCalls map[string]int
}

func (r *fakeFormulaRepo) GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.FormulaParams, error) {
	formulas := r.formulas[namaLokasi]
	if len(formulas) == 0 {
		return nil, nil
	}
	return &formulas[0], nil
}

func (r *fakeFormulaRepo) GetAll(ctx context.Context) ([]domain.FormulaParams, error) {
	r.getAllCalls++
	var all []domain.FormulaParams
	for _, formulas := range r.formulas {
		all = append(all, formulas...)
	}
	return all, nil
}

func (r *fakeFormulaRepo) Create(ctx context.Context, params *domain.FormulaParams) error {
	return nil
}

func (r *fakeFormulaRepo) Update(ctx context.Context, params *domain.FormulaParams) error {
	return nil
}

func (r *fakeFormulaRepo) Delete(ctx context.Context, namaLokasi string) error {
	return nil
}

func (r *fakeFormulaRepo) GetAllByNamaLokasi(ctx context.Context, namaLokasi string) ([]domain.FormulaParams, error) {
	if r.getByNamaLokasiCalls == nil {
		r.getByNamaLokasiCalls = make(map[string]int)
	}
	r.getByNamaLokasiCalls[namaLokasi]++
	return append([]domain.FormulaParams(nil), r.formulas[namaLokasi]...), nil
}

func (r *fakeFormulaRepo) CreateBatch(ctx context.Context, namaLokasi string, formulas []domain.FormulaParams) error {
	return nil
}

func (r *fakeFormulaRepo) DeleteByID(ctx context.Context, id int64) error {
	return nil
}

func (r *fakeFormulaRepo) GetByID(ctx context.Context, id int64) (*domain.FormulaParams, error) {
	return nil, nil
}

func (r *fakeFormulaRepo) GetAllGroupedByStation(ctx context.Context) ([]domain.StationFormulas, error) {
	return nil, nil
}

func TestDebitCalculatorUsesMatchingRatingCurveFormula(t *testing.T) {
	repo := &fakeFormulaRepo{formulas: map[string][]domain.FormulaParams{
		"pda-test": {
			{
				NamaLokasi:      "pda-test",
				C:               10,
				H0:              0.1,
				B:               2,
				TMAMin:          0,
				TMAMinInclusive: true,
				TMAMax:          1,
				TMAMaxInclusive: false,
				Priority:        10,
			},
			{
				NamaLokasi:      "pda-test",
				C:               20,
				H0:              0.5,
				B:               2,
				TMAMin:          1,
				TMAMinInclusive: true,
				TMAMax:          3,
				TMAMaxInclusive: true,
				Priority:        5,
			},
		},
	}}

	calc := NewDebitCalculator(repo)
	result, err := calc.Calculate(context.Background(), domain.PDARecord{
		NamaLokasi: "pda-test",
		WLevel:     150,
		TMA:        1.5,
		RecordedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if !result.IsValid {
		t.Fatal("expected debit calculation to be valid")
	}

	want := 20 * math.Pow(1.5-0.5, 2)
	if math.Abs(result.Debit-want) > 0.000001 {
		t.Fatalf("debit = %v, want %v", result.Debit, want)
	}
}

func TestDebitCalculatorRejectsTMAOutsideFormulaRange(t *testing.T) {
	repo := &fakeFormulaRepo{formulas: map[string][]domain.FormulaParams{
		"pda-test": {
			{
				NamaLokasi:      "pda-test",
				C:               10,
				H0:              0,
				B:               2,
				TMAMin:          0,
				TMAMinInclusive: true,
				TMAMax:          1,
				TMAMaxInclusive: true,
				Priority:        1,
			},
		},
	}}

	calc := NewDebitCalculator(repo)
	result, err := calc.Calculate(context.Background(), domain.PDARecord{
		NamaLokasi: "pda-test",
		TMA:        1.5,
	})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}
	if result.IsValid {
		t.Fatal("expected debit calculation to be invalid outside formula range")
	}
}

func TestDebitCalculatorCalculateBatchPreloadsMissingStationsOnce(t *testing.T) {
	repo := &fakeFormulaRepo{formulas: map[string][]domain.FormulaParams{
		"pda-a": {
			{
				NamaLokasi:      "pda-a",
				C:               10,
				H0:              0,
				B:               1,
				TMAMin:          0,
				TMAMinInclusive: true,
				TMAMax:          10,
				TMAMaxInclusive: true,
				Priority:        1,
			},
		},
		"pda-b": {
			{
				NamaLokasi:      "pda-b",
				C:               5,
				H0:              0,
				B:               2,
				TMAMin:          0,
				TMAMinInclusive: true,
				TMAMax:          10,
				TMAMaxInclusive: true,
				Priority:        1,
			},
		},
	}}

	calc := NewDebitCalculator(repo)
	results, err := calc.CalculateBatch(context.Background(), []domain.PDARecord{
		{NamaLokasi: "pda-a", TMA: 2},
		{NamaLokasi: "pda-a", TMA: 3},
		{NamaLokasi: "pda-b", TMA: 4},
		{NamaLokasi: "pda-b", TMA: 5},
	})
	if err != nil {
		t.Fatalf("CalculateBatch returned error: %v", err)
	}

	if repo.getAllCalls != 1 {
		t.Fatalf("GetAll calls = %d, want 1", repo.getAllCalls)
	}
	if got := repo.getByNamaLokasiCalls["pda-a"]; got != 0 {
		t.Fatalf("GetAllByNamaLokasi calls for pda-a = %d, want 0", got)
	}
	if got := repo.getByNamaLokasiCalls["pda-b"]; got != 0 {
		t.Fatalf("GetAllByNamaLokasi calls for pda-b = %d, want 0", got)
	}
	if len(results) != 4 {
		t.Fatalf("results len = %d, want 4", len(results))
	}
	for i, result := range results {
		if !result.IsValid {
			t.Fatalf("result %d expected valid", i)
		}
	}
}
