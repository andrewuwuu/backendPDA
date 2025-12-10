package handler

import (
    "encoding/json"
    "net/http"
    "strconv"
    "time"

    "github.com/gorilla/mux"

    "pda-monitor/internal/auth"
    "pda-monitor/internal/domain"
    "pda-monitor/internal/middleware"
    "pda-monitor/internal/report"
    "pda-monitor/internal/repository"
    "pda-monitor/internal/service"
)

type APIHandler struct {
    telemetryService *service.TelemetryService
    readingService   *service.ReadingService
    calculator       *service.DebitCalculator
    stationRepo      repository.StationRepository
    formulaRepo      repository.FormulaRepository
    userRepo         repository.UserRepository
    jwtManager       *auth.JWTManager
    excelService     *report.ExcelReportService
}

func NewAPIHandler(
    ts *service.TelemetryService,
    rs *service.ReadingService,
    calc *service.DebitCalculator,
    stationRepo repository.StationRepository,
    formulaRepo repository.FormulaRepository,
    userRepo repository.UserRepository,
    jwtManager *auth.JWTManager,
    excelService *report.ExcelReportService,
) *APIHandler {
    return &APIHandler{
        telemetryService: ts,
        readingService:   rs,
        calculator:       calc,
        stationRepo:      stationRepo,
        formulaRepo:      formulaRepo,
        userRepo:         userRepo,
        jwtManager:       jwtManager,
        excelService:     excelService,
    }
}

func (h *APIHandler) RegisterRoutes(r *mux.Router) {
    authMiddleware := middleware.NewAuthMiddleware(h.jwtManager)

    api := r.PathPrefix("/api").Subrouter()

    // Public routes
    api.HandleFunc("/auth/login", h.Login).Methods("POST")

    // Protected routes
    protected := api.PathPrefix("").Subrouter()
    protected.Use(authMiddleware.Authenticate)

    // PDA endpoints
    protected.HandleFunc("/pda/realtime", h.GetRealtimeWithDebit).Methods("GET")
    protected.HandleFunc("/pda/historical", h.GetHistoricalWithDebit).Methods("GET")

    // Readings endpoints (with debit)
    protected.HandleFunc("/readings/current", h.GetCurrentHourReadings).Methods("GET")
    protected.HandleFunc("/readings/current/summary", h.GetCurrentHourSummary).Methods("GET")
    protected.HandleFunc("/readings/current/latest", h.GetLatestReadings).Methods("GET")
    protected.HandleFunc("/readings/station/{namaLokasi}", h.GetStationReadings).Methods("GET")
    protected.HandleFunc("/readings/historical", h.GetHistoricalReadings).Methods("GET")

    // Stations
    protected.HandleFunc("/stations", h.GetStations).Methods("GET")
    protected.HandleFunc("/stations/{namaLokasi}", h.GetStation).Methods("GET")
    protected.HandleFunc("/stations/sync", h.SyncStations).Methods("POST")

    // Formulas - GET (multiple formulas support)
    protected.HandleFunc("/formulas", h.GetFormulas).Methods("GET")
    protected.HandleFunc("/formulas/grouped", h.GetFormulasGrouped).Methods("GET")
    protected.HandleFunc("/formulas/{namaLokasi}", h.GetFormulasByStation).Methods("GET")
    protected.HandleFunc("/formulas/id/{id}", h.GetFormulaByID).Methods("GET")

    // Admin routes
    adminRoutes := protected.PathPrefix("").Subrouter()
    adminRoutes.Use(authMiddleware.RequireRole("admin"))
    
    // Formulas - Admin (supports multiple formulas per station)
    adminRoutes.HandleFunc("/formulas", h.CreateFormulas).Methods("POST")
    adminRoutes.HandleFunc("/formulas/{namaLokasi}", h.UpdateStationFormulas).Methods("PUT")
    adminRoutes.HandleFunc("/formulas/{namaLokasi}", h.DeleteStationFormulas).Methods("DELETE")
    adminRoutes.HandleFunc("/formulas/id/{id}", h.UpdateFormulaByID).Methods("PUT")
    adminRoutes.HandleFunc("/formulas/id/{id}", h.DeleteFormulaByID).Methods("DELETE")

    // Report endpoints
    protected.HandleFunc("/reports/export", h.ExportReport).Methods("GET")

    // Debug endpoint (admin only)
    adminRoutes.HandleFunc("/debug/jwt", h.GetJWTInfo).Methods("GET")
}

