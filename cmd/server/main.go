// backendPDA/cmd/server/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"

	"pda-monitor/internal/auth"
	"pda-monitor/internal/config"
	"pda-monitor/internal/handler"
	"pda-monitor/internal/logger"
	"pda-monitor/internal/notification"
	"pda-monitor/internal/parser"
	"pda-monitor/internal/report"
	mysqlrepo "pda-monitor/internal/repository/mysql"
	"pda-monitor/internal/scheduler"
	"pda-monitor/internal/service"
)

const component = "Main"

func main() {
	cfg := config.Load()

	if err := logger.Init(logger.Config{
		Level:    cfg.Logging.Level,
		FilePath: cfg.Logging.FilePath,
		Console:  cfg.Logging.Console,
	}); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Close()

	logger.Info(component, "Starting PDA Monitor", logger.Fields(
		"db_host", cfg.Database.Host,
		"db_port", cfg.Database.Port,
		"db_name", cfg.Database.Name,
		"telemetry_url", cfg.Telemetry.BaseURL,
		"log_level", cfg.Logging.Level,
	))

	db, err := sqlx.Connect("mysql", cfg.Database.DSN())
	if err != nil {
		logger.Fatal(component, "Failed to connect to database", logger.F("error", err.Error()))
	}
	defer db.Close()
	logger.Info(component, "Database connected successfully")

	stationRepo := mysqlrepo.NewStationRepo(db)
	formulaRepo := mysqlrepo.NewFormulaRepo(db)
	readingRepo := mysqlrepo.NewReadingRepo(db)
	userRepo := mysqlrepo.NewUserRepo(db)
	alertLevelRepo := mysqlrepo.NewAlertLevelRepo(db)

	jwtManager := auth.NewJWTManager(cfg.JWT.ExpiryHours)
	defer jwtManager.Stop()
	logger.Info(component, "JWT Manager initialized", logger.F("expiry_hours", cfg.JWT.ExpiryHours))

	telemetryParser := parser.NewTelemetryParser()

	calculator := service.NewDebitCalculator(formulaRepo)
	if err := calculator.RefreshCache(context.Background()); err != nil {
		logger.Warn(component, "Failed to preload formula cache", logger.F("error", err.Error()))
	} else {
		logger.Info(component, "Formula cache loaded")
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
	excelService := report.NewExcelReportService()

	go func() {
		ctx := context.Background()

		if count, err := telemetryService.SyncStations(ctx); err != nil {
			logger.Warn(component, "Initial station sync failed", logger.F("error", err.Error()))
		} else {
			logger.Info(component, "Initial station sync completed", logger.F("count", count))
		}

		records, err := telemetryService.FetchRealtime(ctx)
		if err != nil {
			logger.Warn(component, "Initial readings sync failed", logger.F("error", err.Error()))
		} else {
			if count, err := readingService.ProcessAndStoreReadings(ctx, records); err != nil {
				logger.Warn(component, "Storing initial readings failed", logger.F("error", err.Error()))
			} else {
				logger.Info(component, "Initial readings sync completed", logger.F("count", count))
			}
		}
	}()

	var telegram *notification.TelegramNotifier
	if cfg.Telegram.BotToken != "" && (len(cfg.Telegram.ChatIDs) > 0 || len(cfg.Telegram.Channels) > 0) {
		telegram, err = notification.NewTelegramNotifier(notification.TelegramConfig{
			BotToken: cfg.Telegram.BotToken,
			ChatIDs:  cfg.Telegram.ChatIDs,
			Channels: cfg.Telegram.Channels,
		})
		if err != nil {
			logger.Warn(component, "Failed to create Telegram notifier", logger.F("error", err.Error()))
		} else {
			logger.Info(component, "Telegram notifier configured", logger.Fields(
				"chat_ids", len(cfg.Telegram.ChatIDs),
				"channels", len(cfg.Telegram.Channels),
			))
		}
	} else {
		logger.Warn(component, "Telegram not configured")
	}

	sched := scheduler.NewScheduler(
		telemetryService,
		readingService,
		calculator,
		telegram,
		excelService,
		stationRepo,
	)
	if err := sched.Start(); err != nil {
		logger.Fatal(component, "Failed to start scheduler", logger.F("error", err.Error()))
	}
	defer sched.Stop()

	apiHandler := handler.NewAPIHandler(
		telemetryService,
		readingService,
		calculator,
		stationRepo,
		formulaRepo,
		userRepo,
		alertLevelRepo,
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
		logger.Info(component, "HTTP server starting", logger.F("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatal(component, "Server error", logger.F("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info(component, "Shutdown signal received", logger.F("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error(component, "Server shutdown error", logger.F("error", err.Error()))
	}

	logger.Info(component, "Server stopped gracefully")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logger.Info("HTTP", "Request completed", logger.Fields(
			"method", r.Method,
			"path", r.RequestURI,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		))
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
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
