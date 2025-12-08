package scheduler

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/robfig/cron/v3"

    "pda-monitor/internal/domain"
    "pda-monitor/internal/notification"
    "pda-monitor/internal/report"
    "pda-monitor/internal/repository"
    "pda-monitor/internal/service"
    "pda-monitor/internal/util"
)

const HoursToKeep = 24

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
        return err
    }

    if _, err := s.cron.AddFunc("30 0 * * *", s.cleanupOldReadings); err != nil {
        return err
    }

    if _, err := s.cron.AddFunc("0 * * * *", s.syncStationMetadata); err != nil {
        return err
    }

    for _, hour := range []int{7, 12, 17} {
        h := hour
        if _, err := s.cron.AddFunc(fmt.Sprintf("0 %d * * *", h), s.sendScheduledReport); err != nil {
            return err
        }
    }

    if _, err := s.cron.AddFunc("10 17 * * *", s.sendDailyExcelReport); err != nil {
        return err
    }

    s.cron.Start()

    log.Println("Scheduler started (Asia/Jakarta timezone):")
    log.Println("  - Readings sync: every 5 minutes")
    log.Println("  - Cleanup: daily at 00:30 WIB (keeps 24h)")
    log.Println("  - Station sync: every hour at :00")
    log.Println("  - Telegram text: 07:00, 12:00, 17:00 WIB")
    log.Println("  - Daily Excel report: 17:10 WIB")

    return nil
}

func (s *Scheduler) Stop() {
    ctx := s.cron.Stop()
    <-ctx.Done()
    log.Println("Scheduler stopped")
}

func (s *Scheduler) syncReadings() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    records, err := s.telemetryService.FetchRealtime(ctx)
    if err != nil {
        log.Printf("[Scheduler] Error fetching realtime: %v", err)
        return
    }

    count, err := s.readingService.ProcessAndStoreReadings(ctx, records)
    if err != nil {
        log.Printf("[Scheduler] Error storing readings: %v", err)
        return
    }

    log.Printf("[Scheduler] Synced %d readings", count)
}

func (s *Scheduler) cleanupOldReadings() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    deleted, err := s.readingService.CleanupOldData(ctx, HoursToKeep)
    if err != nil {
        log.Printf("[Scheduler] Error cleanup: %v", err)
        return
    }

    if deleted > 0 {
        log.Printf("[Scheduler] Cleaned %d old readings (kept last %d hours)", deleted, HoursToKeep)
    }
}

func (s *Scheduler) syncStationMetadata() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    count, err := s.telemetryService.SyncStations(ctx)
    if err != nil {
        log.Printf("[Scheduler] Error syncing stations: %v", err)
        return
    }

    log.Printf("[Scheduler] Synced %d stations", count)
}

func (s *Scheduler) sendScheduledReport() {
    if s.telegram == nil {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    records, err := s.telemetryService.FetchRealtime(ctx)
    if err != nil {
        log.Printf("[Scheduler] Error fetching for report: %v", err)
        return
    }

    results, err := s.calculator.CalculateBatch(ctx, records)
    if err != nil {
        log.Printf("[Scheduler] Error calculating: %v", err)
        return
    }

    if err := s.telegram.SendDebitReport(ctx, results); err != nil {
        log.Printf("[Scheduler] Error sending telegram: %v", err)
        return
    }

    log.Printf("[Scheduler] Telegram text report sent (%d stations)", len(results))
}

func (s *Scheduler) sendDailyExcelReport() {
    if s.telegram == nil {
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
    defer cancel()

    now := time.Now().In(jakartaLoc)

    // Get all stations for name mapping
    stations, err := s.stationRepo.GetAll(ctx)
    if err != nil {
        log.Printf("[Scheduler] Error fetching stations: %v", err)
        return
    }

    stationMap := make(map[string]string)
    for _, st := range stations {
        stationMap[st.NamaLokasi] = st.NamaAlat
    }

    tmaSummary, err := s.readingService.GetDailyTMASummary(ctx, now)
    if err != nil {
        log.Printf("[Scheduler] Error fetching TMA summary: %v", err)
        return
    }

    debitSnapshots, err := s.readingService.GetDebitSnapshots(ctx, now, []int{7, 12, 17})
    if err != nil {
        log.Printf("[Scheduler] Error fetching debit snapshots: %v", err)
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

    excelBuf, filename, err := s.excelService.GenerateDailyReport(reports, now)
    if err != nil {
        log.Printf("[Scheduler] Error generating Excel: %v", err)
        return
    }

    if err := s.telegram.SendDailyExcelReport(ctx, reports, excelBuf, filename); err != nil {
        log.Printf("[Scheduler] Error sending daily Excel report: %v", err)
        return
    }

    log.Printf("[Scheduler] Daily Excel report sent (%d stations)", len(reports))
}