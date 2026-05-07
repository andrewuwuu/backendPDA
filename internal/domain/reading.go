package domain

import "time"

type HourlyReading struct {
	ID         int64     `json:"id" db:"id"`
	NamaLokasi string    `json:"nama_lokasi" db:"nama_lokasi"`
	HourBucket time.Time `json:"hour_bucket" db:"hour_bucket"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
	WLevel     float64   `json:"w_level" db:"w_level"`
	TMA        float64   `json:"tma" db:"tma"`
	Debit      *float64  `json:"debit" db:"debit"`
	IsValid    bool      `json:"is_valid" db:"is_valid"`
	Rain       float64   `json:"rain" db:"rain"`
}

type HourlySummary struct {
	NamaLokasi   string    `json:"nama_lokasi" db:"nama_lokasi"`
	HourBucket   time.Time `json:"hour_bucket" db:"hour_bucket"`
	ReadingCount int       `json:"reading_count" db:"reading_count"`
	AvgWLevel    float64   `json:"avg_w_level" db:"avg_w_level"`
	AvgTMA       float64   `json:"avg_tma" db:"avg_tma"`
	AvgDebit     *float64  `json:"avg_debit" db:"avg_debit"`
	MinDebit     *float64  `json:"min_debit" db:"min_debit"`
	MaxDebit     *float64  `json:"max_debit" db:"max_debit"`
	TotalRain    float64   `json:"total_rain" db:"total_rain"`
}

type TMARangeSummary struct {
	NamaLokasi string   `json:"nama_lokasi" db:"nama_lokasi"`
	MinTMA     *float64 `json:"min_tma" db:"min_tma"`
	MaxTMA     *float64 `json:"max_tma" db:"max_tma"`
}

type HourlyDebitSnapshot struct {
	NamaLokasi string   `json:"nama_lokasi" db:"nama_lokasi"`
	Hour       int      `json:"hour" db:"hour"`
	Debit      *float64 `json:"debit" db:"debit"`
	TMA        float64  `json:"tma" db:"tma"`
}

type HourlyDebitSnapshotRow struct {
	NamaLokasi string   `db:"nama_lokasi"`
	Hour       int      `db:"hour"`
	Debit      *float64 `db:"debit"`
	TMA        float64  `db:"tma"`
}

type DailyStationReport struct {
	NamaLokasi string
	NamaAlat   string
	Debit07    *float64
	Debit12    *float64
	Debit17    *float64
	MinTMA     *float64
	MaxTMA     *float64
}

// ReadingWithAlertLevel combines a reading with its computed alert level
type ReadingWithAlertLevel struct {
	HourlyReading
	AlertLevel AlertLevel `json:"alert_level"`
}
