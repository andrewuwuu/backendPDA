package mysql

import (
    "context"
    "database/sql"
    "time"

    "github.com/jmoiron/sqlx"

    "pda-monitor/internal/domain"
)

type StationRepo struct {
    db *sqlx.DB
}

func NewStationRepo(db *sqlx.DB) *StationRepo {
    return &StationRepo{db: db}
}

func (r *StationRepo) Upsert(ctx context.Context, station *domain.Station) error {
    query := `
        INSERT INTO stations (nama_lokasi, nama_alat, lat, lng, sungai, status, last_synced_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            nama_alat = VALUES(nama_alat),
            lat = VALUES(lat),
            lng = VALUES(lng),
            sungai = VALUES(sungai),
            status = VALUES(status),
            last_synced_at = VALUES(last_synced_at)`

    _, err := r.db.ExecContext(ctx, query,
        station.NamaLokasi,
        station.NamaAlat,
        station.Lat,
        station.Lng,
        station.Sungai,
        station.Status,
        time.Now(),
    )
    return err
}

func (r *StationRepo) UpsertBatch(ctx context.Context, stations []domain.Station) error {
    if len(stations) == 0 {
        return nil
    }

    query := `
        INSERT INTO stations (nama_lokasi, nama_alat, lat, lng, sungai, status, last_synced_at)
        VALUES (:nama_lokasi, :nama_alat, :lat, :lng, :sungai, :status, :last_synced_at)
        ON DUPLICATE KEY UPDATE
            nama_alat = VALUES(nama_alat),
            lat = VALUES(lat),
            lng = VALUES(lng),
            sungai = VALUES(sungai),
            status = VALUES(status),
            last_synced_at = VALUES(last_synced_at)`

    now := time.Now()
    for i := range stations {
        stations[i].LastSyncedAt = &now
    }

    _, err := r.db.NamedExecContext(ctx, query, stations)
    return err
}

func (r *StationRepo) GetAll(ctx context.Context) ([]domain.Station, error) {
    var stations []domain.Station
    query := `SELECT * FROM stations ORDER BY sungai, nama_lokasi`
    err := r.db.SelectContext(ctx, &stations, query)
    return stations, err
}

func (r *StationRepo) GetByNamaLokasi(ctx context.Context, namaLokasi string) (*domain.Station, error) {
    var station domain.Station
    query := `SELECT * FROM stations WHERE nama_lokasi = ?`
    err := r.db.GetContext(ctx, &station, query, namaLokasi)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &station, err
}

func (r *StationRepo) UpdateStatus(ctx context.Context, namaLokasi, status string) error {
    query := `UPDATE stations SET status = ?, last_synced_at = ? WHERE nama_lokasi = ?`
    _, err := r.db.ExecContext(ctx, query, status, time.Now(), namaLokasi)
    return err
}