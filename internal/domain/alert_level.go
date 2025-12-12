package domain

import "time"

type AlertLevel string

const (
	AlertLevelNormal  AlertLevel = "normal"
	AlertLevelSiaga   AlertLevel = "siaga"
	AlertLevelWaspada AlertLevel = "waspada"
	AlertLevelAwas    AlertLevel = "awas"
)

func (a AlertLevel) IsValid() bool {
	switch a {
	case AlertLevelNormal, AlertLevelSiaga, AlertLevelWaspada, AlertLevelAwas:
		return true
	}
	return false
}

type StationAlertLevel struct {
	ID                int64      `json:"id" db:"id"`
	NamaLokasi        string     `json:"nama_lokasi" db:"nama_lokasi"`
	AlertLevel        AlertLevel `json:"alert_level" db:"alert_level"`
	UpperLimitNormal  *float64   `json:"upper_limit_normal" db:"upper_limit_normal"`
	UpperLimitSiaga   *float64   `json:"upper_limit_siaga" db:"upper_limit_siaga"`
	UpperLimitWaspada *float64   `json:"upper_limit_waspada" db:"upper_limit_waspada"`
	UpperLimitAwas    *float64   `json:"upper_limit_awas" db:"upper_limit_awas"`
	UpdatedBy         string     `json:"updated_by" db:"updated_by"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

type AlertLevelUpdateRequest struct {
	AlertLevel        AlertLevel `json:"alert_level"`
	UpperLimitNormal  *float64   `json:"upper_limit_normal,omitempty"`
	UpperLimitSiaga   *float64   `json:"upper_limit_siaga,omitempty"`
	UpperLimitWaspada *float64   `json:"upper_limit_waspada,omitempty"`
	UpperLimitAwas    *float64   `json:"upper_limit_awas,omitempty"`
}

type BulkAlertLevelUpdateRequest struct {
	Updates []AlertLevelUpdateItem `json:"updates"`
}

type AlertLevelUpdateItem struct {
	NamaLokasi        string     `json:"nama_lokasi"`
	AlertLevel        AlertLevel `json:"alert_level"`
	UpperLimitNormal  *float64   `json:"upper_limit_normal,omitempty"`
	UpperLimitSiaga   *float64   `json:"upper_limit_siaga,omitempty"`
	UpperLimitWaspada *float64   `json:"upper_limit_waspada,omitempty"`
	UpperLimitAwas    *float64   `json:"upper_limit_awas,omitempty"`
}