// ==================== Auth ====================

func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req domain.LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    user, err := h.userRepo.GetByUsername(ctx, req.Username)
    if err != nil || user == nil {
        h.jsonError(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    if !h.userRepo.ValidatePassword(user, req.Password) {
        h.jsonError(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
    if err != nil {
        h.jsonError(w, "failed to generate token", http.StatusInternalServerError)
        return
    }

    h.jsonResponse(w, domain.LoginResponse{
        Token: token,
        User:  *user,
    })
}

// ==================== Debug ====================

func (h *APIHandler) GetJWTInfo(w http.ResponseWriter, r *http.Request) {
    h.jsonResponse(w, h.jwtManager.GetKeyInfo())
}

// ==================== Reports ====================

func (h *APIHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    records, err := h.telemetryService.FetchRealtime(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    results, err := h.calculator.CalculateBatch(ctx, records)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    now := time.Now()
    buf, filename, err := h.excelService.GenerateDebitReport(results, now)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", "attachment; filename="+filename)
    w.Write(buf.Bytes())
}

// ==================== Readings ====================

func (h *APIHandler) GetCurrentHourReadings(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    readings, err := h.readingService.GetCurrentHourData(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, map[string]interface{}{
        "hour_bucket":   truncateToHour(time.Now()),
        "reading_count": len(readings),
        "readings":      readings,
    })
}

func (h *APIHandler) GetCurrentHourSummary(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    summaries, err := h.readingService.GetCurrentHourSummary(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, map[string]interface{}{
        "hour_bucket":   truncateToHour(time.Now()),
        "station_count": len(summaries),
        "summaries":     summaries,
    })
}

func (h *APIHandler) GetLatestReadings(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    readings, err := h.readingService.GetLatestReadings(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, map[string]interface{}{
        "hour_bucket":     truncateToHour(time.Now()),
        "station_count":   len(readings),
        "latest_readings": readings,
    })
}

func (h *APIHandler) GetStationReadings(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := mux.Vars(r)["namaLokasi"]
    readings, err := h.readingService.GetCurrentHourByStation(ctx, namaLokasi)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, map[string]interface{}{
        "nama_lokasi":   namaLokasi,
        "hour_bucket":   truncateToHour(time.Now()),
        "reading_count": len(readings),
        "readings":      readings,
    })
}

func (h *APIHandler) GetHistoricalReadings(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    namaLokasi := r.URL.Query().Get("nama_lokasi")
    fromStr := r.URL.Query().Get("from")
    toStr := r.URL.Query().Get("to")

    if fromStr == "" || toStr == "" {
        h.jsonError(w, "from and to dates are required (format: 2006-01-02T15:04:05)", http.StatusBadRequest)
        return
    }

    from, err := time.Parse("2006-01-02T15:04:05", fromStr)
    if err != nil {
        from, err = time.Parse("2006-01-02", fromStr)
        if err != nil {
            h.jsonError(w, "invalid from date format", http.StatusBadRequest)
            return
        }
    }

    to, err := time.Parse("2006-01-02T15:04:05", toStr)
    if err != nil {
        to, err = time.Parse("2006-01-02", toStr)
        if err != nil {
            h.jsonError(w, "invalid to date format", http.StatusBadRequest)
            return
        }
        to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
    }

    var readings []domain.HourlyReading
    if namaLokasi != "" {
        readings, err = h.readingService.GetReadingsByTimeRange(ctx, namaLokasi, from, to)
    } else {
        readings, err = h.readingService.GetAllReadingsByTimeRange(ctx, from, to)
    }

    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.jsonResponse(w, map[string]interface{}{
        "from":          from,
        "to":            to,
        "nama_lokasi":   namaLokasi,
        "reading_count": len(readings),
        "readings":      readings,
    })
}

// ==================== PDA ====================

func (h *APIHandler) GetRealtimeWithDebit(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    records, err := h.telemetryService.FetchRealtime(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    results, err := h.calculator.CalculateBatch(ctx, records)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, results)
}

func (h *APIHandler) GetHistoricalWithDebit(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := r.URL.Query().Get("nama_lokasi")
    if namaLokasi == "" {
        h.jsonError(w, "nama_lokasi is required", http.StatusBadRequest)
        return
    }

    from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
    if err != nil {
        h.jsonError(w, "invalid from date", http.StatusBadRequest)
        return
    }
    to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
    if err != nil {
        h.jsonError(w, "invalid to date", http.StatusBadRequest)
        return
    }

    records, err := h.telemetryService.FetchHistorical(ctx, namaLokasi, from, to)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    results, err := h.calculator.CalculateBatch(ctx, records)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, results)
}

// ==================== Stations ====================

func (h *APIHandler) GetStations(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    stations, err := h.stationRepo.GetAll(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, stations)
}

func (h *APIHandler) GetStation(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := mux.Vars(r)["namaLokasi"]
    station, err := h.stationRepo.GetByNamaLokasi(ctx, namaLokasi)
    if err != nil || station == nil {
        h.jsonError(w, "station not found", http.StatusNotFound)
        return
    }
    h.jsonResponse(w, station)
}

func (h *APIHandler) SyncStations(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    count, err := h.telemetryService.SyncStations(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, map[string]interface{}{"status": "synced", "count": count})
}

// ==================== Formulas ====================

// GetFormulas returns all formulas (flat list)
func (h *APIHandler) GetFormulas(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    formulas, err := h.formulaRepo.GetAll(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, formulas)
}

// GetFormulasGrouped returns formulas grouped by station
func (h *APIHandler) GetFormulasGrouped(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    grouped, err := h.formulaRepo.GetAllGroupedByStation(ctx)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, grouped)
}

// GetFormulasByStation returns all formulas for a specific station
func (h *APIHandler) GetFormulasByStation(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := mux.Vars(r)["namaLokasi"]
    formulas, err := h.formulaRepo.GetAllByNamaLokasi(ctx, namaLokasi)
    if err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    if len(formulas) == 0 {
        h.jsonError(w, "no formulas found for station", http.StatusNotFound)
        return
    }

    h.jsonResponse(w, domain.StationFormulas{
        NamaLokasi:  namaLokasi,
        StationName: formulas[0].StationName,
        Formulas:    formulas,
    })
}

// GetFormulaByID returns a specific formula by ID
func (h *APIHandler) GetFormulaByID(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    idStr := mux.Vars(r)["id"]
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        h.jsonError(w, "invalid formula id", http.StatusBadRequest)
        return
    }

    formula, err := h.formulaRepo.GetByID(ctx, id)
    if err != nil {
        h.jsonError(w, "formula not found", http.StatusNotFound)
        return
    }
    h.jsonResponse(w, formula)
}

