package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"pda-monitor/internal/httpx"
)

func (h *APIHandler) GetStations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stations, err := h.stationRepo.GetAll(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, stations)
}

func (h *APIHandler) GetStation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")
	station, err := h.stationRepo.GetByNamaLokasi(ctx, namaLokasi)
	if err != nil || station == nil {
		httpx.Error(w, http.StatusNotFound, "station not found")
		return
	}
	httpx.JSON(w, http.StatusOK, station)
}

func (h *APIHandler) SyncStations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, err := h.telemetryService.SyncStations(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"status": "synced", "count": count})
}
