package mysql

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/timeutil"
)

type ReadingRepo struct {
	db *sqlx.DB
}

const hourlyReadingColumns = `
id,
nama_lokasi,
hour_bucket,
recorded_at,
w_level,
tma,
debit,
is_valid,
rain`

func NewReadingRepo(db *sqlx.DB) *ReadingRepo {
	return &ReadingRepo{db: db}
}

func (r *ReadingRepo) InsertReading(ctx context.Context, reading *domain.HourlyReading) error {
	query := `
        INSERT INTO hourly_readings
            (nama_lokasi, hour_bucket, recorded_at, w_level, tma, debit, is_valid, rain)
        VALUES
            (?, ?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            w_level = VALUES(w_level),
            tma = VALUES(tma),
            debit = VALUES(debit),
            is_valid = VALUES(is_valid),
            rain = VALUES(rain)`

	_, err := r.db.ExecContext(ctx, query,
		reading.NamaLokasi,
		reading.HourBucket,
		reading.RecordedAt,
		reading.WLevel,
		reading.TMA,
		reading.Debit,
		reading.IsValid,
		reading.Rain,
	)
	return err
}

func (r *ReadingRepo) BulkInsertReadings(ctx context.Context, readings []domain.HourlyReading) error {
	if len(readings) == 0 {
		return nil
	}

	query := `
        INSERT INTO hourly_readings
            (nama_lokasi, hour_bucket, recorded_at, w_level, tma, debit, is_valid, rain)
        VALUES
            (:nama_lokasi, :hour_bucket, :recorded_at, :w_level, :tma, :debit, :is_valid, :rain)
        ON DUPLICATE KEY UPDATE
            w_level = VALUES(w_level),
            tma = VALUES(tma),
            debit = VALUES(debit),
            is_valid = VALUES(is_valid),
            rain = VALUES(rain)`

	_, err := r.db.NamedExecContext(ctx, query, readings)
	return err
}

func (r *ReadingRepo) GetCurrentHourReadings(ctx context.Context) ([]domain.HourlyReading, error) {
	var readings []domain.HourlyReading
	hourBucket := timeutil.TruncateToJakartaHour(time.Now())

	query := `SELECT ` + hourlyReadingColumns + ` FROM hourly_readings WHERE hour_bucket = ? ORDER BY nama_lokasi, recorded_at DESC`
	err := r.db.SelectContext(ctx, &readings, query, hourBucket)
	return readings, err
}

func (r *ReadingRepo) GetCurrentHourByStation(ctx context.Context, namaLokasi string) ([]domain.HourlyReading, error) {
	var readings []domain.HourlyReading
	hourBucket := timeutil.TruncateToJakartaHour(time.Now())

	query := `SELECT ` + hourlyReadingColumns + ` FROM hourly_readings WHERE nama_lokasi = ? AND hour_bucket = ? ORDER BY recorded_at DESC`
	err := r.db.SelectContext(ctx, &readings, query, namaLokasi, hourBucket)
	return readings, err
}

func (r *ReadingRepo) GetHourlySummary(ctx context.Context) ([]domain.HourlySummary, error) {
	var summaries []domain.HourlySummary
	hourBucket := timeutil.TruncateToJakartaHour(time.Now())

	query := `
        SELECT
            nama_lokasi,
            hour_bucket,
            COUNT(*) as reading_count,
            AVG(w_level) as avg_w_level,
            AVG(tma) as avg_tma,
            AVG(CASE WHEN is_valid THEN debit END) as avg_debit,
            MIN(CASE WHEN is_valid THEN debit END) as min_debit,
            MAX(CASE WHEN is_valid THEN debit END) as max_debit,
            SUM(rain) as total_rain
        FROM hourly_readings
        WHERE hour_bucket = ?
        GROUP BY nama_lokasi, hour_bucket
        ORDER BY nama_lokasi`

	err := r.db.SelectContext(ctx, &summaries, query, hourBucket)
	return summaries, err
}

func (r *ReadingRepo) GetLatestReadingPerStation(ctx context.Context) ([]domain.HourlyReading, error) {
	var readings []domain.HourlyReading
	// Look back 2 hours to find the latest available reading for each station
	cutoff := time.Now().Add(-2 * time.Hour)

	query := `
        SELECT
            hr.id,
            hr.nama_lokasi,
            hr.hour_bucket,
            hr.recorded_at,
            hr.w_level,
            hr.tma,
            hr.debit,
            hr.is_valid,
            hr.rain
        FROM hourly_readings hr
        INNER JOIN (
            SELECT nama_lokasi, MAX(recorded_at) as max_time
            FROM hourly_readings
            WHERE recorded_at >= ?
            GROUP BY nama_lokasi
        ) latest ON hr.nama_lokasi = latest.nama_lokasi
               AND hr.recorded_at = latest.max_time
        ORDER BY hr.nama_lokasi`

	err := r.db.SelectContext(ctx, &readings, query, cutoff)
	return readings, err
}

