package service

import (
	"testing"

	"pda-monitor/internal/domain"
)

func TestAssembleDailyReports_Combined(t *testing.T) {
	stationMap := map[string]string{
		"station_a": "PDA A Alat",
		"station_b": "PDA B Alat",
	}

	d07 := 1.5
	d12 := 2.0
	d17 := 3.0
	minTMA := 0.5
	maxTMA := 1.2

	tmaSummary := map[string]domain.TMARangeSummary{
		"station_a": {NamaLokasi: "station_a", MinTMA: &minTMA, MaxTMA: &maxTMA},
	}

	debitSnapshots := map[string]map[int]*float64{
		"station_a": {7: &d07, 12: &d12, 17: &d17},
	}

	reports := assembleDailyReports(stationMap, tmaSummary, debitSnapshots)

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	r := reports[0]
	if r.NamaLokasi != "station_a" {
		t.Errorf("expected NamaLokasi='station_a', got %q", r.NamaLokasi)
	}
	if r.Debit07 == nil || *r.Debit07 != 1.5 {
		t.Errorf("expected Debit07=1.5, got %v", r.Debit07)
	}
	if r.MinTMA == nil || *r.MinTMA != 0.5 {
		t.Errorf("expected MinTMA=0.5, got %v", r.MinTMA)
	}
}

func TestAssembleDailyReports_TMAOnly(t *testing.T) {
	stationMap := map[string]string{"station_a": "PDA A"}

	minTMA := 0.3
	maxTMA := 0.9
	tmaSummary := map[string]domain.TMARangeSummary{
		"station_a": {NamaLokasi: "station_a", MinTMA: &minTMA, MaxTMA: &maxTMA},
	}

	reports := assembleDailyReports(stationMap, tmaSummary, nil)

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].Debit07 != nil {
		t.Error("expected nil Debit07 when no debit snapshots")
	}
	if reports[0].MinTMA == nil || *reports[0].MinTMA != 0.3 {
		t.Errorf("expected MinTMA=0.3, got %v", reports[0].MinTMA)
	}
}

func TestAssembleDailyReports_DebitOnly(t *testing.T) {
	stationMap := map[string]string{"station_a": "PDA A"}
	d07 := 2.5
	debitSnapshots := map[string]map[int]*float64{
		"station_a": {7: &d07, 12: nil, 17: nil},
	}

	reports := assembleDailyReports(stationMap, nil, debitSnapshots)

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].MinTMA != nil {
		t.Error("expected nil MinTMA when no TMA summary")
	}
	if reports[0].Debit07 == nil || *reports[0].Debit07 != 2.5 {
		t.Errorf("expected Debit07=2.5, got %v", reports[0].Debit07)
	}
}

func TestAssembleDailyReports_MissingStationName(t *testing.T) {
	// station not in stationMap — should fall back to namaLokasi
	stationMap := map[string]string{}
	d07 := 1.0
	debitSnapshots := map[string]map[int]*float64{
		"unknown_station": {7: &d07},
	}

	reports := assembleDailyReports(stationMap, nil, debitSnapshots)

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	// When stationMap doesn't have the station, namaAlat defaults to namaLokasi,
	// and CleanStationName with equal nama_lokasi/namaAlat returns namaLokasi.
	if reports[0].NamaAlat != "unknown_station" {
		t.Errorf("expected NamaAlat fallback to 'unknown_station', got %q", reports[0].NamaAlat)
	}
}

func TestAssembleDailyReports_Empty(t *testing.T) {
	reports := assembleDailyReports(nil, nil, nil)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports for empty input, got %d", len(reports))
	}
}

func TestAssembleDailyReports_MergesStationsFromBothSources(t *testing.T) {
	stationMap := map[string]string{
		"station_a": "PDA A",
		"station_b": "PDA B",
	}
	minTMA := 0.1
	tmaSummary := map[string]domain.TMARangeSummary{
		"station_a": {NamaLokasi: "station_a", MinTMA: &minTMA},
	}
	d07 := 5.0
	debitSnapshots := map[string]map[int]*float64{
		"station_b": {7: &d07},
	}

	reports := assembleDailyReports(stationMap, tmaSummary, debitSnapshots)

	if len(reports) != 2 {
		t.Fatalf("expected 2 reports (one from TMA, one from debit), got %d", len(reports))
	}

	seen := map[string]bool{}
	for _, r := range reports {
		seen[r.NamaLokasi] = true
	}
	if !seen["station_a"] || !seen["station_b"] {
		t.Errorf("expected both station_a and station_b, got %v", seen)
	}
}
