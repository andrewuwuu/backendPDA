package scheduler

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/robfig/cron/v3"

    "pda-monitor/internal/notification"
    "pda-monitor/internal/service"
)

const HoursToKeep = 2

type Scheduler struct {
    cron             *cron.Cron
    telemetryService *service.TelemetryService
    readingService   *service.ReadingService
    calculator       *service.DebitCalculator
    telegram         *notification.TelegramNotifier
}

func NewScheduler(
    telemetryService *service.TelemetryService,
    readingService *service.ReadingService,
    calculator *service.DebitCalculator,
    telegram *notification.TelegramNotifier,
) *Scheduler {
    loc, _ := time.LoadLocation("Asia/Jakarta")

    return &Scheduler{
        cron:             cron.New(cron.WithLocation(loc)),
        telemetryService: telemetryService,
        readingService:   readingService,
        calculator:       calculator,
        telegram:         telegram,
    }
}

func (s *Scheduler) Start() error {
    // Sync readings every 5 minutes
    _, err := s.cron.AddFunc("*/5 * * * *", s.syncReadings)
    if err != nil {
        return err
    }

    // Cleanup old readings at :01 every hour
    _, err = s.cron.AddFunc("1 * * * *", s.cleanupOldReadings)
    if err != nil {
        return err
    }

    // Sync station metadata every hour at :00
    _, err = s.cron.AddFunc("0 * * * *", s.syncStationMetadata)
    if err != nil {
        return err
    }

    // Telegram at 07:00, 12:00, 17:00 WIB
    for _, hour := range []int{7, 12, 17} {
        h := hour
        _, err = s.cron.AddFunc(fmt.Sprintf("0 %d * * *", h), s.sendScheduledReport)
        if err != nil {
            return err
        }
    }

    s.cron.Start()

    log.Println("Scheduler started:")
    log.Println("  - Readings sync: every 5 minutes")
    log.Println("  - Cleanup: every hour at :01")
    log.Println("  - Station sync: every hour at :00")
    log.Println("  - Telegram: 07:00, 12:00, 17:00 WIB")

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
        log.Printf("[Scheduler] Cleaned %d old readings", deleted)
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

    log.Printf("[Scheduler] Telegram sent (%d stations)", len(results))
}