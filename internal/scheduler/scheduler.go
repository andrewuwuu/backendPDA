package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"pda-monitor/internal/logger"
	"pda-monitor/internal/notification"
	"pda-monitor/internal/report"
	"pda-monitor/internal/service"
	"pda-monitor/internal/timeutil"
)

const component = "Scheduler"
const HoursToKeep = 168

type Scheduler struct {
	cron             *cron.Cron
	telemetryService *service.TelemetryService
	readingService   *service.ReadingService
	reportService    *service.ReportService
	calculator       *service.DebitCalculator
	telegram         *notification.TelegramNotifier
	excelService     *report.ExcelReportService
}

func NewScheduler(
	telemetryService *service.TelemetryService,
	readingService *service.ReadingService,
	reportService *service.ReportService,
	calculator *service.DebitCalculator,
	telegram *notification.TelegramNotifier,
	excelService *report.ExcelReportService,
) *Scheduler {
	return &Scheduler{
		cron:             cron.New(cron.WithLocation(timeutil.JakartaLocation())),
		telemetryService: telemetryService,
		readingService:   readingService,
		reportService:    reportService,
		calculator:       calculator,
		telegram:         telegram,
		excelService:     excelService,
	}
}

func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc("*/5 * * * *", s.syncReadings); err != nil {
		return fmt.Errorf("failed to schedule syncReadings: %w", err)
	}

	if _, err := s.cron.AddFunc("30 0 * * *", s.cleanupOldReadings); err != nil {
		return fmt.Errorf("failed to schedule cleanupOldReadings: %w", err)
	}

	for _, hour := range []int{7, 12, 17} {
		h := hour
		if _, err := s.cron.AddFunc(fmt.Sprintf("6 %d * * *", h), s.sendScheduledReport); err != nil {
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
		"telegram_text", "07:06, 12:06, 17:06 WIB",
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

	if startTime.Minute() == 0 {
		if _, err := s.telemetryService.SyncStations(ctx, records); err != nil {
			logger.Error(component, "Failed to sync station metadata", logger.Fields(
				"error", err.Error(),
			))
		}
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

	logMemoryStats("syncReadings")
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

func (s *Scheduler) sendScheduledReport() {
	if s.telegram == nil {
		logger.Warn(component, "Telegram not configured, skipping scheduled report")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	startTime := time.Now()
	now := timeutil.NowJakarta()

	results, err := s.reportService.BuildScheduledReport(ctx)
	if err != nil {
		logger.Error(component, "Failed to fetch latest readings for scheduled report", logger.Fields(
			"error", err.Error(),
			"scheduled_time", now.Format("15:04"),
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
	var missingNames []string
	for _, r := range results {
		if r.IsValid {
			validCount++
		} else {
			missingNames = append(missingNames, r.NamaLokasi)
		}
	}

	logger.Info(component, "Telegram text report sent", logger.Fields(
		"scheduled_time", now.Format("15:04 WIB"),
		"total_stations", len(results),
		"valid_debit", validCount,
		"missing_stations", strings.Join(missingNames, ", "),
		"duration_ms", time.Since(startTime).Milliseconds(),
	))

	logMemoryStats("sendScheduledReport")
}

func (s *Scheduler) sendDailyExcelReport() {
	if s.telegram == nil {
		logger.Warn(component, "Telegram not configured, skipping daily Excel report")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	now := timeutil.NowJakarta()

	logger.Info(component, "Starting daily Excel report generation", logger.F("date", now.Format("2006-01-02")))

	reports, err := s.reportService.BuildDailyReport(ctx, now)
	if err != nil {
		logger.Error(component, "Failed to build daily report data", logger.Fields(
			"error", err.Error(),
			"duration_ms", time.Since(startTime).Milliseconds(),
		))
		return
	}

	logger.Debug(component, "Daily report data collected", logger.Fields(
		"stations", len(reports),
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

	validDebitCount := 0
	for _, r := range reports {
		if r.Debit07 != nil || r.Debit12 != nil || r.Debit17 != nil {
			validDebitCount++
		}
	}

	logger.Info(component, "Daily Excel report sent successfully", logger.Fields(
		"date", now.Format("2006-01-02"),
		"stations", len(reports),
		"valid_debit_stations", validDebitCount,
		"filename", filename,
		"duration_ms", time.Since(startTime).Milliseconds(),
	))

	logMemoryStats("sendDailyExcelReport")
}

func logMemoryStats(after string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.Debug(component, "Memory stats", logger.Fields(
		"after", after,
		"alloc_mb", float64(m.Alloc)/(1024*1024),
		"heap_inuse_mb", float64(m.HeapInuse)/(1024*1024),
		"heap_objects", m.HeapObjects,
		"num_gc", m.NumGC,
	))
}