func (r *ReadingRepo) CleanupOldReadings(ctx context.Context, hoursToKeep int) (int64, error) {
	cutoff := timeutil.TruncateToJakartaHour(time.Now()).Add(-time.Duration(hoursToKeep) * time.Hour)

	query := `DELETE FROM hourly_readings WHERE hour_bucket < ?`
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *ReadingRepo) GetTMARangeForDay(ctx context.Context, date time.Time, startHour, endHour int) ([]domain.TMARangeSummary, error) {
	var summaries []domain.TMARangeSummary

	loc := timeutil.JakartaLocation()
	startTime := time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, loc)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), endHour, 59, 59, 0, loc)

	query := `
        SELECT 
            nama_lokasi,
            MIN(tma) as min_tma,
            MAX(tma) as max_tma
        FROM hourly_readings
        WHERE recorded_at >= ? AND recorded_at <= ?
        GROUP BY nama_lokasi
        ORDER BY nama_lokasi`

	err := r.db.SelectContext(ctx, &summaries, query, startTime, endTime)
	return summaries, err
}

func (r *ReadingRepo) GetDebitAtHours(ctx context.Context, date time.Time, hours []int) ([]domain.HourlyDebitSnapshot, error) {
	if len(hours) == 0 {
		return nil, nil
	}

	loc := timeutil.JakartaLocation()
	// Day start: 00:00:00 on the target date in Jakarta time
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)

	// Build a single UNION ALL query: one subquery per target hour.
	// Each subquery finds the latest valid debit reading per station
	// between dayStart and hour:05:00.
	var queryParts []string
	var args []interface{}

	for _, hour := range hours {
		cutoff := time.Date(date.Year(), date.Month(), date.Day(), hour, 5, 0, 0, loc)

		part := `
		SELECT sub.nama_lokasi, ? AS hour, sub.debit, sub.tma
		FROM hourly_readings sub
		INNER JOIN (
			SELECT nama_lokasi, MAX(recorded_at) AS max_recorded_at
			FROM hourly_readings
			WHERE recorded_at >= ?
			  AND recorded_at <= ?
			  AND is_valid = TRUE
			  AND debit IS NOT NULL
			GROUP BY nama_lokasi
		) latest ON sub.nama_lokasi = latest.nama_lokasi
		       AND sub.recorded_at = latest.max_recorded_at`

		queryParts = append(queryParts, part)
		args = append(args, hour, dayStart, cutoff)
	}

	fullQuery := strings.Join(queryParts, "\nUNION ALL\n") + "\nORDER BY nama_lokasi, hour"

	var rows []domain.HourlyDebitSnapshotRow
	if err := r.db.SelectContext(ctx, &rows, fullQuery, args...); err != nil {
		return nil, err
	}

	snapshots := make([]domain.HourlyDebitSnapshot, len(rows))
	for i, row := range rows {
		snapshots[i] = domain.HourlyDebitSnapshot{
			NamaLokasi: row.NamaLokasi,
			Hour:       row.Hour,
			Debit:      row.Debit,
			TMA:        row.TMA,
		}
	}

	return snapshots, nil
}


func (r *ReadingRepo) GetReadingsByTimeRange(ctx context.Context, namaLokasi string, from, to time.Time) ([]domain.HourlyReading, error) {
	var readings []domain.HourlyReading

	query := `
        SELECT ` + hourlyReadingColumns + ` FROM hourly_readings 
        WHERE nama_lokasi = ? 
        AND recorded_at >= ? 
        AND recorded_at <= ?
        ORDER BY recorded_at ASC`

	err := r.db.SelectContext(ctx, &readings, query, namaLokasi, from, to)
	return readings, err
}

func (r *ReadingRepo) GetAllReadingsByTimeRange(ctx context.Context, from, to time.Time) ([]domain.HourlyReading, error) {
	var readings []domain.HourlyReading

	query := `
        SELECT ` + hourlyReadingColumns + ` FROM hourly_readings 
        WHERE recorded_at >= ? 
        AND recorded_at <= ?
        ORDER BY nama_lokasi, recorded_at ASC`

	err := r.db.SelectContext(ctx, &readings, query, from, to)
	return readings, err
}
