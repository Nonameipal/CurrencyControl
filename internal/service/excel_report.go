package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"CurrencyControl/internal/delivery/dto"

	"github.com/xuri/excelize/v2"
)

const templatesDir = "templates/reports"

func getTemplatePath(reportType string) string {
	return filepath.Join(getProjectRoot(), templatesDir, fmt.Sprintf("%s_template.xlsx", reportType))
}

func ensureTemplateDir() {
	_ = os.MkdirAll(filepath.Join(getProjectRoot(), templatesDir), os.ModePerm)
}

func getProjectRoot() string {
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

func createBaseFile(sheetName string) *excelize.File {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheetName)
	return f
}

func getStyles(f *excelize.File) (headerStyle, dataStyle, amountStyle, dateStyle, titleStyle int) {
	headerStyle, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Family: "Calibri"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})

	dataStyle, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Family: "Calibri"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})

	amountFmt := "#,##0.00"
	amountStyle, _ = f.NewStyle(&excelize.Style{
		CustomNumFmt: &amountFmt,
		Font:         &excelize.Font{Size: 10, Family: "Calibri"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})

	dateStyle, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Family: "Calibri"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	titleStyle, _ = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	return
}

func GenerateContractsExcel(rows []dto.ContractExcelRow, clientName string) ([]byte, error) {
	ensureTemplateDir()
	tplPath := getTemplatePath(dto.ReportTypeContracts)

	var f *excelize.File
	var err error

	sheet := "Контракты"
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err = excelize.OpenFile(tplPath)
		if err != nil {
			f = createBaseFile(sheet)
		} else {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
		}
	} else {
		f = createBaseFile(sheet)
	}
	defer f.Close()

	headerStyle, dataStyle, amountStyle, dateStyle, titleStyle := getStyles(f)

	titleText := fmt.Sprintf("Контракты \"%s\"", clientName)
	_ = f.MergeCell(sheet, "A1", "K1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "K1", titleStyle)

	headers := []string{
		"Номер", "Дата", "Предмет", "Сумма", "Валюта",
		"Срок возврата", "Срок поставки", "Дата окончании контракта",
		"Наименование получателя", "Счет получателя", "Страна получателя",
	}

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	startRow := 3
	for i, r := range rows {
		rowIdx := startRow + i
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.Number)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Date)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.Subject)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.Amount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.Currency)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), r.ReturnDate)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), r.DeliveryDate)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), r.ContractEndDate)
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), r.ReceiverName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), r.ReceiverAccount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", rowIdx), r.ReceiverCountry)

		for col := 1; col <= 11; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			switch col {
			case 4:
				_ = f.SetCellStyle(sheet, cell, cell, amountStyle)
			case 2, 6, 7, 8:
				_ = f.SetCellStyle(sheet, cell, cell, dateStyle)
			default:
				_ = f.SetCellStyle(sheet, cell, cell, dataStyle)
			}
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 16)
	_ = f.SetColWidth(sheet, "B", "B", 13)
	_ = f.SetColWidth(sheet, "C", "C", 25)
	_ = f.SetColWidth(sheet, "D", "D", 16)
	_ = f.SetColWidth(sheet, "E", "E", 10)
	_ = f.SetColWidth(sheet, "F", "F", 14)
	_ = f.SetColWidth(sheet, "G", "G", 14)
	_ = f.SetColWidth(sheet, "H", "H", 24)
	_ = f.SetColWidth(sheet, "I", "I", 24)
	_ = f.SetColWidth(sheet, "J", "J", 20)
	_ = f.SetColWidth(sheet, "K", "K", 18)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateInvoicesExcel(rows []dto.InvoiceExcelRow, clientName string) ([]byte, error) {
	ensureTemplateDir()
	tplPath := getTemplatePath(dto.ReportTypeInvoices)

	var f *excelize.File
	var err error

	sheet := "Инвойсы"
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err = excelize.OpenFile(tplPath)
		if err != nil {
			f = createBaseFile(sheet)
		} else {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
		}
	} else {
		f = createBaseFile(sheet)
	}
	defer f.Close()

	headerStyle, dataStyle, amountStyle, dateStyle, titleStyle := getStyles(f)

	titleText := fmt.Sprintf("Инвойсы \"%s\"", clientName)
	_ = f.MergeCell(sheet, "A1", "F1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "F1", titleStyle)

	headers := []string{
		"Номер", "Дата", "Сумма", "Валюта", "HS CODE", "Назначение оплаты товар/услуга",
	}

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	startRow := 3
	for i, r := range rows {
		rowIdx := startRow + i
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.Number)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Date)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.Amount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.Currency)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.HSCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), r.PaymentPurpose)

		for col := 1; col <= 6; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			switch col {
			case 3:
				_ = f.SetCellStyle(sheet, cell, cell, amountStyle)
			case 2:
				_ = f.SetCellStyle(sheet, cell, cell, dateStyle)
			default:
				_ = f.SetCellStyle(sheet, cell, cell, dataStyle)
			}
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 18)
	_ = f.SetColWidth(sheet, "B", "B", 14)
	_ = f.SetColWidth(sheet, "C", "C", 16)
	_ = f.SetColWidth(sheet, "D", "D", 10)
	_ = f.SetColWidth(sheet, "E", "E", 14)
	_ = f.SetColWidth(sheet, "F", "F", 35)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateGTDExcel(rows []dto.GTDExcelRow, clientName string) ([]byte, error) {
	ensureTemplateDir()
	tplPath := getTemplatePath(dto.ReportTypeGTD)

	var f *excelize.File
	var err error

	sheet := "ГТД"
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err = excelize.OpenFile(tplPath)
		if err != nil {
			f = createBaseFile(sheet)
		} else {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
		}
	} else {
		f = createBaseFile(sheet)
	}
	defer f.Close()

	headerStyle, dataStyle, amountStyle, dateStyle, titleStyle := getStyles(f)

	titleText := fmt.Sprintf("ГТД \"%s\"", clientName)
	_ = f.MergeCell(sheet, "A1", "G1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "G1", titleStyle)

	headers := []string{
		"Номер ГТД", "Дата", "Сумма", "Валюта", "HS CODE", "Наименование отправителя", "Страна отправителя",
	}

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	startRow := 3
	for i, r := range rows {
		rowIdx := startRow + i
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.Number)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Date)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.Amount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.Currency)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.HSCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), r.SenderName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), r.Country)

		for col := 1; col <= 7; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			switch col {
			case 3:
				_ = f.SetCellStyle(sheet, cell, cell, amountStyle)
			case 2:
				_ = f.SetCellStyle(sheet, cell, cell, dateStyle)
			default:
				_ = f.SetCellStyle(sheet, cell, cell, dataStyle)
			}
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 20)
	_ = f.SetColWidth(sheet, "B", "B", 14)
	_ = f.SetColWidth(sheet, "C", "C", 16)
	_ = f.SetColWidth(sheet, "D", "D", 10)
	_ = f.SetColWidth(sheet, "E", "E", 14)
	_ = f.SetColWidth(sheet, "F", "F", 28)
	_ = f.SetColWidth(sheet, "G", "G", 20)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateAAExcel(rows []dto.AAExcelRow, clientName string) ([]byte, error) {
	ensureTemplateDir()
	tplPath := getTemplatePath(dto.ReportTypeAdditionalAgreements)

	var f *excelize.File
	var err error

	sheet := "Доп. соглашения"
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err = excelize.OpenFile(tplPath)
		if err != nil {
			f = createBaseFile(sheet)
		} else {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
		}
	} else {
		f = createBaseFile(sheet)
	}
	defer f.Close()

	headerStyle, dataStyle, amountStyle, dateStyle, titleStyle := getStyles(f)

	titleText := fmt.Sprintf("Дополнительные соглашения \"%s\"", clientName)
	_ = f.MergeCell(sheet, "A1", "J1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "J1", titleStyle)

	headers := []string{
		"Номер", "Тип документа", "Дата", "Номер контракта", "Сумма",
		"Валюта", "Срок поставки", "Срок возврата", "Продление до", "Предмет",
	}

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	startRow := 3
	for i, r := range rows {
		rowIdx := startRow + i
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.Number)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.DocType)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.Date)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.ContractNum)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.Amount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), r.Currency)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), r.DeliveryDate)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), r.ReturnDate)
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), r.ExtendDateTo)
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), r.Subject)

		for col := 1; col <= 10; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			switch col {
			case 5:
				_ = f.SetCellStyle(sheet, cell, cell, amountStyle)
			case 3, 7, 8, 9:
				_ = f.SetCellStyle(sheet, cell, cell, dateStyle)
			default:
				_ = f.SetCellStyle(sheet, cell, cell, dataStyle)
			}
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 16)
	_ = f.SetColWidth(sheet, "B", "B", 18)
	_ = f.SetColWidth(sheet, "C", "C", 14)
	_ = f.SetColWidth(sheet, "D", "D", 18)
	_ = f.SetColWidth(sheet, "E", "E", 16)
	_ = f.SetColWidth(sheet, "F", "F", 10)
	_ = f.SetColWidth(sheet, "G", "G", 14)
	_ = f.SetColWidth(sheet, "H", "H", 14)
	_ = f.SetColWidth(sheet, "I", "I", 14)
	_ = f.SetColWidth(sheet, "J", "J", 28)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateClientsExcel(rows []dto.ClientExcelRow) ([]byte, error) {
	ensureTemplateDir()
	tplPath := getTemplatePath(dto.ReportTypeClients)

	var f *excelize.File
	var err error

	sheet := "Клиенты"
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err = excelize.OpenFile(tplPath)
		if err != nil {
			f = createBaseFile(sheet)
		} else {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
		}
	} else {
		f = createBaseFile(sheet)
	}
	defer f.Close()

	headerStyle, dataStyle, amountStyle, dateStyle, titleStyle := getStyles(f)

	titleText := "Отчет по клиентам банка"
	_ = f.MergeCell(sheet, "A1", "G1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "G1", titleStyle)

	headers := []string{
		"ID", "Наименование компании", "ИНН", "Филиал", "Кол-во контрактов", "Общая сумма", "Дата создания",
	}

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	startRow := 3
	for i, r := range rows {
		rowIdx := startRow + i
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), r.ID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Name)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.INN)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.BranchName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.ContractsCount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), r.TotalAmount)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), r.CreatedAt)

		for col := 1; col <= 7; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowIdx)
			switch col {
			case 6:
				_ = f.SetCellStyle(sheet, cell, cell, amountStyle)
			case 7:
				_ = f.SetCellStyle(sheet, cell, cell, dateStyle)
			default:
				_ = f.SetCellStyle(sheet, cell, cell, dataStyle)
			}
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 10)
	_ = f.SetColWidth(sheet, "B", "B", 30)
	_ = f.SetColWidth(sheet, "C", "C", 16)
	_ = f.SetColWidth(sheet, "D", "D", 25)
	_ = f.SetColWidth(sheet, "E", "E", 18)
	_ = f.SetColWidth(sheet, "F", "F", 20)
	_ = f.SetColWidth(sheet, "G", "G", 15)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatDocNumber(num string) string {
	num = strings.TrimSpace(num)
	if num == "" {
		return ""
	}
	if strings.HasPrefix(num, "№") {
		return num
	}
	return "№" + num
}

