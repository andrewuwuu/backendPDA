package handler

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/gorilla/mux"

    "pda-monitor/internal/auth"
    "pda-monitor/internal/domain"
    "pda-monitor/internal/logger"
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

    // Readings endpoints
    protected.HandleFunc("/readings/current", h.GetCurrentHourReadings).Methods("GET")
    protected.HandleFunc("/readings/current/summary", h.GetCurrentHourSummary).Methods("GET")
    protected.HandleFunc("/readings/current/latest", h.GetLatestReadings).Methods("GET")
    protected.HandleFunc("/readings/station/{namaLokasi}", h.GetStationReadings).Methods("GET")

    // Stations
    protected.HandleFunc("/stations", h.GetStations).Methods("GET")
    protected.HandleFunc("/stations/{namaLokasi}", h.GetStation).Methods("GET")
    protected.HandleFunc("/stations/sync", h.SyncStations).Methods("POST")

    // Formulas
    protected.HandleFunc("/formulas", h.GetFormulas).Methods("GET")
    protected.HandleFunc("/formulas/{namaLokasi}", h.GetFormula).Methods("GET")

    // Admin routes
    adminRoutes := protected.PathPrefix("").Subrouter()
    adminRoutes.Use(authMiddleware.RequireRole("admin"))
    adminRoutes.HandleFunc("/formulas", h.CreateFormula).Methods("POST")
    adminRoutes.HandleFunc("/formulas/{namaLokasi}", h.UpdateFormula).Methods("PUT")
    adminRoutes.HandleFunc("/formulas/{namaLokasi}", h.DeleteFormula).Methods("DELETE")

    // Report endpoints
    protected.HandleFunc("/reports/export", h.ExportReport).Methods("GET")

    // Admin user management
    adminRoutes.HandleFunc("/users", h.CreateUser).Methods("POST")
    adminRoutes.HandleFunc("/users/{id}/password", h.UpdateUserPassword).Methods("PUT")

    // Debug endpoint (admin only)
    adminRoutes.HandleFunc("/debug/jwt", h.GetJWTInfo).Methods("GET")
}

// ==================== Auth ====================

func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req domain.LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        logger.Warn("Auth", "Invalid login request body", logger.Fields(
            "remote_addr", r.RemoteAddr,
            "error", err.Error(),
        ))
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    user, err := h.userRepo.GetByUsername(ctx, req.Username)
    if err != nil {
        logger.Error("Auth", "Database error during login", logger.Fields(
            "username", req.Username,
            "remote_addr", r.RemoteAddr,
            "error", err.Error(),
        ))
        h.jsonError(w, "internal error", http.StatusInternalServerError)
        return
    }

    if user == nil {
        logger.Warn("Auth", "Login attempt with unknown username", logger.Fields(
            "username", req.Username,
            "remote_addr", r.RemoteAddr,
        ))
        h.jsonError(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    if !h.userRepo.ValidatePassword(user, req.Password) {
        logger.Warn("Auth", "Login failed - invalid password", logger.Fields(
            "user_id", user.ID,
            "username", req.Username,
            "remote_addr", r.RemoteAddr,
        ))
        h.jsonError(w, "invalid credentials", http.StatusUnauthorized)
        return
    }

    token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
    if err != nil {
        logger.Error("Auth", "Failed to generate JWT token", logger.Fields(
            "user_id", user.ID,
            "username", user.Username,
            "error", err.Error(),
        ))
        h.jsonError(w, "failed to generate token", http.StatusInternalServerError)
        return
    }

    logger.Info("Auth", "User logged in successfully", logger.Fields(
        "user_id", user.ID,
        "username", user.Username,
        "role", user.Role,
        "remote_addr", r.RemoteAddr,
    ))

    h.jsonResponse(w, domain.LoginResponse{
        Token: token,
        User:  *user,
    })
}

// ==================== User Management ====================

type CreateUserRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
    Role     string `json:"role"`
}

