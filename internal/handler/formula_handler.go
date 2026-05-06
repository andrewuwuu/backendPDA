package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/httpx"
	"pda-monitor/internal/logger"
)

func (h *APIHandler) GetFormulas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	formulas, err := h.formulaRepo.GetAll(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, formulas)
}

func (h *APIHandler) GetFormulasGrouped(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	grouped, err := h.formulaRepo.GetAllGroupedByStation(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, grouped)
}

func (h *APIHandler) GetFormulasByStation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")
	formulas, err := h.formulaRepo.GetAllByNamaLokasi(ctx, namaLokasi)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(formulas) == 0 {
		httpx.Error(w, http.StatusNotFound, "no formulas found for station")
		return
	}

	httpx.JSON(w, http.StatusOK, domain.StationFormulas{
		NamaLokasi:  namaLokasi,
		StationName: formulas[0].StationName,
		Formulas:    formulas,
	})
}

func (h *APIHandler) GetFormulaByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid formula id")
		return
	}

	formula, err := h.formulaRepo.GetByID(ctx, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "formula not found")
		return
	}
	httpx.JSON(w, http.StatusOK, formula)
}

func (h *APIHandler) CreateFormulas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var batchReq domain.FormulaCreateRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&batchReq); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(batchReq.Formulas) > 0 {
		if batchReq.NamaLokasi == "" {
			httpx.Error(w, http.StatusBadRequest, "nama_lokasi is required")
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
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		recalculated := h.refreshDebitAfterFormulaChange(ctx, batchReq.NamaLokasi)
		httpx.JSON(w, http.StatusCreated, map[string]interface{}{
			"status":       "created",
			"nama_lokasi":  batchReq.NamaLokasi,
			"count":        len(formulas),
			"formulas":     formulas,
			"recalculated": recalculated,
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if batchReq.NamaLokasi == "" {
		httpx.Error(w, http.StatusBadRequest, "nama_lokasi is required")
		return
	}

	httpx.Error(w, http.StatusBadRequest, "use formulas array for creating formulas")
}

func (h *APIHandler) UpdateStationFormulas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")

	var req domain.FormulaCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Formulas) == 0 {
		httpx.Error(w, http.StatusBadRequest, "formulas array is required")
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
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	recalculated := h.refreshDebitAfterFormulaChange(ctx, namaLokasi)
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"status":       "updated",
		"nama_lokasi":  namaLokasi,
		"count":        len(formulas),
		"recalculated": recalculated,
	})
}

func (h *APIHandler) UpdateFormulaByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid formula id")
		return
	}

	existing, err := h.formulaRepo.GetByID(ctx, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "formula not found")
		return
	}

	var params domain.FormulaParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	params.ID = id
	params.NamaLokasi = existing.NamaLokasi

	if err := h.formulaRepo.Update(ctx, &params); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	recalculated := h.refreshDebitAfterFormulaChange(ctx, params.NamaLokasi)
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"status":       "updated",
		"recalculated": recalculated,
	})
}

func (h *APIHandler) DeleteStationFormulas(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namaLokasi := chi.URLParam(r, "namaLokasi")

	if err := h.formulaRepo.Delete(ctx, namaLokasi); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.refreshDebitAfterFormulaChange(ctx, namaLokasi)
	httpx.NoContent(w)
}

func (h *APIHandler) DeleteFormulaByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid formula id")
		return
	}

	formula, err := h.formulaRepo.GetByID(ctx, id)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "formula not found")
		return
	}

	if err := h.formulaRepo.DeleteByID(ctx, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.refreshDebitAfterFormulaChange(ctx, formula.NamaLokasi)
	httpx.NoContent(w)
}

func (h *APIHandler) refreshDebitAfterFormulaChange(ctx context.Context, namaLokasi string) bool {
	h.calculator.InvalidateCache(namaLokasi)
	if h.readingService == nil {
		return false
	}
	if err := h.readingService.RecalculateDebit(ctx, namaLokasi); err != nil {
		logger.Warn("Handler", "Failed to recalculate debit after formula change", logger.Fields(
			"nama_lokasi", namaLokasi,
			"error", err.Error(),
		))
		return false
	}
	return true
}