// CreateFormulas creates one or more formulas for a station
// Accepts either:
// - Single formula: {"nama_lokasi": "...", "c": ..., ...}
// - Multiple formulas: {"nama_lokasi": "...", "station_name": "...", "formulas": [...]}
func (h *APIHandler) CreateFormulas(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var batchReq domain.FormulaCreateRequest
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&batchReq); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if len(batchReq.Formulas) > 0 {
        if batchReq.NamaLokasi == "" {
            h.jsonError(w, "nama_lokasi is required", http.StatusBadRequest)
            return
        }

        formulas := make([]domain.FormulaParams, len(batchReq.Formulas))
        for i, f := range batchReq.Formulas {
            formulas[i] = domain.FormulaParams{
                NamaLokasi:      batchReq.NamaLokasi,
                StationName:     batchReq.StationName,
                C:               f.C,
                H0:              f.H0,
                B:               f.B,
                TMAMin:          f.TMAMin,
                TMAMinInclusive: f.TMAMinInclusive,
                TMAMax:          f.TMAMax,
                TMAMaxInclusive: f.TMAMaxInclusive,
                Priority:        f.Priority,
            }
        }

        if err := h.formulaRepo.CreateBatch(ctx, batchReq.NamaLokasi, formulas); err != nil {
            h.jsonError(w, err.Error(), http.StatusInternalServerError)
            return
        }

        h.calculator.InvalidateCache(batchReq.NamaLokasi)
        w.WriteHeader(http.StatusCreated)
        h.jsonResponse(w, map[string]interface{}{
            "status":      "created",
            "nama_lokasi": batchReq.NamaLokasi,
            "count":       len(formulas),
            "formulas":    formulas,
        })
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
    
    if batchReq.NamaLokasi == "" {
        h.jsonError(w, "nama_lokasi is required", http.StatusBadRequest)
        return
    }

    h.jsonError(w, "use formulas array for creating formulas, e.g., {\"nama_lokasi\": \"...\", \"formulas\": [{...}]}", http.StatusBadRequest)
}