func (h *APIHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    claims := middleware.GetUserFromContext(ctx)

    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if req.Username == "" || req.Password == "" {
        h.jsonError(w, "username and password are required", http.StatusBadRequest)
        return
    }

    if len(req.Password) < 8 {
        h.jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
        return
    }

    role := req.Role
    if role == "" {
        role = "user"
    }

    if role != "admin" && role != "user" {
        h.jsonError(w, "role must be 'admin' or 'user'", http.StatusBadRequest)
        return
    }

    user := &domain.User{
        Username: req.Username,
        Role:     role,
    }

    if err := h.userRepo.Create(ctx, user, req.Password); err != nil {
        logger.Error("Handler", "Failed to create user", logger.Fields(
            "username", req.Username,
            "created_by", claims.Username,
            "error", err.Error(),
        ))
        h.jsonError(w, "failed to create user", http.StatusInternalServerError)
        return
    }

    logger.Info("Handler", "User created by admin", logger.Fields(
        "new_user_id", user.ID,
        "new_username", user.Username,
        "new_role", user.Role,
        "created_by", claims.Username,
    ))

    w.WriteHeader(http.StatusCreated)
    h.jsonResponse(w, user)
}

type UpdatePasswordRequest struct {
    NewPassword string `json:"new_password"`
}

