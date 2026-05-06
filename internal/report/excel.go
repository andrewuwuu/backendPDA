package report

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/timeutil"
)

type ExcelReportService struct{}

func NewExcelReportService() *ExcelReportService {
	return &ExcelReportService{}
}

func (s *ExcelReportService) GenerateDebitReport(results []domain.DebitResult, reportTime time.Time) (*bytes.Buffer, string, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {

		}
	}()

	sheetName := "Laporan Debit"
	f.SetSheetName("Sheet1", sheetName)

	f.SetColWidth(sheetName, "A", "A", 10)
	f.SetColWidth(sheetName, "B", "B", 40)
	f.SetColWidth(sheetName, "C", "C", 15)
	f.SetColWidth(sheetName, "D", "D", 15)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	reportTimeJkt := reportTime.In(timeutil.JakartaLocation())

	f.MergeCell(sheetName, "A1", "D1")
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("LAPORAN DEBIT PDA - %s", reportTimeJkt.Format("02 January 2006, 15:04 WIB")))
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "D1", titleStyle)
	f.SetRowHeight(sheetName, 1, 30)

	headers := []string{"No", "Nama Stasiun", "TMA (m)", "Debit (m³/s)"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}
	f.SetCellStyle(sheetName, "A3", "D3", headerStyle)
	f.SetRowHeight(sheetName, 3, 25)

	row := 4
	for i, r := range results {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), r.NamaAlat)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("%.4f", r.TMA))

		if r.IsValid {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("%.3f", r.Debit))
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), "-")
		}

		row++
	}

	filename := fmt.Sprintf("Laporan_Debit_%s.xlsx", reportTimeJkt.Format("2006-01-02_15-04"))

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, "", fmt.Errorf("failed to write excel: %w", err)
	}

	return buf, filename, nil
}

func (s *ExcelReportService) GenerateDailyReport(reports []domain.DailyStationReport, reportDate time.Time) (*bytes.Buffer, string, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {

		}
	}()

	sheetName := "Laporan Harian"
	f.SetSheetName("Sheet1", sheetName)

	f.SetColWidth(sheetName, "A", "A", 8)  // No PDA
	f.SetColWidth(sheetName, "B", "B", 35) // Nama PDA
	f.SetColWidth(sheetName, "C", "C", 15) // Debit 07:00
	f.SetColWidth(sheetName, "D", "D", 15) // Debit 12:00
	f.SetColWidth(sheetName, "E", "E", 15) // Debit 17:00
	f.SetColWidth(sheetName, "F", "F", 15) // TMA Terendah
	f.SetColWidth(sheetName, "G", "G", 15) // TMA Tertinggi

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	nameStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	reportDateJkt := reportDate.In(timeutil.JakartaLocation())

	f.MergeCell(sheetName, "A1", "G1")
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("LAPORAN HARIAN DEBIT PDA - %s", reportDateJkt.Format("02 January 2006")))
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.SetCellStyle(sheetName, "A1", "G1", titleStyle)
	f.SetRowHeight(sheetName, 1, 30)

	headers := []string{
		"No PDA",
		"Nama PDA",
		"Debit (07:00)",
		"Debit (12:00)",
		"Debit (17:00)",
		"TMA Terendah",
		"TMA Tertinggi",
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}
	f.SetCellStyle(sheetName, "A3", "G3", headerStyle)
	f.SetRowHeight(sheetName, 3, 35)

	row := 4
	for i, r := range reports {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)

		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), r.NamaAlat)
		f.SetCellStyle(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), nameStyle)

		if r.Debit07 != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("%.3f", *r.Debit07))
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), "-")
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataStyle)

		if r.Debit12 != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("%.3f", *r.Debit12))
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), "-")
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataStyle)

		if r.Debit17 != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("%.3f", *r.Debit17))
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), "-")
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)

		if r.MinTMA != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), fmt.Sprintf("%.4f", *r.MinTMA))
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), "-")
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataStyle)

		if r.MaxTMA != nil {
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("%.4f", *r.MaxTMA))
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), "-")
		}
		f.SetCellStyle(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), dataStyle)

		row++
	}

	footerRow := row + 1
	f.MergeCell(sheetName, fmt.Sprintf("A%d", footerRow), fmt.Sprintf("G%d", footerRow))
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", footerRow), "Keterangan: Debit dalam m³/s, TMA dalam meter")
	footerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", footerRow), fmt.Sprintf("G%d", footerRow), footerStyle)

	filename := fmt.Sprintf("Laporan_Harian_PDA_%s.xlsx", reportDateJkt.Format("2006-01-02"))

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, "", fmt.Errorf("failed to write excel: %w", err)
	}

	return buf, filename, nil
}
