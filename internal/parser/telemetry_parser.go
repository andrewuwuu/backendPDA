package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/timeutil"
)

type RealtimeResponse struct {
	TelemetryJakarta []RealtimeTelemetryItem `json:"telemetryjakarta"`
}

type RealtimeTelemetryItem struct {
	NamaLokasi   string      `json:"nama_lokasi"`
	NamaAlat     string      `json:"nama_alat"`
	ReceivedDate string      `json:"ReceivedDate"`
	ReceivedTime string      `json:"ReceivedTime"`
	Rain         string      `json:"Rain"`
	WLevel       interface{} `json:"WLevel"`
	Lat          string      `json:"Lat"`
	Lng          string      `json:"Lng"`
	IdTipe       string      `json:"id_tipe"`
	Siaga1       string      `json:"siaga1"`
	Siaga2       string      `json:"siaga2"`
	Siaga3       string      `json:"siaga3"`
	Siaga4       string      `json:"siaga4"`
	Status       string      `json:"status"`
	Sungai       string      `json:"sungai"`
}

type HistoricalResponse struct {
	DataTelemetry []HistoricalTelemetryItem `json:"data_telemetryjakarta"`
}

type HistoricalTelemetryItem struct {
	ReceivedDate string `json:"ReceivedDate"`
	ReceivedTime string `json:"ReceivedTime"`
	Rain         string `json:"Rain"`
	WLevel       string `json:"WLevel"`
}

type TelemetryParser struct{}

func NewTelemetryParser() *TelemetryParser {
	return &TelemetryParser{}
}

func (p *TelemetryParser) ParseRealtime(r io.Reader) ([]domain.PDARecord, error) {
	var response RealtimeResponse
	if err := json.NewDecoder(r).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode realtime data: %w", err)
	}

	return p.processRealtimeItems(response.TelemetryJakarta)
}

func (p *TelemetryParser) processRealtimeItems(items []RealtimeTelemetryItem) ([]domain.PDARecord, error) {
	var records []domain.PDARecord
	for _, item := range items {
		// Filter: only id_tipe = "PDA"
		if item.IdTipe != "PDA" {
			continue
		}

		record, err := p.convertRealtimeItem(item)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *TelemetryParser) ParseHistorical(r io.Reader, namaLokasi string) ([]domain.PDARecord, error) {
	var response HistoricalResponse
	if err := json.NewDecoder(r).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode historical data: %w", err)
	}

	var records []domain.PDARecord
	for _, item := range response.DataTelemetry {
		record, err := p.convertHistoricalItem(item, namaLokasi)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *TelemetryParser) convertRealtimeItem(item RealtimeTelemetryItem) (domain.PDARecord, error) {
	wLevelCm, err := p.parseWLevel(item.WLevel)
	if err != nil {
		return domain.PDARecord{}, err
	}

	recordedAt, err := p.parseDateTime(item.ReceivedDate, item.ReceivedTime)
	if err != nil {
		return domain.PDARecord{}, err
	}

	rain, _ := strconv.ParseFloat(item.Rain, 64)
	siaga1, _ := strconv.ParseFloat(item.Siaga1, 64)
	siaga2, _ := strconv.ParseFloat(item.Siaga2, 64)
	siaga3, _ := strconv.ParseFloat(item.Siaga3, 64)
	siaga4, _ := strconv.ParseFloat(item.Siaga4, 64)

	return domain.PDARecord{
		NamaLokasi: item.NamaLokasi,
		NamaAlat:   item.NamaAlat,
		WLevel:     wLevelCm,
		TMA:        wLevelCm / 100.0, // cm to m
		Rain:       rain,
		Lat:        item.Lat,
		Lng:        item.Lng,
		Sungai:     item.Sungai,
		Status:     item.Status,
		Siaga1:     siaga1,
		Siaga2:     siaga2,
		Siaga3:     siaga3,
		Siaga4:     siaga4,
		RecordedAt: recordedAt,
	}, nil
}

func (p *TelemetryParser) convertHistoricalItem(item HistoricalTelemetryItem, namaLokasi string) (domain.PDARecord, error) {
	wLevelCm, err := strconv.ParseFloat(item.WLevel, 64)
	if err != nil {
		return domain.PDARecord{}, err
	}

	recordedAt, err := p.parseDateTime(item.ReceivedDate, item.ReceivedTime)
	if err != nil {
		return domain.PDARecord{}, err
	}

	rain, _ := strconv.ParseFloat(item.Rain, 64)

	return domain.PDARecord{
		NamaLokasi: namaLokasi,
		WLevel:     wLevelCm,
		TMA:        wLevelCm / 100.0,
		Rain:       rain,
		RecordedAt: recordedAt,
	}, nil
}

func (p *TelemetryParser) parseWLevel(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case nil:
		return 0, fmt.Errorf("WLevel is nil")
	default:
		return 0, fmt.Errorf("unexpected WLevel type: %T", value)
	}
}

func (p *TelemetryParser) parseDateTime(dateStr, timeStr string) (time.Time, error) {
	if dateStr == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("empty date or time")
	}

	dateTimeStr := fmt.Sprintf("%s %s", dateStr, timeStr)
	return time.ParseInLocation("2006-01-02 15:04:05", dateTimeStr, timeutil.JakartaLocation())
}
