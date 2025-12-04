package domain

import "time"

type Station struct {
    ID           int64      `json:"id" db:"id"`
    NamaLokasi   string     `json:"nama_lokasi" db:"nama_lokasi"`
    NamaAlat     string     `json:"nama_alat" db:"nama_alat"`
    Lat          string     `json:"lat" db:"lat"`
    Lng          string     `json:"lng" db:"lng"`
    Sungai       string     `json:"sungai" db:"sungai"`
    Status       string     `json:"status" db:"status"`
    LastSyncedAt *time.Time `json:"last_synced_at" db:"last_synced_at"`
    CreatedAt    time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type PDARecord struct {
    NamaLokasi string    `json:"nama_lokasi"`
    NamaAlat   string    `json:"nama_alat"`
    WLevel     float64   `json:"w_level"`
    TMA        float64   `json:"tma"`
    Rain       float64   `json:"rain"`
    Lat        string    `json:"lat"`
    Lng        string    `json:"lng"`
    Sungai     string    `json:"sungai"`
    Status     string    `json:"status"`
    Siaga1     float64   `json:"siaga1"`
    Siaga2     float64   `json:"siaga2"`
    Siaga3     float64   `json:"siaga3"`
    Siaga4     float64   `json:"siaga4"`
    RecordedAt time.Time `json:"recorded_at"`
}

type DebitResult struct {
    NamaLokasi   string    `json:"nama_lokasi"`
    NamaAlat     string    `json:"nama_alat"`
    WLevel       float64   `json:"w_level"`
    TMA          float64   `json:"tma"`
    Debit        float64   `json:"debit"`
    IsValid      bool      `json:"is_valid"`
    Status       string    `json:"status"`
    Sungai       string    `json:"sungai"`
    Lat          string    `json:"lat"`
    Lng          string    `json:"lng"`
    RecordedAt   time.Time `json:"recorded_at"`
    CalculatedAt time.Time `json:"calculated_at"`
}