// UpdateStationFormulas replaces all formulas for a station
func (h *APIHandler) UpdateStationFormulas(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := mux.Vars(r)["namaLokasi"]

    var req domain.FormulaCreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if len(req.Formulas) == 0 {
        h.jsonError(w, "formulas array is required", http.StatusBadRequest)
        return
    }

    formulas := make([]domain.FormulaParams, len(req.Formulas))
    for i, f := range req.Formulas {
        formulas[i] = domain.FormulaParams{
            NamaLokasi:      namaLokasi,
            StationName:     req.StationName,
            C:               f.C,
            H0:              f.H0,
            B:               f.B,
            TMAMin:          f.TMAMin,
            TMAMinInclusive: f.TMAMinInclusive,
            TMAMax:          f.TMAMax,
            TMAMaxInclusive: f.TMAMaxInclusive,
            Priority:        f.Priority,
        }
    }

    if err := h.formulaRepo.CreateBatch(ctx, namaLokasi, formulas); err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.calculator.InvalidateCache(namaLokasi)
    h.jsonResponse(w, map[string]interface{}{
        "status":      "updated",
        "nama_lokasi": namaLokasi,
        "count":       len(formulas),
    })
}

// UpdateFormulaByID updates a specific formula
func (h *APIHandler) UpdateFormulaByID(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    idStr := mux.Vars(r)["id"]
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        h.jsonError(w, "invalid formula id", http.StatusBadRequest)
        return
    }

    existing, err := h.formulaRepo.GetByID(ctx, id)
    if err != nil {
        h.jsonError(w, "formula not found", http.StatusNotFound)
        return
    }

    var params domain.FormulaParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    params.ID = id
    params.NamaLokasi = existing.NamaLokasi

    if err := h.formulaRepo.Update(ctx, &params); err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.calculator.InvalidateCache(params.NamaLokasi)
    h.jsonResponse(w, map[string]string{"status": "updated"})
}

// DeleteStationFormulas deletes all formulas for a station
func (h *APIHandler) DeleteStationFormulas(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := mux.Vars(r)["namaLokasi"]
    
    if err := h.formulaRepo.Delete(ctx, namaLokasi); err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    h.calculator.InvalidateCache(namaLokasi)
    w.WriteHeader(http.StatusNoContent)
}

// DeleteFormulaByID deletes a specific formula
func (h *APIHandler) DeleteFormulaByID(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    idStr := mux.Vars(r)["id"]
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        h.jsonError(w, "invalid formula id", http.StatusBadRequest)
        return
    }

    formula, err := h.formulaRepo.GetByID(ctx, id)
    if err != nil {
        h.jsonError(w, "formula not found", http.StatusNotFound)
        return
    }

    if err := h.formulaRepo.DeleteByID(ctx, id); err != nil {
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.calculator.InvalidateCache(formula.NamaLokasi)
    w.WriteHeader(http.StatusNoContent)
}

// ==================== Helpers ====================

func (h *APIHandler) jsonResponse(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}

func (h *APIHandler) jsonError(w http.ResponseWriter, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func truncateToHour(t time.Time) time.Time {
    return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}