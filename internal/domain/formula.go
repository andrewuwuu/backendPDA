package domain

import "time"

type FormulaParams struct {
    ID          int64     `json:"id" db:"id"`
    NamaLokasi  string    `json:"nama_lokasi" db:"nama_lokasi"`
    StationName string    `json:"station_name" db:"station_name"`
    C           float64   `json:"c" db:"c"`
    H0          float64   `json:"h0" db:"h0"`
    B           float64   `json:"b" db:"b"`
    TMAMin      float64   `json:"tma_min" db:"tma_min"`
    TMAMax      float64   `json:"tma_max" db:"tma_max"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (f *FormulaParams) Validate(tma float64) bool {
    return tma >= f.TMAMin && tma <= f.TMAMax
}