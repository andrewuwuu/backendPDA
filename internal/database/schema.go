package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

var schemaStatements = []struct {
	name  string
	query string
}{
	{
		name: "stations",
		query: `
CREATE TABLE IF NOT EXISTS stations (
	id BIGINT NOT NULL AUTO_INCREMENT,
	nama_lokasi VARCHAR(191) NOT NULL,
	nama_alat VARCHAR(255) NOT NULL,
	lat VARCHAR(64) NOT NULL,
	lng VARCHAR(64) NOT NULL,
	sungai VARCHAR(191) NOT NULL,
	status VARCHAR(64) NOT NULL,
	last_synced_at DATETIME NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE KEY uk_stations_nama_lokasi (nama_lokasi),
	KEY idx_stations_sungai (sungai)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		name: "formula_params",
		query: `
CREATE TABLE IF NOT EXISTS formula_params (
	id BIGINT NOT NULL AUTO_INCREMENT,
	nama_lokasi VARCHAR(191) NOT NULL,
	station_name VARCHAR(255) NOT NULL,
	c DOUBLE NOT NULL,
	h0 DOUBLE NOT NULL,
	b DOUBLE NOT NULL,
	tma_min DOUBLE NOT NULL,
	tma_min_inclusive BOOLEAN NOT NULL DEFAULT TRUE,
	tma_max DOUBLE NOT NULL,
	tma_max_inclusive BOOLEAN NOT NULL DEFAULT TRUE,
	priority INT NOT NULL DEFAULT 0,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_formula_params_nama_lokasi (nama_lokasi),
	KEY idx_formula_params_station_name (station_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		name: "hourly_readings",
		query: `
CREATE TABLE IF NOT EXISTS hourly_readings (
	id BIGINT NOT NULL AUTO_INCREMENT,
	nama_lokasi VARCHAR(191) NOT NULL,
	hour_bucket DATETIME NOT NULL,
	recorded_at DATETIME NOT NULL,
	w_level DOUBLE NOT NULL,
	tma DOUBLE NOT NULL,
	debit DOUBLE NULL,
	is_valid BOOLEAN NOT NULL DEFAULT FALSE,
	rain DOUBLE NOT NULL DEFAULT 0,
	PRIMARY KEY (id),
	UNIQUE KEY uk_hourly_readings_station_bucket_recorded_at (nama_lokasi, hour_bucket, recorded_at),
	KEY idx_hourly_readings_hour_bucket (hour_bucket),
	KEY idx_hourly_readings_recorded_at (recorded_at),
	KEY idx_hourly_readings_station_recorded_at (nama_lokasi, recorded_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		name: "users",
		query: `
CREATE TABLE IF NOT EXISTS users (
	id BIGINT NOT NULL AUTO_INCREMENT,
	username VARCHAR(191) NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	role VARCHAR(64) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE KEY uk_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		name: "station_alert_levels",
		query: `
CREATE TABLE IF NOT EXISTS station_alert_levels (
	id BIGINT NOT NULL AUTO_INCREMENT,
	nama_lokasi VARCHAR(191) NOT NULL,
	alert_level ENUM('normal', 'siaga', 'waspada', 'awas') NOT NULL DEFAULT 'normal',
	upper_limit_normal DOUBLE NULL,
	upper_limit_siaga DOUBLE NULL,
	upper_limit_waspada DOUBLE NULL,
	upper_limit_awas DOUBLE NULL,
	updated_by VARCHAR(191) NOT NULL DEFAULT 'system',
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE KEY uk_station_alert_levels_nama_lokasi (nama_lokasi),
	KEY idx_station_alert_levels_alert_level (alert_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
}

func InitSchema(ctx context.Context, db *sqlx.DB) error {
	for _, stmt := range schemaStatements {
		if _, err := db.ExecContext(ctx, stmt.query); err != nil {
			return fmt.Errorf("create %s table: %w", stmt.name, err)
		}
	}

	return nil
}

func TableNames() []string {
	names := make([]string, 0, len(schemaStatements))
	for _, stmt := range schemaStatements {
		names = append(names, stmt.name)
	}
	return names
}
