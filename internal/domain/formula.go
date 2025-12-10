package domain

import "time"

type FormulaParams struct {
    ID              int64     `json:"id" db:"id"`
    NamaLokasi      string    `json:"nama_lokasi" db:"nama_lokasi"`
    StationName     string    `json:"station_name" db:"station_name"`
    C               float64   `json:"c" db:"c"`
    H0              float64   `json:"h0" db:"h0"`
    B               float64   `json:"b" db:"b"`
    TMAMin          float64   `json:"tma_min" db:"tma_min"`
    TMAMinInclusive bool      `json:"tma_min_inclusive" db:"tma_min_inclusive"`
    TMAMax          float64   `json:"tma_max" db:"tma_max"`
    TMAMaxInclusive bool      `json:"tma_max_inclusive" db:"tma_max_inclusive"`
    Priority        int       `json:"priority" db:"priority"`
    UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

func (f *FormulaParams) Validate(tma float64) bool {
    var minValid, maxValid bool

    if f.TMAMinInclusive {
        minValid = tma >= f.TMAMin
    } else {
        minValid = tma > f.TMAMin
    }

    if f.TMAMaxInclusive {
        maxValid = tma <= f.TMAMax
    } else {
        maxValid = tma < f.TMAMax
    }

    return minValid && maxValid
}

type FormulaCreateRequest struct {
    NamaLokasi  string          `json:"nama_lokasi"`
    StationName string          `json:"station_name"`
    Formulas    []FormulaRange  `json:"formulas"`
}

type FormulaRange struct {
    C               float64 `json:"c"`
    H0              float64 `json:"h0"`
    B               float64 `json:"b"`
    TMAMin          float64 `json:"tma_min"`
    TMAMinInclusive bool    `json:"tma_min_inclusive"`
    TMAMax          float64 `json:"tma_max"`
    TMAMaxInclusive bool    `json:"tma_max_inclusive"`
    Priority        int     `json:"priority"`
}

type StationFormulas struct {
    NamaLokasi  string          `json:"nama_lokasi"`
    StationName string          `json:"station_name"`
    Formulas    []FormulaParams `json:"formulas"`
}