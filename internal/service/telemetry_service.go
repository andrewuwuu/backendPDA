package service

import (
    "context"
    "fmt"
    "net/http"
    "net/url"
    "time"

    "pda-monitor/internal/domain"
    "pda-monitor/internal/parser"
    "pda-monitor/internal/repository"
)

type TelemetryConfig struct {
    BaseURL    string
    IDBBWS     string
    TimeoutSec int
}

type TelemetryService struct {
    client      *http.Client
    config      TelemetryConfig
    parser      *parser.TelemetryParser
    stationRepo repository.StationRepository
}

func NewTelemetryService(
    config TelemetryConfig,
    parser *parser.TelemetryParser,
    stationRepo repository.StationRepository,
) *TelemetryService {
    timeout := time.Duration(config.TimeoutSec) * time.Second
    if timeout == 0 {
        timeout = 30 * time.Second
    }

    return &TelemetryService{
        client:      &http.Client{Timeout: timeout},
        config:      config,
        parser:      parser,
        stationRepo: stationRepo,
    }
}

func (s *TelemetryService) FetchRealtime(ctx context.Context) ([]domain.PDARecord, error) {
    endpoint := fmt.Sprintf("%s/datatelemetry2.php?idbbws=%s",
        s.config.BaseURL,
        s.config.IDBBWS,
    )

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch realtime data: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return s.parser.ParseRealtime(resp.Body)
}

func (s *TelemetryService) FetchHistorical(ctx context.Context, namaLokasi string, from, to time.Time) ([]domain.PDARecord, error) {
    params := url.Values{}
    params.Set("nama_lokasi", namaLokasi)
    params.Set("dt", from.Format("2006-01-02"))
    params.Set("dtf", to.Format("2006-01-02"))

    endpoint := fmt.Sprintf("%s/loc_datatelemetry_forbintek.php?%s",
        s.config.BaseURL,
        params.Encode(),
    )

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch historical data: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return s.parser.ParseHistorical(resp.Body, namaLokasi)
}

func (s *TelemetryService) SyncStations(ctx context.Context, records []domain.PDARecord) (int, error) {

    stations := make([]domain.Station, 0, len(records))
    now := time.Now()

    for _, r := range records {
        stations = append(stations, domain.Station{
            NamaLokasi:   r.NamaLokasi,
            NamaAlat:     r.NamaAlat,
            Lat:          r.Lat,
            Lng:          r.Lng,
            Sungai:       r.Sungai,
            Status:       r.Status,
            LastSyncedAt: &now,
        })
    }

    if err := s.stationRepo.UpsertBatch(ctx, stations); err != nil {
        return 0, fmt.Errorf("failed to upsert stations: %w", err)
    }

    return len(stations), nil
}