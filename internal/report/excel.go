package report

import (
    "bytes"
    "fmt"
    "time"

    "github.com/xuri/excelize/v2"

    "pda-monitor/internal/domain"
)

type ExcelReportService struct{}

func NewExcelReportService() *ExcelReportService {
    return &ExcelReportService{}
}

// GenerateDebitReport creates Excel file with station names and debit values
func (s *ExcelReportService) GenerateDebitReport(results []domain.DebitResult, reportTime time.Time) (*bytes.Buffer, string, error) {
    f := excelize.NewFile()
    defer f.Close()

    sheetName := "Debit Report"
    f.SetSheetName("Sheet1", sheetName)

    // Set column widths
    f.SetColWidth(sheetName, "A", "A", 10)
    f.SetColWidth(sheetName, "B", "B", 40)
    f.SetColWidth(sheetName, "C", "C", 15)
    f.SetColWidth(sheetName, "D", "D", 15)

    // Header style
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

    // Title
    f.MergeCell(sheetName, "A1", "D1")
    f.SetCellValue(sheetName, "A1", fmt.Sprintf("LAPORAN DEBIT PDA - %s", reportTime.Format("02 January 2006, 15:04 WIB")))
    titleStyle, _ := f.NewStyle(&excelize.Style{
        Font:      &excelize.Font{Bold: true, Size: 14},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
    })
    f.SetCellStyle(sheetName, "A1", "D1", titleStyle)
    f.SetRowHeight(sheetName, 1, 30)

    // Headers (row 3)
    headers := []string{"No", "Nama Stasiun", "TMA (m)", "Debit (m³/s)"}
    for i, header := range headers {
        cell := fmt.Sprintf("%c3", 'A'+i)
        f.SetCellValue(sheetName, cell, header)
    }
    f.SetCellStyle(sheetName, "A3", "D3", headerStyle)
    f.SetRowHeight(sheetName, 3, 25)

    // Data rows (starting row 4)
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

    // Generate filename
    filename := fmt.Sprintf("Debit_Report_%s.xlsx", reportTime.Format("2006-01-02_15-04"))

    // Write to buffer
    buf := new(bytes.Buffer)
    if err := f.Write(buf); err != nil {
        return nil, "", fmt.Errorf("failed to write excel: %w", err)
    }

    return buf, filename, nil
}