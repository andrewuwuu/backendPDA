package handler

import (
	"github.com/go-chi/chi/v5"

	"pda-monitor/internal/auth"
	"pda-monitor/internal/middleware"
	"pda-monitor/internal/report"
	"pda-monitor/internal/repository"
	"pda-monitor/internal/service"
)

// APIHandler holds all dependencies for HTTP handler methods.
type APIHandler struct {
	telemetryService *service.TelemetryService
	readingService   *service.ReadingService
	reportService    *service.ReportService
	calculator       *service.DebitCalculator
	stationRepo      repository.StationRepository
	formulaRepo      repository.FormulaRepository
	userRepo         repository.UserRepository
	alertLevelRepo   repository.AlertLevelRepository
	jwtManager       *auth.JWTManager
	excelService     *report.ExcelReportService
}

func NewAPIHandler(
	ts *service.TelemetryService,
	rs *service.ReadingService,
	reportSvc *service.ReportService,
	calc *service.DebitCalculator,
	stationRepo repository.StationRepository,
	formulaRepo repository.FormulaRepository,
	userRepo repository.UserRepository,
	alertLevelRepo repository.AlertLevelRepository,
	jwtManager *auth.JWTManager,
	excelService *report.ExcelReportService,
) *APIHandler {
	return &APIHandler{
		telemetryService: ts,
		readingService:   rs,
		reportService:    reportSvc,
		calculator:       calc,
		stationRepo:      stationRepo,
		formulaRepo:      formulaRepo,
		userRepo:         userRepo,
		alertLevelRepo:   alertLevelRepo,
		jwtManager:       jwtManager,
		excelService:     excelService,
	}
}

// RegisterRoutes registers all API routes on the given Chi router.
func (h *APIHandler) RegisterRoutes(r chi.Router) {
	authMiddleware := middleware.NewAuthMiddleware(h.jwtManager)

	r.Route("/api", func(api chi.Router) {
		api.Post("/auth/login", h.Login)

		api.Group(func(protected chi.Router) {
			protected.Use(authMiddleware.Authenticate)

			protected.Post("/auth/logout", h.Logout)

			h.registerReadingRoutes(protected)
			h.registerStationRoutes(protected)
			h.registerFormulaRoutes(protected)
			h.registerAlertLevelRoutes(protected)
			h.registerExportRoutes(protected)

			h.registerAdminRoutes(protected, authMiddleware)
		})
	})
}

func (h *APIHandler) registerReadingRoutes(r chi.Router) {
	r.Get("/pda/realtime", h.GetRealtimeWithDebit)
	r.Get("/pda/historical", h.GetHistoricalWithDebit)

	r.Get("/readings/current", h.GetCurrentHourReadings)
	r.Get("/readings/current/summary", h.GetCurrentHourSummary)
	r.Get("/readings/current/latest", h.GetLatestReadings)
	r.Get("/readings/station/{namaLokasi}", h.GetStationReadings)
	r.Get("/readings/historical", h.GetHistoricalReadings)
}

func (h *APIHandler) registerStationRoutes(r chi.Router) {
	r.Get("/stations", h.GetStations)
	r.Post("/stations/sync", h.SyncStations)
	r.Get("/stations/{namaLokasi}", h.GetStation)
}

func (h *APIHandler) registerFormulaRoutes(r chi.Router) {
	r.Get("/formulas", h.GetFormulas)
	r.Get("/formulas/grouped", h.GetFormulasGrouped)
	r.Get("/formulas/id/{id}", h.GetFormulaByID)
	r.Get("/formulas/{namaLokasi}", h.GetFormulasByStation)
}

func (h *APIHandler) registerAlertLevelRoutes(r chi.Router) {
	r.Get("/alert-levels", h.GetAllAlertLevels)
	r.Get("/alert-levels/filter/{level}", h.GetAlertLevelsByLevel)
	r.Get("/alert-levels/{namaLokasi}", h.GetAlertLevel)
}

func (h *APIHandler) registerExportRoutes(r chi.Router) {
	r.Get("/export/daily", h.ExportDailyReport)
	r.Get("/export/weekly", h.ExportWeeklyReports)
}

func (h *APIHandler) registerAdminRoutes(r chi.Router, authMiddleware *middleware.AuthMiddleware) {
	r.Group(func(admin chi.Router) {
		admin.Use(authMiddleware.RequireRole("admin"))

		admin.Post("/formulas", h.CreateFormulas)
		admin.Put("/formulas/id/{id}", h.UpdateFormulaByID)
		admin.Delete("/formulas/id/{id}", h.DeleteFormulaByID)
		admin.Put("/formulas/{namaLokasi}", h.UpdateStationFormulas)
		admin.Delete("/formulas/{namaLokasi}", h.DeleteStationFormulas)

		admin.Put("/alert-levels/{namaLokasi}", h.UpdateAlertLevel)
		admin.Put("/alert-levels", h.BulkUpdateAlertLevels)
		admin.Delete("/alert-levels/{namaLokasi}", h.DeleteAlertLevel)

		admin.Get("/debug/jwt", h.GetJWTInfo)
	})
}
