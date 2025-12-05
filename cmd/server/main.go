package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    _ "github.com/go-sql-driver/mysql"
    "github.com/jmoiron/sqlx"

    "pda-monitor/internal/auth"
    "pda-monitor/internal/config"
    "pda-monitor/internal/handler"
    "pda-monitor/internal/notification"
    "pda-monitor/internal/parser"
    "pda-monitor/internal/report"
    mysqlrepo "pda-monitor/internal/repository/mysql"
    "pda-monitor/internal/scheduler"
    "pda-monitor/internal/service"
)

func main() {
    cfg := config.Load()

    log.Println("Starting PDA Monitor...")
    log.Printf("  Database: %s@%s:%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
    log.Printf("  Telemetry API: %s", cfg.Telemetry.BaseURL)

    // Database connection
    db, err := sqlx.Connect("mysql", cfg.Database.DSN())
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Repositories
    stationRepo := mysqlrepo.NewStationRepo(db)
    formulaRepo := mysqlrepo.NewFormulaRepo(db)
    readingRepo := mysqlrepo.NewReadingRepo(db)
    userRepo := mysqlrepo.NewUserRepo(db)

    // JWT Manager (auto-generates 256-bit key, rotates every 24h)
    jwtManager := auth.NewJWTManager(cfg.JWT.ExpiryHours)
    defer jwtManager.Stop()

    // Services
    telemetryParser := parser.NewTelemetryParser()

    calculator := service.NewDebitCalculator(formulaRepo)
    if err := calculator.RefreshCache(context.Background()); err != nil {
        log.Printf("Warning: failed to preload formula cache: %v", err)
    }

    telemetryService := service.NewTelemetryService(
        service.TelemetryConfig{
            BaseURL:    cfg.Telemetry.BaseURL,
            IDBBWS:     cfg.Telemetry.IDBBWS,
            TimeoutSec: cfg.Telemetry.TimeoutSec,
        },
        telemetryParser,
        stationRepo,
    )

    readingService := service.NewReadingService(readingRepo, calculator)

    // Excel Report Service
    excelService := report.NewExcelReportService()

    // Initial sync on startup
    go func() {
        ctx := context.Background()

        if count, err := telemetryService.SyncStations(ctx); err != nil {
            log.Printf("Warning: initial station sync failed: %v", err)
        } else {
            log.Printf("Initial station sync: %d stations", count)
        }

        records, err := telemetryService.FetchRealtime(ctx)
        if err != nil {
            log.Printf("Warning: initial readings sync failed: %v", err)
        } else {
            if count, err := readingService.ProcessAndStoreReadings(ctx, records); err != nil {
                log.Printf("Warning: storing initial readings failed: %v", err)
            } else {
                log.Printf("Initial readings sync: %d readings", count)
            }
        }
    }()

    // Telegram notifier
    var telegram *notification.TelegramNotifier
    if cfg.Telegram.BotToken != "" && (len(cfg.Telegram.ChatIDs) > 0 || len(cfg.Telegram.Channels) > 0) {
        telegram, err = notification.NewTelegramNotifier(notification.TelegramConfig{
            BotToken: cfg.Telegram.BotToken,
            ChatIDs:  cfg.Telegram.ChatIDs,
            Channels: cfg.Telegram.Channels,
        })
        if err != nil {
            log.Printf("Warning: failed to create Telegram notifier: %v", err)
        } else {
            log.Println("Telegram notifier configured")
        }
    } else {
        log.Println("Warning: Telegram not configured")
    }

    // Scheduler
    sched := scheduler.NewScheduler(
        telemetryService,
        readingService,
        calculator,
        telegram,
    )
    if err := sched.Start(); err != nil {
        log.Fatalf("Failed to start scheduler: %v", err)
    }
    defer sched.Stop()

    // HTTP Handler
    apiHandler := handler.NewAPIHandler(
        telemetryService,
        readingService,
        calculator,
        stationRepo,
        formulaRepo,
        userRepo,
        jwtManager,
        excelService,
    )

    router := mux.NewRouter()
    apiHandler.RegisterRoutes(router)
    router.Use(loggingMiddleware, corsMiddleware)

    router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }).Methods("GET")

    server := &http.Server{
        Addr:         ":" + cfg.Server.Port,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
    }

    go func() {
        log.Printf("Server starting on port %s", cfg.Server.Port)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    server.Shutdown(ctx)
    log.Println("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
    })
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}