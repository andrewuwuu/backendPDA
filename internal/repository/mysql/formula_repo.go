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
    query := `SELECT id, nama_lokasi, station_name, c, h0, b, tma_min, 
              COALESCE(tma_min_inclusive, TRUE) as tma_min_inclusive,
              tma_max, 
              COALESCE(tma_max_inclusive, TRUE) as tma_max_inclusive,
              updated_at 
              FROM formula_params WHERE nama_lokasi = ?`

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
    query := `SELECT id, nama_lokasi, station_name, c, h0, b, tma_min,
              COALESCE(tma_min_inclusive, TRUE) as tma_min_inclusive,
              tma_max,
              COALESCE(tma_max_inclusive, TRUE) as tma_max_inclusive,
              updated_at
              FROM formula_params ORDER BY nama_lokasi`
    err := r.db.SelectContext(ctx, &params, query)
    return params, err
}

func (r *FormulaRepo) Create(ctx context.Context, params *domain.FormulaParams) error {
    query := `INSERT INTO formula_params 
              (nama_lokasi, station_name, c, h0, b, tma_min, tma_min_inclusive, tma_max, tma_max_inclusive)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.ExecContext(ctx, query,
        params.NamaLokasi, params.StationName, params.C, params.H0, params.B,
        params.TMAMin, params.TMAMinInclusive, params.TMAMax, params.TMAMaxInclusive,
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
              SET station_name = ?, c = ?, h0 = ?, b = ?, 
                  tma_min = ?, tma_min_inclusive = ?,
                  tma_max = ?, tma_max_inclusive = ?
              WHERE nama_lokasi = ?`

    _, err := r.db.ExecContext(ctx, query,
        params.StationName, params.C, params.H0, params.B,
        params.TMAMin, params.TMAMinInclusive,
        params.TMAMax, params.TMAMaxInclusive,
        params.NamaLokasi,
    )
    return err
}

func (r *FormulaRepo) Delete(ctx context.Context, namaLokasi string) error {
    query := `DELETE FROM formula_params WHERE nama_lokasi = ?`
    _, err := r.db.ExecContext(ctx, query, namaLokasi)
    return err
}