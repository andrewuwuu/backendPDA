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
              COALESCE(priority, 0) as priority,
              updated_at 
              FROM formula_params 
              WHERE nama_lokasi = ?
              ORDER BY priority DESC
              LIMIT 1`

    err := r.db.GetContext(ctx, &params, query, namaLokasi)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("formula not found for: %s", namaLokasi)
    }
    if err != nil {
        return nil, err
    }

    return &params, nil
}

func (r *FormulaRepo) GetAllByNamaLokasi(ctx context.Context, namaLokasi string) ([]domain.FormulaParams, error) {
    var params []domain.FormulaParams
    query := `SELECT id, nama_lokasi, station_name, c, h0, b, tma_min,
              COALESCE(tma_min_inclusive, TRUE) as tma_min_inclusive,
              tma_max,
              COALESCE(tma_max_inclusive, TRUE) as tma_max_inclusive,
              COALESCE(priority, 0) as priority,
              updated_at
              FROM formula_params 
              WHERE nama_lokasi = ?
              ORDER BY priority DESC, tma_min ASC`
    
    err := r.db.SelectContext(ctx, &params, query, namaLokasi)
    if err != nil {
        return nil, err
    }
    
    return params, nil
}

func (r *FormulaRepo) GetAll(ctx context.Context) ([]domain.FormulaParams, error) {
    var params []domain.FormulaParams
    query := `SELECT id, nama_lokasi, station_name, c, h0, b, tma_min,
              COALESCE(tma_min_inclusive, TRUE) as tma_min_inclusive,
              tma_max,
              COALESCE(tma_max_inclusive, TRUE) as tma_max_inclusive,
              COALESCE(priority, 0) as priority,
              updated_at
              FROM formula_params 
              ORDER BY nama_lokasi, priority DESC, tma_min ASC`
    err := r.db.SelectContext(ctx, &params, query)
    return params, err
}

func (r *FormulaRepo) GetAllGroupedByStation(ctx context.Context) ([]domain.StationFormulas, error) {
    allFormulas, err := r.GetAll(ctx)
    if err != nil {
        return nil, err
    }

    groupedMap := make(map[string]*domain.StationFormulas)
    for _, f := range allFormulas {
        if _, exists := groupedMap[f.NamaLokasi]; !exists {
            groupedMap[f.NamaLokasi] = &domain.StationFormulas{
                NamaLokasi:  f.NamaLokasi,
                StationName: f.StationName,
                Formulas:    []domain.FormulaParams{},
            }
        }
        groupedMap[f.NamaLokasi].Formulas = append(groupedMap[f.NamaLokasi].Formulas, f)
    }

    result := make([]domain.StationFormulas, 0, len(groupedMap))
    for _, sf := range groupedMap {
        result = append(result, *sf)
    }

    return result, nil
}

func (r *FormulaRepo) GetByID(ctx context.Context, id int64) (*domain.FormulaParams, error) {
    var params domain.FormulaParams
    query := `SELECT id, nama_lokasi, station_name, c, h0, b, tma_min,
              COALESCE(tma_min_inclusive, TRUE) as tma_min_inclusive,
              tma_max,
              COALESCE(tma_max_inclusive, TRUE) as tma_max_inclusive,
              COALESCE(priority, 0) as priority,
              updated_at
              FROM formula_params WHERE id = ?`
    
    err := r.db.GetContext(ctx, &params, query, id)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("formula not found with id: %d", id)
    }
    if err != nil {
        return nil, err
    }

    return &params, nil
}

func (r *FormulaRepo) Create(ctx context.Context, params *domain.FormulaParams) error {
    query := `INSERT INTO formula_params 
              (nama_lokasi, station_name, c, h0, b, tma_min, tma_min_inclusive, tma_max, tma_max_inclusive, priority)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.ExecContext(ctx, query,
        params.NamaLokasi, params.StationName, params.C, params.H0, params.B,
        params.TMAMin, params.TMAMinInclusive, params.TMAMax, params.TMAMaxInclusive,
        params.Priority,
    )
    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    params.ID = id
    return nil
}

func (r *FormulaRepo) CreateBatch(ctx context.Context, namaLokasi string, formulas []domain.FormulaParams) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    deleteQuery := `DELETE FROM formula_params WHERE nama_lokasi = ?`
    if _, err := tx.ExecContext(ctx, deleteQuery, namaLokasi); err != nil {
        return fmt.Errorf("failed to delete existing formulas: %w", err)
    }

    insertQuery := `INSERT INTO formula_params 
                    (nama_lokasi, station_name, c, h0, b, tma_min, tma_min_inclusive, tma_max, tma_max_inclusive, priority)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

    for i := range formulas {
        formulas[i].NamaLokasi = namaLokasi
        result, err := tx.ExecContext(ctx, insertQuery,
            formulas[i].NamaLokasi, formulas[i].StationName, formulas[i].C, formulas[i].H0, formulas[i].B,
            formulas[i].TMAMin, formulas[i].TMAMinInclusive, formulas[i].TMAMax, formulas[i].TMAMaxInclusive,
            formulas[i].Priority,
        )
        if err != nil {
            return fmt.Errorf("failed to insert formula: %w", err)
        }
        id, _ := result.LastInsertId()
        formulas[i].ID = id
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}

func (r *FormulaRepo) Update(ctx context.Context, params *domain.FormulaParams) error {
    query := `UPDATE formula_params
              SET station_name = ?, c = ?, h0 = ?, b = ?, 
                  tma_min = ?, tma_min_inclusive = ?,
                  tma_max = ?, tma_max_inclusive = ?,
                  priority = ?
              WHERE id = ?`

    _, err := r.db.ExecContext(ctx, query,
        params.StationName, params.C, params.H0, params.B,
        params.TMAMin, params.TMAMinInclusive,
        params.TMAMax, params.TMAMaxInclusive,
        params.Priority,
        params.ID,
    )
    return err
}

func (r *FormulaRepo) Delete(ctx context.Context, namaLokasi string) error {
    query := `DELETE FROM formula_params WHERE nama_lokasi = ?`
    _, err := r.db.ExecContext(ctx, query, namaLokasi)
    return err
}

func (r *FormulaRepo) DeleteByID(ctx context.Context, id int64) error {
    query := `DELETE FROM formula_params WHERE id = ?`
    result, err := r.db.ExecContext(ctx, query, id)
    if err != nil {
        return err
    }
    
    affected, _ := result.RowsAffected()
    if affected == 0 {
        return fmt.Errorf("formula not found with id: %d", id)
    }
    
    return nil
}