func (h *APIHandler) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    claims := middleware.GetUserFromContext(ctx)

    vars := mux.Vars(r)
    userID := vars["id"]

    var id int64
    if _, err := json.Number(userID).Int64(); err != nil {
        h.jsonError(w, "invalid user id", http.StatusBadRequest)
        return
    }
    id, _ = json.Number(userID).Int64()

    var req UpdatePasswordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if len(req.NewPassword) < 8 {
        h.jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
        return
    }

    if err := h.userRepo.UpdatePassword(ctx, id, req.NewPassword); err != nil {
        logger.Error("Handler", "Failed to update password", logger.Fields(
            "target_user_id", id,
            "updated_by", claims.Username,
            "error", err.Error(),
        ))
        h.jsonError(w, "failed to update password", http.StatusInternalServerError)
        return
    }

    logger.Info("Handler", "Password updated by admin", logger.Fields(
        "target_user_id", id,
        "updated_by", claims.Username,
    ))

    h.jsonResponse(w, map[string]string{"status": "password updated"})
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
        logger.Error("Handler", "Failed to fetch realtime for export", logger.F("error", err.Error()))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    results, err := h.calculator.CalculateBatch(ctx, records)
    if err != nil {
        logger.Error("Handler", "Failed to calculate debit for export", logger.F("error", err.Error()))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    now := time.Now()
    buf, filename, err := h.excelService.GenerateDebitReport(results, now)
    if err != nil {
        logger.Error("Handler", "Failed to generate Excel", logger.F("error", err.Error()))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    logger.Info("Handler", "Excel report exported", logger.Fields(
        "filename", filename,
        "stations", len(results),
    ))

    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", "attachment; filename="+filename)
    w.Write(buf.Bytes())
}

// ==================== Readings ====================

func (h *APIHandler) GetCurrentHourReadings(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    readings, err := h.readingService.GetCurrentHourData(ctx)
    if err != nil {
        logger.Error("Handler", "Failed to get current hour readings", logger.F("error", err.Error()))
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
        logger.Error("Handler", "Failed to get current hour summary", logger.F("error", err.Error()))
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
        logger.Error("Handler", "Failed to get latest readings", logger.F("error", err.Error()))
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
        logger.Error("Handler", "Failed to get station readings", logger.Fields(
            "nama_lokasi", namaLokasi,
            "error", err.Error(),
        ))
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

// ==================== PDA ====================

func (h *APIHandler) GetRealtimeWithDebit(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    records, err := h.telemetryService.FetchRealtime(ctx)
    if err != nil {
        logger.Error("Handler", "Failed to fetch realtime", logger.F("error", err.Error()))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    results, err := h.calculator.CalculateBatch(ctx, records)
    if err != nil {
        logger.Error("Handler", "Failed to calculate batch", logger.F("error", err.Error()))
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
        logger.Error("Handler", "Failed to fetch historical", logger.Fields(
            "nama_lokasi", namaLokasi,
            "from", from,
            "to", to,
            "error", err.Error(),
        ))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    results, err := h.calculator.CalculateBatch(ctx, records)
    if err != nil {
        logger.Error("Handler", "Failed to calculate historical batch", logger.F("error", err.Error()))
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
        logger.Error("Handler", "Failed to get stations", logger.F("error", err.Error()))
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
        logger.Error("Handler", "Failed to sync stations", logger.F("error", err.Error()))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    logger.Info("Handler", "Stations synced via API", logger.F("count", count))
    h.jsonResponse(w, map[string]interface{}{"status": "synced", "count": count})
}

// ==================== Formulas ====================

func (h *APIHandler) GetFormulas(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    formulas, err := h.formulaRepo.GetAll(ctx)
    if err != nil {
        logger.Error("Handler", "Failed to get formulas", logger.F("error", err.Error()))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    h.jsonResponse(w, formulas)
}

func (h *APIHandler) GetFormula(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    namaLokasi := mux.Vars(r)["namaLokasi"]
    formula, err := h.formulaRepo.GetByNamaLokasi(ctx, namaLokasi)
    if err != nil {
        h.jsonError(w, "formula not found", http.StatusNotFound)
        return
    }
    h.jsonResponse(w, formula)
}

func (h *APIHandler) CreateFormula(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    claims := middleware.GetUserFromContext(ctx)

    var params domain.FormulaParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    if err := h.formulaRepo.Create(ctx, &params); err != nil {
        logger.Error("Handler", "Failed to create formula", logger.Fields(
            "nama_lokasi", params.NamaLokasi,
            "created_by", claims.Username,
            "error", err.Error(),
        ))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.calculator.RefreshCache(ctx)

    logger.Info("Handler", "Formula created", logger.Fields(
        "nama_lokasi", params.NamaLokasi,
        "created_by", claims.Username,
    ))

    w.WriteHeader(http.StatusCreated)
    h.jsonResponse(w, params)
}

func (h *APIHandler) UpdateFormula(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    claims := middleware.GetUserFromContext(ctx)
    namaLokasi := mux.Vars(r)["namaLokasi"]

    var params domain.FormulaParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        h.jsonError(w, "invalid request body", http.StatusBadRequest)
        return
    }

    params.NamaLokasi = namaLokasi
    if err := h.formulaRepo.Update(ctx, &params); err != nil {
        logger.Error("Handler", "Failed to update formula", logger.Fields(
            "nama_lokasi", namaLokasi,
            "updated_by", claims.Username,
            "error", err.Error(),
        ))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.calculator.InvalidateCache(namaLokasi)

    logger.Info("Handler", "Formula updated", logger.Fields(
        "nama_lokasi", namaLokasi,
        "updated_by", claims.Username,
    ))

    h.jsonResponse(w, map[string]string{"status": "updated"})
}

func (h *APIHandler) DeleteFormula(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    claims := middleware.GetUserFromContext(ctx)
    namaLokasi := mux.Vars(r)["namaLokasi"]

    if err := h.formulaRepo.Delete(ctx, namaLokasi); err != nil {
        logger.Error("Handler", "Failed to delete formula", logger.Fields(
            "nama_lokasi", namaLokasi,
            "deleted_by", claims.Username,
            "error", err.Error(),
        ))
        h.jsonError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.calculator.InvalidateCache(namaLokasi)

    logger.Info("Handler", "Formula deleted", logger.Fields(
        "nama_lokasi", namaLokasi,
        "deleted_by", claims.Username,
    ))

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