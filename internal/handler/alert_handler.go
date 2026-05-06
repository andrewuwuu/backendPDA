package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/httpx"
	"pda-monitor/internal/middleware"
)

func (h *APIHandler) GetAllAlertLevels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alerts, err := h.alertLevelRepo.GetAll(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"count":        len(alerts),
		"alert_levels": alerts,
	})
}

func (h *APIHandler) GetAlertLevel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")

	alert, err := h.alertLevelRepo.GetByNamaLokasi(ctx, namaLokasi)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if alert == nil {
		httpx.JSON(w, http.StatusOK, map[string]interface{}{
			"nama_lokasi": namaLokasi,
			"alert_level": domain.AlertLevelNormal,
			"message":     "no custom alert level set, defaulting to normal",
		})
		return
	}

	httpx.JSON(w, http.StatusOK, alert)
}

func (h *APIHandler) GetAlertLevelsByLevel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	levelStr := chi.URLParam(r, "level")
	level := domain.AlertLevel(levelStr)

	if !level.IsValid() {
		httpx.Error(w, http.StatusBadRequest, "invalid alert level, must be one of: normal, siaga, waspada, awas")
		return
	}

	alerts, err := h.alertLevelRepo.GetByAlertLevel(ctx, level)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"level":        level,
		"count":        len(alerts),
		"alert_levels": alerts,
	})
}

func (h *APIHandler) UpdateAlertLevel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")

	claims := middleware.GetUserFromContext(ctx)
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.AlertLevelUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	station, err := h.stationRepo.GetByNamaLokasi(ctx, namaLokasi)
	if err != nil || station == nil {
		httpx.Error(w, http.StatusNotFound, "station not found")
		return
	}

	alert := &domain.StationAlertLevel{
		NamaLokasi:        namaLokasi,
		AlertLevel:        domain.AlertLevelNormal,
		UpperLimitNormal:  req.UpperLimitNormal,
		UpperLimitSiaga:   req.UpperLimitSiaga,
		UpperLimitWaspada: req.UpperLimitWaspada,
		UpperLimitAwas:    req.UpperLimitAwas,
		UpdatedBy:         claims.Username,
	}

	if err := h.alertLevelRepo.Upsert(ctx, alert); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"status":      "updated",
		"nama_lokasi": namaLokasi,
		"updated_by":  claims.Username,
	})
}

func (h *APIHandler) BulkUpdateAlertLevels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims := middleware.GetUserFromContext(ctx)
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req domain.BulkAlertLevelUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Updates) == 0 {
		httpx.Error(w, http.StatusBadRequest, "updates array is required")
		return
	}

	alerts := make([]domain.StationAlertLevel, 0, len(req.Updates))
	for _, u := range req.Updates {
		alerts = append(alerts, domain.StationAlertLevel{
			NamaLokasi:        u.NamaLokasi,
			AlertLevel:        domain.AlertLevelNormal,
			UpperLimitNormal:  u.UpperLimitNormal,
			UpperLimitSiaga:   u.UpperLimitSiaga,
			UpperLimitWaspada: u.UpperLimitWaspada,
			UpperLimitAwas:    u.UpperLimitAwas,
			UpdatedBy:         claims.Username,
		})
	}

	if err := h.alertLevelRepo.UpsertBatch(ctx, alerts); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"status":     "updated",
		"count":      len(alerts),
		"updated_by": claims.Username,
	})
}

func (h *APIHandler) DeleteAlertLevel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")

	if err := h.alertLevelRepo.Delete(ctx, namaLokasi); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.NoContent(w)
}