func GenerateClientConsolidatedExcel(data *dto.ClientConsolidatedReportData) ([]byte, error) {
	ensureTemplateDir()
	tplPath := getTemplatePath(dto.ReportTypeClientConsolidated)

	var f *excelize.File
	var err error

	sheet := "Умуми"
	if _, statErr := os.Stat(tplPath); statErr == nil {
		f, err = excelize.OpenFile(tplPath)
		if err != nil {
			f = createBaseFile(sheet)
		} else {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheet = sheetList[0]
			}
		}
	} else {
		f = createBaseFile(sheet)
	}
	defer f.Close()

	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
	}

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    border,
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 9, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})

	centerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    border,
	})

	centerWrapStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    border,
	})

	leftWrapStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Family: "Calibri"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border:    border,
	})

	amountFmt := "#,##0.00"
	amountStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &amountFmt,
		Font:         &excelize.Font{Size: 10, Family: "Calibri"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       border,
	})

	if data == nil {
		data = &dto.ClientConsolidatedReportData{ClientName: ""}
	}

	titleText := "Клиент"
	if strings.TrimSpace(data.ClientName) != "" {
		titleText = fmt.Sprintf("Клиент: %s", data.ClientName)
	}
	_ = f.MergeCell(sheet, "A1", "O1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "O1", titleStyle)
	_ = f.SetRowHeight(sheet, 1, 26)

	// Row 2: Numbers (A2:C2 merged -> 1, then 2..13)
	_ = f.MergeCell(sheet, "A2", "C2")
	_ = f.SetCellValue(sheet, "A2", "1")
	_ = f.SetCellValue(sheet, "D2", "2")
	_ = f.SetCellValue(sheet, "E2", "3")
	_ = f.SetCellValue(sheet, "F2", "4")
	_ = f.SetCellValue(sheet, "G2", "5")
	_ = f.SetCellValue(sheet, "H2", "6")
	_ = f.SetCellValue(sheet, "I2", "7")
	_ = f.SetCellValue(sheet, "J2", "8")
	_ = f.SetCellValue(sheet, "K2", "9")
	_ = f.SetCellValue(sheet, "L2", "10")
	_ = f.SetCellValue(sheet, "M2", "11")
	_ = f.SetCellValue(sheet, "N2", "12")
	_ = f.SetCellValue(sheet, "O2", "13")

	for col := 1; col <= 15; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 2)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheet, 2, 20)

	// Row 3: Headers in Tajik
	headers := []string{
		"№ Шартномаи воридоти мол, кор ва хизматрасонӣ",
		"Санаи шартнома",
		"Санаи ба итмом расидани шартнома",
		"Ширкати хориҷии пардохт-гиранда",
		"Давлати пардохт-гиранда",
		"Маблағи умумии тибқи шартнома пардохтшуда",
		"Санаи пардохт (ҳисобнома-счет фактура)",
		"Маблағи пардохт",
		"Маблағи ҳуҷҷати тасдиқкунандаи воридоти мол (ГТД) (Асъор)",
		"Рақами тартибии ҳуҷҷати тасдиқкунандаи воридоти мол (ГТД)",
		"Фарқияти маблағи интиқолгардида бо маблағи ҲТВ мол (ГТД) (6-7)",
		"Мӯҳлати воридоти мол тибқи шартнома (сана)",
		"Мӯҳлати воридоти мол дар асл (сана)",
		"Фарқияти мӯҳлати воридоти мол тибқи шартнома ва дар асл (9-10) (рӯз)",
		"Шартномаи иловагӣ",
	}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 3)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheet, 3, 55)

	currentRow := 4
	for _, c := range data.Contracts {
		contractTransfers := len(c.Transfers)
		rowCount := contractTransfers
		if rowCount == 0 {
			rowCount = 1
		}

		var contractAALabel string
		if len(c.AdditionalAgreements) > 0 {
			var aaNums []string
			for _, aa := range c.AdditionalAgreements {
				if aa.Number != "" {
					aaNums = append(aaNums, formatDocNumber(aa.Number))
				}
			}
			contractAALabel = strings.Join(aaNums, ", ")
		}

		for i := 0; i < rowCount; i++ {
			rowIdx := currentRow + i
			_ = f.SetRowHeight(sheet, rowIdx, 22)

			if i == 0 {
				_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), formatDocNumber(c.Number))
				_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), c.Date)
				_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), c.EndDate)
				_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), c.ForeignCompany)
				_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), c.Country)
				if c.TotalAmount > 0 {
					_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), c.TotalAmount)
				}
				if contractAALabel != "" {
					_ = f.SetCellValue(sheet, fmt.Sprintf("O%d", rowIdx), contractAALabel)
				}
			}

			if i < len(c.Transfers) {
				t := c.Transfers[i]
				_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), t.InvoiceDate)
				_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), t.InvoiceAmount)
				_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), t.GTDAmount)
				_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), t.GTDNumber)
				_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", rowIdx), t.DiffAmount)
				if t.ContractDeliveryTerm > 0 {
					_ = f.SetCellValue(sheet, fmt.Sprintf("L%d", rowIdx), t.ContractDeliveryTerm)
				}
				if t.ActualDeliveryTerm > 0 {
					_ = f.SetCellValue(sheet, fmt.Sprintf("M%d", rowIdx), t.ActualDeliveryTerm)
				}
				_ = f.SetCellValue(sheet, fmt.Sprintf("N%d", rowIdx), t.DiffDays)
			}

			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), leftWrapStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), amountStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), amountStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), amountStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowIdx), fmt.Sprintf("J%d", rowIdx), centerWrapStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("K%d", rowIdx), fmt.Sprintf("K%d", rowIdx), amountStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("L%d", rowIdx), fmt.Sprintf("L%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("M%d", rowIdx), fmt.Sprintf("M%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("N%d", rowIdx), fmt.Sprintf("N%d", rowIdx), centerStyle)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("O%d", rowIdx), fmt.Sprintf("O%d", rowIdx), centerWrapStyle)
		}

		if rowCount > 1 {
			endRow := currentRow + rowCount - 1
			_ = f.MergeCell(sheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", endRow))
			_ = f.MergeCell(sheet, fmt.Sprintf("B%d", currentRow), fmt.Sprintf("B%d", endRow))
			_ = f.MergeCell(sheet, fmt.Sprintf("C%d", currentRow), fmt.Sprintf("C%d", endRow))
			_ = f.MergeCell(sheet, fmt.Sprintf("D%d", currentRow), fmt.Sprintf("D%d", endRow))
			_ = f.MergeCell(sheet, fmt.Sprintf("E%d", currentRow), fmt.Sprintf("E%d", endRow))
			_ = f.MergeCell(sheet, fmt.Sprintf("F%d", currentRow), fmt.Sprintf("F%d", endRow))
			_ = f.MergeCell(sheet, fmt.Sprintf("O%d", currentRow), fmt.Sprintf("O%d", endRow))
		}
		currentRow += rowCount

		// Render Additional Agreements directly underneath
		for _, aa := range c.AdditionalAgreements {
			aaTransfers := len(aa.Transfers)
			aaRowCount := aaTransfers
			if aaRowCount == 0 {
				aaRowCount = 1
			}

			cleanParentNum := strings.TrimPrefix(strings.TrimSpace(c.Number), "№")
			aaNote := fmt.Sprintf("Шартномаи иловагӣ ба шартномаи №%s", cleanParentNum)

			for j := 0; j < aaRowCount; j++ {
				rowIdx := currentRow + j
				_ = f.SetRowHeight(sheet, rowIdx, 22)

				if j == 0 {
					_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), formatDocNumber(aa.Number))
					_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), aa.Date)
					_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), aa.EndDate)
					_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), aa.ForeignCompany)
					_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), aa.Country)
					if aa.TotalAmount > 0 {
						_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), aa.TotalAmount)
					}
					_ = f.SetCellValue(sheet, fmt.Sprintf("O%d", rowIdx), aaNote)
				}

				if j < len(aa.Transfers) {
					t := aa.Transfers[j]
					_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), t.InvoiceDate)
					_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), t.InvoiceAmount)
					_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), t.GTDAmount)
					_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), t.GTDNumber)
					_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", rowIdx), t.DiffAmount)
					if t.ContractDeliveryTerm > 0 {
						_ = f.SetCellValue(sheet, fmt.Sprintf("L%d", rowIdx), t.ContractDeliveryTerm)
					}
					if t.ActualDeliveryTerm > 0 {
						_ = f.SetCellValue(sheet, fmt.Sprintf("M%d", rowIdx), t.ActualDeliveryTerm)
					}
					_ = f.SetCellValue(sheet, fmt.Sprintf("N%d", rowIdx), t.DiffDays)
				}

				_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), leftWrapStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), amountStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), amountStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), amountStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowIdx), fmt.Sprintf("J%d", rowIdx), centerWrapStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("K%d", rowIdx), fmt.Sprintf("K%d", rowIdx), amountStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("L%d", rowIdx), fmt.Sprintf("L%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("M%d", rowIdx), fmt.Sprintf("M%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("N%d", rowIdx), fmt.Sprintf("N%d", rowIdx), centerStyle)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("O%d", rowIdx), fmt.Sprintf("O%d", rowIdx), centerWrapStyle)
			}

			if aaRowCount > 1 {
				endRow := currentRow + aaRowCount - 1
				_ = f.MergeCell(sheet, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", endRow))
				_ = f.MergeCell(sheet, fmt.Sprintf("B%d", currentRow), fmt.Sprintf("B%d", endRow))
				_ = f.MergeCell(sheet, fmt.Sprintf("C%d", currentRow), fmt.Sprintf("C%d", endRow))
				_ = f.MergeCell(sheet, fmt.Sprintf("D%d", currentRow), fmt.Sprintf("D%d", endRow))
				_ = f.MergeCell(sheet, fmt.Sprintf("E%d", currentRow), fmt.Sprintf("E%d", endRow))
				_ = f.MergeCell(sheet, fmt.Sprintf("F%d", currentRow), fmt.Sprintf("F%d", endRow))
				_ = f.MergeCell(sheet, fmt.Sprintf("O%d", currentRow), fmt.Sprintf("O%d", endRow))
			}
			currentRow += aaRowCount
		}
	}

	_ = f.SetColWidth(sheet, "A", "A", 16)
	_ = f.SetColWidth(sheet, "B", "B", 14)
	_ = f.SetColWidth(sheet, "C", "C", 16)
	_ = f.SetColWidth(sheet, "D", "D", 26)
	_ = f.SetColWidth(sheet, "E", "E", 14)
	_ = f.SetColWidth(sheet, "F", "F", 18)
	_ = f.SetColWidth(sheet, "G", "G", 15)
	_ = f.SetColWidth(sheet, "H", "H", 16)
	_ = f.SetColWidth(sheet, "I", "I", 20)
	_ = f.SetColWidth(sheet, "J", "J", 26)
	_ = f.SetColWidth(sheet, "K", "K", 18)
	_ = f.SetColWidth(sheet, "L", "L", 16)
	_ = f.SetColWidth(sheet, "M", "M", 16)
	_ = f.SetColWidth(sheet, "N", "N", 16)
	_ = f.SetColWidth(sheet, "O", "O", 26)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SaveReportTemplate(reportType string, data []byte) error {
	ensureTemplateDir()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("некорректный файл Excel: %w", err)
	}
	defer f.Close()

	path := getTemplatePath(reportType)
	return os.WriteFile(path, data, 0644)
}

func GetReportTemplate(reportType string) ([]byte, error) {
	ensureTemplateDir()
	path := getTemplatePath(reportType)
	data, err := os.ReadFile(path)
	if err != nil {
		switch reportType {
		case dto.ReportTypeContracts:
			return GenerateContractsExcel(nil, "Пример клиента")
		case dto.ReportTypeInvoices:
			return GenerateInvoicesExcel(nil, "Пример клиента")
		case dto.ReportTypeGTD:
			return GenerateGTDExcel(nil, "Пример клиента")
		case dto.ReportTypeAdditionalAgreements:
			return GenerateAAExcel(nil, "Пример клиента")
		case dto.ReportTypeClients:
			return GenerateClientsExcel(nil)
		case dto.ReportTypeClientConsolidated:
			return GenerateClientConsolidatedExcel(&dto.ClientConsolidatedReportData{
				ClientName: "Пример клиента",
			})
		default:
			return nil, fmt.Errorf("неизвестный тип отчета: %s", reportType)
		}
	}
	return data, nil
}
