package domain

import "time"

// AlertLevel represents the alert status of a station
type AlertLevel string

const (
	AlertLevelNormal  AlertLevel = "normal"
	AlertLevelSiaga   AlertLevel = "siaga"
	AlertLevelWaspada AlertLevel = "waspada"
	AlertLevelAwas    AlertLevel = "awas"
)

// IsValid checks if the alert level is a valid value
func (a AlertLevel) IsValid() bool {
	switch a {
	case AlertLevelNormal, AlertLevelSiaga, AlertLevelWaspada, AlertLevelAwas:
		return true
	}
	return false
}

// StationAlertLevel represents the alert level configuration for a station
type StationAlertLevel struct {
	ID         int64      `json:"id" db:"id"`
	NamaLokasi string     `json:"nama_lokasi" db:"nama_lokasi"`
	AlertLevel AlertLevel `json:"alert_level" db:"alert_level"`
	UpdatedBy  string     `json:"updated_by" db:"updated_by"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// AlertLevelUpdateRequest is the request body for updating a station's alert level
type AlertLevelUpdateRequest struct {
	AlertLevel AlertLevel `json:"alert_level"`
}

// BulkAlertLevelUpdateRequest is the request body for bulk updating alert levels
type BulkAlertLevelUpdateRequest struct {
	Updates []AlertLevelUpdateItem `json:"updates"`
}

// AlertLevelUpdateItem represents a single update in a bulk request
type AlertLevelUpdateItem struct {
	NamaLokasi string     `json:"nama_lokasi"`
	AlertLevel AlertLevel `json:"alert_level"`
}
