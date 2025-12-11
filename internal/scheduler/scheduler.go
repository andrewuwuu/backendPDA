package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/logger"
	"pda-monitor/internal/notification"
	"pda-monitor/internal/report"
	"pda-monitor/internal/repository"
	"pda-monitor/internal/service"
	"pda-monitor/internal/util"
)

const component = "Scheduler"
const HoursToKeep = 168

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.FixedZone("WIB", 7*60*60)
	}
}

type Scheduler struct {
	cron             *cron.Cron
	telemetryService *service.TelemetryService
	readingService   *service.ReadingService
	calculator       *service.DebitCalculator
	telegram         *notification.TelegramNotifier
	excelService     *report.ExcelReportService
	stationRepo      repository.StationRepository
}

func NewScheduler(
	telemetryService *service.TelemetryService,
	readingService *service.ReadingService,
	calculator *service.DebitCalculator,
	telegram *notification.TelegramNotifier,
	excelService *report.ExcelReportService,
	stationRepo repository.StationRepository,
) *Scheduler {
	return &Scheduler{
		cron:             cron.New(cron.WithLocation(jakartaLoc)),
		telemetryService: telemetryService,
		readingService:   readingService,
		calculator:       calculator,
		telegram:         telegram,
		excelService:     excelService,
		stationRepo:      stationRepo,
	}
}

func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc("*/5 * * * *", s.syncReadings); err != nil {
		return fmt.Errorf("failed to schedule syncReadings: %w", err)
	}

	if _, err := s.cron.AddFunc("30 0 * * *", s.cleanupOldReadings); err != nil {
		return fmt.Errorf("failed to schedule cleanupOldReadings: %w", err)
	}

	if _, err := s.cron.AddFunc("0 * * * *", s.syncStationMetadata); err != nil {
		return fmt.Errorf("failed to schedule syncStationMetadata: %w", err)
	}

	for _, hour := range []int{7, 12, 17} {
		h := hour
		if _, err := s.cron.AddFunc(fmt.Sprintf("0 %d * * *", h), s.sendScheduledReport); err != nil {
			return fmt.Errorf("failed to schedule sendScheduledReport for hour %d: %w", h, err)
		}
	}

	if _, err := s.cron.AddFunc("10 17 * * *", s.sendDailyExcelReport); err != nil {
		return fmt.Errorf("failed to schedule sendDailyExcelReport: %w", err)
	}

	s.cron.Start()

	logger.Info(component, "Scheduler started (Asia/Jakarta timezone)", logger.Fields(
		"readings_sync", "every 5 minutes",
		"cleanup", "daily at 00:30 WIB (keeps 7 days)",
		"station_sync", "every hour at :00",
		"telegram_text", "07:00, 12:00, 17:00 WIB",
		"daily_excel", "17:10 WIB",
	))

	return nil
}

func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	logger.Info(component, "Scheduler stopped")
}

func (s *Scheduler) syncReadings() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	records, err := s.telemetryService.FetchRealtime(ctx)
	if err != nil {
		logger.Error(component, "Failed to fetch realtime data", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	count, err := s.readingService.ProcessAndStoreReadings(ctx, records)
	if err != nil {
		logger.Error(component, "Failed to store readings", logger.Fields(
			"error", err.Error(),
			"fetched", len(records),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	logger.Info(component, "Readings synced successfully", logger.Fields(
		"count", count,
		"duration_ms", time.Since(startTime).Milliseconds(),
	))
}

func (s *Scheduler) cleanupOldReadings() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	deleted, err := s.readingService.CleanupOldData(ctx, HoursToKeep)
	if err != nil {
		logger.Error(component, "Failed to cleanup old readings", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	if deleted > 0 {
		logger.Info(component, "Old readings cleaned up", logger.Fields(
			"deleted", deleted,
			"hours_kept", HoursToKeep,
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
	} else {
		logger.Debug(component, "No old readings to clean up")
	}
}

func (s *Scheduler) syncStationMetadata() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	count, err := s.telemetryService.SyncStations(ctx)
	if err != nil {
		logger.Error(component, "Failed to sync station metadata", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	logger.Info(component, "Station metadata synced", logger.Fields(
		"count", count,
		"duration_ms", time.Since(startTime).Milliseconds(),
	))
}

func (s *Scheduler) sendScheduledReport() {
	if s.telegram == nil {
		logger.Warn(component, "Telegram not configured, skipping scheduled report")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	startTime := time.Now()
	now := time.Now().In(jakartaLoc)

	records, err := s.telemetryService.FetchRealtime(ctx)
	if err != nil {
		logger.Error(component, "Failed to fetch data for scheduled report", logger.Fields(
			"error", err.Error(),
			"scheduled_time", now.Format("15:04"),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	results, err := s.calculator.CalculateBatch(ctx, records)
	if err != nil {
		logger.Error(component, "Failed to calculate debit for scheduled report", logger.Fields(
			"error", err.Error(),
			"records", len(records),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	if err := s.telegram.SendDebitReport(ctx, results); err != nil {
		logger.Error(component, "Failed to send Telegram report", logger.Fields(
			"error", err.Error(),
			"stations", len(results),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	validCount := 0
	for _, r := range results {
		if r.IsValid {
			validCount++
		}
	}

	logger.Info(component, "Telegram text report sent", logger.Fields(
		"scheduled_time", now.Format("15:04 WIB"),
		"total_stations", len(results),
		"valid_debit", validCount,
		"duration_ms", time.Since(startTime).Milliseconds(),
	))
}

func (s *Scheduler) sendDailyExcelReport() {
	if s.telegram == nil {
		logger.Warn(component, "Telegram not configured, skipping daily Excel report")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	now := time.Now().In(jakartaLoc)

	logger.Info(component, "Starting daily Excel report generation", logger.F("date", now.Format("2006-01-02")))

	stations, err := s.stationRepo.GetAll(ctx)
	if err != nil {
		logger.Error(component, "Failed to fetch stations for daily report", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	stationMap := make(map[string]string)
	for _, st := range stations {
		stationMap[st.NamaLokasi] = st.NamaAlat
	}

	tmaSummary, err := s.readingService.GetDailyTMASummary(ctx, now)
	if err != nil {
		logger.Error(component, "Failed to fetch TMA summary", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	debitSnapshots, err := s.readingService.GetDebitSnapshots(ctx, now, []int{7, 12, 17})
	if err != nil {
		logger.Error(component, "Failed to fetch debit snapshots", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	var reports []domain.DailyStationReport

	stationNames := make(map[string]bool)
	for name := range tmaSummary {
		stationNames[name] = true
	}
	for name := range debitSnapshots {
		stationNames[name] = true
	}

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

	logger.Debug(component, "Daily report data collected", logger.Fields(
		"stations", len(reports),
		"tma_records", len(tmaSummary),
		"debit_snapshots", len(debitSnapshots),
	))

	excelBuf, filename, err := s.excelService.GenerateDailyReport(reports, now)
	if err != nil {
		logger.Error(component, "Failed to generate Excel file", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	logger.Debug(component, "Excel file generated", logger.Fields(
		"filename", filename,
		"size_bytes", excelBuf.Len(),
	))

	if err := s.telegram.SendDailyExcelReport(ctx, reports, excelBuf, filename); err != nil {
		logger.Error(component, "Failed to send daily Excel report to Telegram", logger.Fields(
			"error", err.Error(),
			"filename", filename,
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	logger.Info(component, "Daily Excel report sent successfully", logger.Fields(
		"date", now.Format("2006-01-02"),
		"stations", len(reports),
		"filename", filename,
		"duration_ms", time.Since(startTime).Milliseconds(),
	))
}
