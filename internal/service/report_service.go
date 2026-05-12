package service

import (
	"context"
	"strings"
	"time"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/repository"
	"pda-monitor/internal/util"
)

// ReportService assembles daily station report data from stations,
// TMA summaries, and debit snapshots. It provides a single aggregation
// path for both API export handlers and scheduler reports.
type ReportService struct {
	stationRepo    repository.StationRepository
	readingService *ReadingService
}

// NewReportService creates a new ReportService.
func NewReportService(
	stationRepo repository.StationRepository,
	readingService *ReadingService,
) *ReportService {
	return &ReportService{
		stationRepo:    stationRepo,
		readingService: readingService,
	}
}

// BuildDailyReport assembles a daily station report for the given date.
// It fetches station metadata, TMA range summaries, and debit snapshots
// at hours 07:00, 12:00, and 17:00, then merges them into a single
// slice of DailyStationReport structs.
func (s *ReportService) BuildDailyReport(ctx context.Context, date time.Time) ([]domain.DailyStationReport, error) {
	stations, err := s.stationRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	stationMap := make(map[string]string, len(stations))
	for _, st := range stations {
		stationMap[st.NamaLokasi] = st.NamaAlat
	}

	tmaSummary, err := s.readingService.GetDailyTMASummary(ctx, date)
	if err != nil {
		return nil, err
	}

	debitSnapshots, err := s.readingService.GetDebitSnapshots(ctx, date, []int{7, 12, 17})
	if err != nil {
		return nil, err
	}

	return assembleDailyReports(stationMap, tmaSummary, debitSnapshots), nil
}

// assembleDailyReports merges station metadata, TMA summaries, and debit
// snapshots into a slice of DailyStationReport. Station names are cleaned
// using util.CleanStationName for consistency across API and scheduler.
func assembleDailyReports(
	stationMap map[string]string,
	tmaSummary map[string]domain.TMARangeSummary,
	debitSnapshots map[string]map[int]*float64,
) []domain.DailyStationReport {
	// Normalize map keys to handle case-sensitivity differences between telemetry API and DB
	normTMA := make(map[string]domain.TMARangeSummary)
	for k, v := range tmaSummary {
		normTMA[strings.ToLower(k)] = v
	}
	normDebit := make(map[string]map[int]*float64)
	for k, v := range debitSnapshots {
		normDebit[strings.ToLower(k)] = v
	}

	stationNames := make(map[string]bool)
	for name := range stationMap {
		stationNames[strings.ToLower(name)] = true
	}
	for name := range normTMA {
		stationNames[name] = true
	}
	for name := range normDebit {
		stationNames[name] = true
	}

	reports := make([]domain.DailyStationReport, 0, len(stationNames))
	for normLokasi := range stationNames {
		// Try to find the original casing from stationMap, or fallback to the normalized one
		var originalLokasi, namaAlat string
		for k, v := range stationMap {
			if strings.ToLower(k) == normLokasi {
				originalLokasi = k
				namaAlat = v
				break
			}
		}
		if originalLokasi == "" {
			originalLokasi = normLokasi
		}
		if namaAlat == "" {
			namaAlat = originalLokasi
		}

		report := domain.DailyStationReport{
			NamaLokasi: originalLokasi,
			NamaAlat:   util.CleanStationName(originalLokasi, namaAlat),
		}

		if tma, ok := normTMA[normLokasi]; ok {
			report.MinTMA = tma.MinTMA
			report.MaxTMA = tma.MaxTMA
		}

		if debits, ok := normDebit[normLokasi]; ok {
			report.Debit07 = debits[7]
			report.Debit12 = debits[12]
			report.Debit17 = debits[17]
		}

		reports = append(reports, report)
	}

	return reports
}

// BuildScheduledReport assembles the latest debit results for all stations
func (s *ReportService) BuildScheduledReport(ctx context.Context) ([]domain.DebitResult, error) {
	stations, err := s.stationRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	readings, err := s.readingService.GetLatestReadings(ctx)
	if err != nil {
		return nil, err
	}

	readingMap := make(map[string]domain.HourlyReading)
	for _, r := range readings {
		readingMap[strings.ToLower(r.NamaLokasi)] = r
	}

	results := make([]domain.DebitResult, 0, len(stations))
	for _, st := range stations {
		debit := float64(0)
		isValid := false
		wLevel := float64(0)
		tma := float64(0)
		var recordedAt time.Time

		if r, ok := readingMap[strings.ToLower(st.NamaLokasi)]; ok {
			if r.Debit != nil {
				debit = *r.Debit
			}
			isValid = r.IsValid
			wLevel = r.WLevel
			tma = r.TMA
			recordedAt = r.RecordedAt
		}

		results = append(results, domain.DebitResult{
			NamaLokasi:   st.NamaLokasi,
			NamaAlat:     st.NamaAlat,
			WLevel:       wLevel,
			TMA:          tma,
			Debit:        debit,
			IsValid:      isValid,
			Status:       st.Status,
			Sungai:       st.Sungai,
			Lat:          st.Lat,
			Lng:          st.Lng,
			RecordedAt:   recordedAt,
			CalculatedAt: time.Now(),
		})
	}

	return results, nil
}
