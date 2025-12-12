package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"pda-monitor/internal/domain"
)

type AlertLevelRepo struct {
	db *sqlx.DB
}

func NewAlertLevelRepo(db *sqlx.DB) *AlertLevelRepo {
	return &AlertLevelRepo{db: db}
}

func (r *AlertLevelRepo) GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.StationAlertLevel, error) {
	var alert domain.StationAlertLevel
	query := `SELECT id, nama_lokasi, alert_level, 
              upper_limit_normal, upper_limit_siaga, upper_limit_waspada, upper_limit_awas,
              updated_by, updated_at, created_at 
              FROM station_alert_levels WHERE nama_lokasi = ?`

	err := r.db.GetContext(ctx, &alert, query, namaLokasi)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *AlertLevelRepo) GetAll(ctx context.Context) ([]domain.StationAlertLevel, error) {
	var alerts []domain.StationAlertLevel
	query := `SELECT id, nama_lokasi, alert_level,
              upper_limit_normal, upper_limit_siaga, upper_limit_waspada, upper_limit_awas,
              updated_by, updated_at, created_at 
              FROM station_alert_levels ORDER BY nama_lokasi`
	err := r.db.SelectContext(ctx, &alerts, query)
	return alerts, err
}

func (r *AlertLevelRepo) GetByAlertLevel(ctx context.Context, level domain.AlertLevel) ([]domain.StationAlertLevel, error) {
	var alerts []domain.StationAlertLevel
	query := `SELECT id, nama_lokasi, alert_level,
              upper_limit_normal, upper_limit_siaga, upper_limit_waspada, upper_limit_awas,
              updated_by, updated_at, created_at 
              FROM station_alert_levels WHERE alert_level = ? ORDER BY nama_lokasi`
	err := r.db.SelectContext(ctx, &alerts, query, level)
	return alerts, err
}

func (r *AlertLevelRepo) Upsert(ctx context.Context, alert *domain.StationAlertLevel) error {
	query := `
        INSERT INTO station_alert_levels (nama_lokasi, alert_level, 
            upper_limit_normal, upper_limit_siaga, upper_limit_waspada, upper_limit_awas, updated_by)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            alert_level = VALUES(alert_level),
            upper_limit_normal = VALUES(upper_limit_normal),
            upper_limit_siaga = VALUES(upper_limit_siaga),
            upper_limit_waspada = VALUES(upper_limit_waspada),
            upper_limit_awas = VALUES(upper_limit_awas),
            updated_by = VALUES(updated_by)`

	result, err := r.db.ExecContext(ctx, query, alert.NamaLokasi, alert.AlertLevel,
		alert.UpperLimitNormal, alert.UpperLimitSiaga, alert.UpperLimitWaspada, alert.UpperLimitAwas,
		alert.UpdatedBy)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	if id > 0 {
		alert.ID = id
	}
	return nil
}

func (r *AlertLevelRepo) UpsertBatch(ctx context.Context, alerts []domain.StationAlertLevel) error {
	if len(alerts) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
        INSERT INTO station_alert_levels (nama_lokasi, alert_level,
            upper_limit_normal, upper_limit_siaga, upper_limit_waspada, upper_limit_awas, updated_by)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            alert_level = VALUES(alert_level),
            upper_limit_normal = VALUES(upper_limit_normal),
            upper_limit_siaga = VALUES(upper_limit_siaga),
            upper_limit_waspada = VALUES(upper_limit_waspada),
            upper_limit_awas = VALUES(upper_limit_awas),
            updated_by = VALUES(updated_by)`

	for _, alert := range alerts {
		if _, err := tx.ExecContext(ctx, query, alert.NamaLokasi, alert.AlertLevel,
			alert.UpperLimitNormal, alert.UpperLimitSiaga, alert.UpperLimitWaspada, alert.UpperLimitAwas,
			alert.UpdatedBy); err != nil {
			return fmt.Errorf("failed to upsert alert level for %s: %w", alert.NamaLokasi, err)
		}
	}

	return tx.Commit()
}

func (r *AlertLevelRepo) Delete(ctx context.Context, namaLokasi string) error {
	query := `DELETE FROM station_alert_levels WHERE nama_lokasi = ?`
	result, err := r.db.ExecContext(ctx, query, namaLokasi)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("alert level not found for station: %s", namaLokasi)
	}
	return nil
}
