package excel

import (
	"bytes"
	"fmt"
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
	SheetName string
	Title     string
	SubTitle  string
	Columns   []ColumnDef[T]
	Rows      []T
}

type Styles struct {
	Header     int
	Data       int
	Amount     int
	Date       int
	Title      int
	MainTitle  int
	SubTitle   int
	Center     int
	CenterWrap int
	LeftWrap   int
}

func CreateBaseFile(sheetName string) *excelize.File {
	if strings.TrimSpace(sheetName) == "" {
		sheetName = "Sheet1"
	}
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheetName)
	return f
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

	mainTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	subTitleStyle, _ := f.NewStyle(&excelize.Style{
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
		MainTitle:  mainTitleStyle,
		SubTitle:   subTitleStyle,
		Center:     centerStyle,
		CenterWrap: centerWrapStyle,
		LeftWrap:   leftWrapStyle,
	}
}

func GenerateTable[T any](cfg TableConfig[T]) ([]byte, error) {
	sheet := cfg.SheetName
	if strings.TrimSpace(sheet) == "" {
		sheet = "Sheet1"
	}
	f := CreateBaseFile(sheet)
	defer f.Close()

	styles := InitStyles(f)
	colCount := len(cfg.Columns)
	if colCount == 0 {
		return nil, fmt.Errorf("таблица должна содержать хотя бы одну колонку")
	}

	headerRow := 2
	startDataRow := 3

	if strings.TrimSpace(cfg.SubTitle) != "" {
		headerRow = 3
		startDataRow = 4

		if strings.TrimSpace(cfg.Title) != "" {
			_ = f.SetCellValue(sheet, "A1", cfg.Title)
			_ = f.SetCellStyle(sheet, "A1", "A1", styles.MainTitle)
			_ = f.SetRowHeight(sheet, 1, 26)
		}
		subEndCol, _ := excelize.CoordinatesToCellName(colCount, 2)
		_ = f.MergeCell(sheet, "A2", subEndCol)
		_ = f.SetCellValue(sheet, "A2", cfg.SubTitle)
		_ = f.SetCellStyle(sheet, "A2", subEndCol, styles.SubTitle)
		_ = f.SetRowHeight(sheet, 2, 20)
	} else if strings.TrimSpace(cfg.Title) != "" {
		endColName, _ := excelize.CoordinatesToCellName(colCount, 1)
		_ = f.MergeCell(sheet, "A1", endColName)
		_ = f.SetCellValue(sheet, "A1", cfg.Title)
		_ = f.SetCellStyle(sheet, "A1", endColName, styles.Title)
		_ = f.SetRowHeight(sheet, 1, 22)
	}

	_ = f.SetRowHeight(sheet, headerRow, 24)
	for colIdx, col := range cfg.Columns {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, headerRow)
		_ = f.SetCellValue(sheet, cell, col.Header)
		_ = f.SetCellStyle(sheet, cell, cell, styles.Header)
	}

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
			case CellInt, CellCenter:
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
