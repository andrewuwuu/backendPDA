package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/httpx"
	"pda-monitor/internal/timeutil"
)

func (h *APIHandler) GetCurrentHourReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	readings, err := h.readingService.GetCurrentHourData(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"hour_bucket":   timeutil.TruncateToHour(time.Now()),
		"reading_count": len(readings),
		"readings":      readings,
	})
}

func (h *APIHandler) GetCurrentHourSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	summaries, err := h.readingService.GetCurrentHourSummary(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"hour_bucket":   timeutil.TruncateToHour(time.Now()),
		"station_count": len(summaries),
		"summaries":     summaries,
	})
}

func (h *APIHandler) GetLatestReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	readings, err := h.readingService.GetLatestReadings(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch all alert levels for computing alert status
	alertLevels, err := h.alertLevelRepo.GetAll(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build lookup map
	alertMap := make(map[string]*domain.StationAlertLevel)
	for i := range alertLevels {
		alertMap[alertLevels[i].NamaLokasi] = &alertLevels[i]
	}

	// Compute alert level for each reading
	results := make([]domain.ReadingWithAlertLevel, len(readings))
	for i, reading := range readings {
		results[i] = domain.ReadingWithAlertLevel{
			HourlyReading: reading,
			AlertLevel:    domain.DetermineAlertLevel(reading.TMA, alertMap[reading.NamaLokasi]),
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"hour_bucket":     timeutil.TruncateToHour(time.Now()),
		"station_count":   len(results),
		"latest_readings": results,
	})
}

func (h *APIHandler) GetStationReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")
	readings, err := h.readingService.GetCurrentHourByStation(ctx, namaLokasi)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"nama_lokasi":   namaLokasi,
		"hour_bucket":   timeutil.TruncateToHour(time.Now()),
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
		httpx.Error(w, http.StatusBadRequest, "from and to dates are required (format: 2006-01-02T15:04:05)")
		return
	}

	from, err := time.Parse("2006-01-02T15:04:05", fromStr)
	if err != nil {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid from date format")
			return
		}
	}

	to, err := time.Parse("2006-01-02T15:04:05", toStr)
	if err != nil {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid to date format")
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
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"from":          from,
		"to":            to,
		"nama_lokasi":   namaLokasi,
		"reading_count": len(readings),
		"readings":      readings,
	})
}

func (h *APIHandler) GetRealtimeWithDebit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	records, err := h.telemetryService.FetchRealtime(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	results, err := h.calculator.CalculateBatch(ctx, records)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, results)
}

func (h *APIHandler) GetHistoricalWithDebit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := r.URL.Query().Get("nama_lokasi")
	if namaLokasi == "" {
		httpx.Error(w, http.StatusBadRequest, "nama_lokasi is required")
		return
	}

	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid from date")
		return
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid to date")
		return
	}

	records, err := h.telemetryService.FetchHistorical(ctx, namaLokasi, from, to)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	results, err := h.calculator.CalculateBatch(ctx, records)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, results)
}
