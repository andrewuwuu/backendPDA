package handler

import (
	"archive/zip"
	"fmt"
	"net/http"

	"pda-monitor/internal/httpx"
	"pda-monitor/internal/timeutil"
)

func (h *APIHandler) ExportDailyReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := timeutil.NowJakarta()

	reports, err := h.reportService.BuildDailyReport(ctx, now)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	excelBuf, filename, err := h.excelService.GenerateDailyReport(reports, now)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpx.Attachment(w, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", filename)
	w.Write(excelBuf.Bytes())
}

func (h *APIHandler) ExportWeeklyReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := timeutil.NowJakarta()

	httpx.Attachment(w, "application/zip", fmt.Sprintf("weekly_reports_%s.zip", now.Format("2006-01-02")))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for i := 0; i < 7; i++ {
		reportDate := now.AddDate(0, 0, -i)

		reports, err := h.reportService.BuildDailyReport(ctx, reportDate)
		if err != nil {
			continue
		}

		excelBuf, filename, err := h.excelService.GenerateDailyReport(reports, reportDate)
		if err != nil {
			continue
		}

		zipFile, err := zipWriter.Create(filename)
		if err != nil {
			continue
		}
		zipFile.Write(excelBuf.Bytes())
	}
}
