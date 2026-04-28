package service

import (
	"context"
	"math"
	"testing"
	"time"

	"pda-monitor/internal/domain"
)

type fakeFormulaRepo struct {
	formulas map[string][]domain.FormulaParams
}

func (r *fakeFormulaRepo) GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.FormulaParams, error) {
	formulas := r.formulas[namaLokasi]
	if len(formulas) == 0 {
		return nil, nil
	}
	return &formulas[0], nil
}

func (r *fakeFormulaRepo) GetAll(ctx context.Context) ([]domain.FormulaParams, error) {
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
