package excel

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

type CellType int

const (
	CellText CellType = iota
	CellDate
	CellAmount
	CellInt
	CellCenter
	CellCenterWrap
	CellLeftWrap
)

type ColumnDef[T any] struct {
	Header   string
	Width    float64
	Type     CellType
	GetValue func(row T) any
}


type TableConfig[T any] struct {
	ReportType string 
	SheetName  string 
	Title      string 
	Columns    []ColumnDef[T]
	Rows       []T
}
type Styles struct {
	Header     int
	Data       int
	Amount     int
	Date       int
	Title      int
	Center     int
	CenterWrap int
	LeftWrap   int
}

const templatesDir = "templates/reports"

func GetProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func EnsureTemplateDir() {
	_ = os.MkdirAll(filepath.Join(GetProjectRoot(), templatesDir), os.ModePerm)
}

func GetTemplatePath(reportType string) string {
	return filepath.Join(GetProjectRoot(), templatesDir, fmt.Sprintf("%s_template.xlsx", reportType))
}
func CreateBaseFile(sheetName string) *excelize.File {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheetName)
	return f
}

func OpenFileOrNew(reportType, defaultSheetName string) (*excelize.File, string) {
	EnsureTemplateDir()
	tplPath := GetTemplatePath(reportType)

	sheet := defaultSheetName
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err := excelize.OpenFile(tplPath)
		if err == nil {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
			return f, sheet
		}
	}

	return CreateBaseFile(sheet), sheet
}

func InitStyles(f *excelize.File) Styles {
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Family: "Calibri"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Border:    border,
		Alignment: &excelize.Alignment{Vertical: "center"},
	})

	amountFmt := "#,##0.00"
	amountStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &amountFmt,
		Font:         &excelize.Font{Size: 10, Family: "Calibri"},
		Border:       border,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})

	dateStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    border,
	})

	centerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	centerWrapStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	leftWrapStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})

	return Styles{
		Header:     headerStyle,
		Data:       dataStyle,
		Amount:     amountStyle,
		Date:       dateStyle,
		Title:      titleStyle,
		Center:     centerStyle,
		CenterWrap: centerWrapStyle,
		LeftWrap:   leftWrapStyle,
	}
}

func GenerateTable[T any](cfg TableConfig[T]) ([]byte, error) {
	f, sheet := OpenFileOrNew(cfg.ReportType, cfg.SheetName)
	defer f.Close()

	styles := InitStyles(f)
	colCount := len(cfg.Columns)
	if colCount == 0 {
		return nil, fmt.Errorf("таблица должна содержать хотя бы одну колонку")
	}
	if strings.TrimSpace(cfg.Title) != "" {
		endColName, _ := excelize.CoordinatesToCellName(colCount, 1)
		_ = f.MergeCell(sheet, "A1", endColName)
		_ = f.SetCellValue(sheet, "A1", cfg.Title)
		_ = f.SetCellStyle(sheet, "A1", endColName, styles.Title)
	}
	headerRow := 2
	for colIdx, col := range cfg.Columns {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, headerRow)
		_ = f.SetCellValue(sheet, cell, col.Header)
		_ = f.SetCellStyle(sheet, cell, cell, styles.Header)
	}

	startDataRow := 3
	for rowIdx, row := range cfg.Rows {
		currentRow := startDataRow + rowIdx
		for colIdx, col := range cfg.Columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, currentRow)
			val := col.GetValue(row)
			_ = f.SetCellValue(sheet, cell, val)
			var styleID int
			switch col.Type {
			case CellAmount:
				styleID = styles.Amount
			case CellDate:
				styleID = styles.Date
			case CellCenter:
				styleID = styles.Center
			case CellCenterWrap:
				styleID = styles.CenterWrap
			case CellLeftWrap:
				styleID = styles.LeftWrap
			default:
				styleID = styles.Data
			}
			_ = f.SetCellStyle(sheet, cell, cell, styleID)
		}
	}

	for colIdx, col := range cfg.Columns {
		if col.Width > 0 {
			colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
			_ = f.SetColWidth(sheet, colLetter, colLetter, col.Width)
		}
	}
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
