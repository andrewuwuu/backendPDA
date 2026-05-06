package service

import (
	"context"
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
	stationNames := make(map[string]bool)
	for name := range tmaSummary {
		stationNames[name] = true
	}
	for name := range debitSnapshots {
		stationNames[name] = true
	}

	reports := make([]domain.DailyStationReport, 0, len(stationNames))
	for namaLokasi := range stationNames {
		namaAlat := stationMap[namaLokasi]
		if namaAlat == "" {
			namaAlat = namaLokasi
		}

		report := domain.DailyStationReport{
			NamaLokasi: namaLokasi,
			NamaAlat:   util.CleanStationName(namaLokasi, namaAlat),
		}

		if tma, ok := tmaSummary[namaLokasi]; ok {
			report.MinTMA = tma.MinTMA
			report.MaxTMA = tma.MaxTMA
		}

		if debits, ok := debitSnapshots[namaLokasi]; ok {
			report.Debit07 = debits[7]
			report.Debit12 = debits[12]
			report.Debit17 = debits[17]
		}

		reports = append(reports, report)
	}

	return reports
}
