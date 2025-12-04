package mysql

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/jmoiron/sqlx"

    "pda-monitor/internal/domain"
)

type FormulaRepo struct {
    db *sqlx.DB
}

func NewFormulaRepo(db *sqlx.DB) *FormulaRepo {
    return &FormulaRepo{db: db}
}

func (r *FormulaRepo) GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.FormulaParams, error) {
    var params domain.FormulaParams
    query := `SELECT * FROM formula_params WHERE nama_lokasi = ?`

    err := r.db.GetContext(ctx, &params, query, namaLokasi)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("formula not found for: %s", namaLokasi)
    }
    if err != nil {
        return nil, err
    }

    return &params, nil
}

func (r *FormulaRepo) GetAll(ctx context.Context) ([]domain.FormulaParams, error) {
    var params []domain.FormulaParams
    query := `SELECT * FROM formula_params ORDER BY nama_lokasi`
    err := r.db.SelectContext(ctx, &params, query)
    return params, err
}

func (r *FormulaRepo) Create(ctx context.Context, params *domain.FormulaParams) error {
    query := `INSERT INTO formula_params (nama_lokasi, station_name, c, h0, b, tma_min, tma_max)
              VALUES (?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.ExecContext(ctx, query,
        params.NamaLokasi, params.StationName, params.C, params.H0, params.B,
        params.TMAMin, params.TMAMax,
    )
    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    params.ID = id
    return nil
}

func (r *FormulaRepo) Update(ctx context.Context, params *domain.FormulaParams) error {
    query := `UPDATE formula_params
              SET station_name = ?, c = ?, h0 = ?, b = ?, tma_min = ?, tma_max = ?
              WHERE nama_lokasi = ?`

    _, err := r.db.ExecContext(ctx, query,
        params.StationName, params.C, params.H0, params.B,
        params.TMAMin, params.TMAMax,
        params.NamaLokasi,
    )
    return err
}

func (r *FormulaRepo) Delete(ctx context.Context, namaLokasi string) error {
    query := `DELETE FROM formula_params WHERE nama_lokasi = ?`
    _, err := r.db.ExecContext(ctx, query, namaLokasi)
    return err
}