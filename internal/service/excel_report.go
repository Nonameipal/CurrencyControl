package service

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/pkg/excel"

	"github.com/xuri/excelize/v2"
)

var contractColumns = []excel.ColumnDef[dto.ContractExcelRow]{
	{Header: "№", Width: 6, Type: excel.CellCenter, GetValue: func(r dto.ContractExcelRow) any { return r.RowNum }},
	{Header: "Номер контракта", Width: 18, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.Number }},
	{Header: "Дата контракта", Width: 14, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.Date }},
	{Header: "Предмет", Width: 25, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.Subject }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.ContractExcelRow) any { return r.Amount }},
	{Header: "Валюта", Width: 10, Type: excel.CellCenter, GetValue: func(r dto.ContractExcelRow) any { return r.Currency }},
	{Header: "Срок контракта", Width: 16, Type: excel.CellCenter, GetValue: func(r dto.ContractExcelRow) any { return r.ContractTerm }},
	{Header: "Срок возврата", Width: 15, Type: excel.CellCenter, GetValue: func(r dto.ContractExcelRow) any { return r.ReturnDays }},
	{Header: "Срок поставки", Width: 14, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.DeliveryDate }},
	{Header: "Дата окончания", Width: 16, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.ContractEndDate }},
	{Header: "Отправитель", Width: 25, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.SenderName }},
	{Header: "Наименование получателя", Width: 25, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.ReceiverName }},
	{Header: "Страна получателя", Width: 18, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.ReceiverCountry }},
}

var invoiceColumns = []excel.ColumnDef[dto.InvoiceExcelRow]{
	{Header: "№", Width: 6, Type: excel.CellCenter, GetValue: func(r dto.InvoiceExcelRow) any { return r.RowNum }},
	{Header: "Номер инвойса", Width: 20, Type: excel.CellText, GetValue: func(r dto.InvoiceExcelRow) any { return r.Number }},
	{Header: "Дата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.InvoiceExcelRow) any { return r.Date }},
	{Header: "Валюта", Width: 12, Type: excel.CellCenter, GetValue: func(r dto.InvoiceExcelRow) any { return r.Currency }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.InvoiceExcelRow) any { return r.Amount }},
}

var gtdColumns = []excel.ColumnDef[dto.GTDExcelRow]{
	{Header: "Номер ГТД", Width: 20, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.Number }},
	{Header: "Дата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.GTDExcelRow) any { return r.Date }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.GTDExcelRow) any { return r.Amount }},
	{Header: "Валюта", Width: 10, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.Currency }},
	{Header: "HS CODE", Width: 14, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any {
		if strings.TrimSpace(r.HSCode) == "" {
			return domain.DefaultHSCode
		}
		return r.HSCode
	}},
	{Header: "Наименование отправителя", Width: 28, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.SenderName }},
	{Header: "Страна отправителя", Width: 20, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.Country }},
}

var clientColumns = []excel.ColumnDef[dto.ClientExcelRow]{
	{Header: "ИНН", Width: 16, Type: excel.CellText, GetValue: func(r dto.ClientExcelRow) any { return r.INN }},
	{Header: "Наименование компании", Width: 30, Type: excel.CellText, GetValue: func(r dto.ClientExcelRow) any { return r.Name }},
	{Header: "Филиал", Width: 25, Type: excel.CellText, GetValue: func(r dto.ClientExcelRow) any { return r.BranchName }},
	{Header: "Кол-во контрактов", Width: 18, Type: excel.CellInt, GetValue: func(r dto.ClientExcelRow) any { return r.ContractsCount }},
	{Header: "Общая сумма", Width: 20, Type: excel.CellAmount, GetValue: func(r dto.ClientExcelRow) any { return r.TotalAmount }},
	{Header: "Дата создания", Width: 15, Type: excel.CellDate, GetValue: func(r dto.ClientExcelRow) any { return r.CreatedAt }},
}

func GenerateContractsExcel(rows []dto.ContractExcelRow, clientName string) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.ContractExcelRow]{
		SheetName: "Контракты",
		Title:     fmt.Sprintf("Контракты \"%s\"", clientName),
		Columns:   contractColumns,
		Rows:      rows,
	})
}

func GenerateInvoicesExcel(rows []dto.InvoiceExcelRow, clientName string) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.InvoiceExcelRow]{
		SheetName: "Инвойсы",
		Title:     fmt.Sprintf("Инвойсы \"%s\"", clientName),
		Columns:   invoiceColumns,
		Rows:      rows,
	})
}

func GenerateGTDExcel(rows []dto.GTDExcelRow, clientName string) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.GTDExcelRow]{
		SheetName: "ГТД",
		Title:     fmt.Sprintf("ГТД \"%s\"", clientName),
		Columns:   gtdColumns,
		Rows:      rows,
	})
}

func GenerateClientsExcel(rows []dto.ClientExcelRow) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.ClientExcelRow]{
		SheetName: "Клиенты",
		Title:     "Отчет по клиентам банка",
		Columns:   clientColumns,
		Rows:      rows,
	})
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

func applyConsolidatedRowStyles(f *excelize.File, sheet string, rowIdx int, styles excel.Styles) {
	colStyles := []int{
		styles.Center,     // A: № шартнома
		styles.Center,     // B: Санаи шартнома
		styles.Center,     // C: Санаи ба итмом расидани шартнома
		styles.LeftWrap,   // D: Ширкати хориҷии пардохт-гиранда
		styles.Center,     // E: Давлати пардохт-гиранда
		styles.Center,     // F: Асъор
		styles.Amount,     // G: Маблағи умумӣ
		styles.Center,     // H: Санаи пардохт
		styles.Amount,     // I: Маблағи пардохт
		styles.Amount,     // J: Маблағи ГТД
		styles.CenterWrap, // K: Рақами ГТД
		styles.Amount,     // L: Фарқияти маблағ
		styles.Center,     // M: Мӯҳлати тибқи шартнома
		styles.Center,     // N: Мӯҳлати дар асл
		styles.CenterWrap, // O: Фарқияти мӯҳлат
	}
	for colIdx, st := range colStyles {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx)
		_ = f.SetCellStyle(sheet, cell, cell, st)
	}
}

func populateConsolidatedTransfer(f *excelize.File, sheet string, rowIdx int, t dto.ClientConsolidatedTransfer) {
	_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), t.InvoiceDate)
	_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), t.InvoiceAmount)
	_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), t.GTDAmount)
	_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", rowIdx), t.GTDNumber)
	_ = f.SetCellValue(sheet, fmt.Sprintf("L%d", rowIdx), t.DiffAmount)
	if t.ContractDeliveryTerm > 0 {
		_ = f.SetCellValue(sheet, fmt.Sprintf("M%d", rowIdx), t.ContractDeliveryTerm)
	}
	if t.ActualDeliveryTerm > 0 {
		_ = f.SetCellValue(sheet, fmt.Sprintf("N%d", rowIdx), t.ActualDeliveryTerm)
	}
	_ = f.SetCellValue(sheet, fmt.Sprintf("O%d", rowIdx), t.DiffDays)
}

func mergeConsolidatedCells(f *excelize.File, sheet string, startRow, endRow int) {
	if startRow >= endRow {
		return
	}
	for _, col := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		_ = f.MergeCell(sheet, fmt.Sprintf("%s%d", col, startRow), fmt.Sprintf("%s%d", col, endRow))
	}
}

type consolidatedBlock struct {
	number         string
	date           string
	endDate        string
	foreignCompany string
	country        string
	currency       string
	totalAmount    float64
	transfers      []dto.ClientConsolidatedTransfer
}

func writeConsolidatedBlock(f *excelize.File, sheet string, startRow int, b consolidatedBlock, styles excel.Styles) int {
	rowCount := len(b.transfers)
	if rowCount == 0 {
		rowCount = 1
	}
	for i := 0; i < rowCount; i++ {
		rowIdx := startRow + i
		_ = f.SetRowHeight(sheet, rowIdx, 22)
		if i == 0 {
			if b.number != "" {
				_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), b.number)
			}
			if b.date != "" {
				_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), b.date)
			}
			if b.endDate != "" {
				_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), b.endDate)
			}
			_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), b.foreignCompany)
			_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), b.country)
			_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), b.currency)
			if b.totalAmount > 0 {
				_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), b.totalAmount)
			}
		}
		if i < len(b.transfers) {
			populateConsolidatedTransfer(f, sheet, rowIdx, b.transfers[i])
		}
		applyConsolidatedRowStyles(f, sheet, rowIdx, styles)
	}
	mergeConsolidatedCells(f, sheet, startRow, startRow+rowCount-1)
	return startRow + rowCount
}

func GenerateClientConsolidatedExcel(data *dto.ClientConsolidatedReportData) ([]byte, error) {
	sheet := "Умуми"
	f := excel.CreateBaseFile(sheet)
	defer f.Close()

	styles := excel.InitStyles(f)

	if data == nil {
		data = &dto.ClientConsolidatedReportData{ClientName: ""}
	}

	titleText := strings.TrimSpace(data.ClientName)
	if !strings.HasPrefix(titleText, "Шахси воқеӣ") && !strings.HasPrefix(titleText, "Шахси ҳуқуқӣ") {
		if titleText != "" {
			lowerType := strings.ToLower(data.ClientType)
			if data.ClientType == domain.ClientTypeIndividual || data.ClientType == domain.ClientTypeSoleProprietor ||
				strings.Contains(lowerType, "физ") || strings.Contains(lowerType, "ип") || strings.Contains(lowerType, "предприниматель") {
				titleText = "Шахси воқеӣ " + titleText
			} else {
				titleText = "Шахси ҳуқуқӣ " + titleText
			}
		} else {
			titleText = "Шахси ҳуқуқӣ"
		}
	}
	_ = f.MergeCell(sheet, "A1", "O1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "O1", styles.Title)
	_ = f.SetRowHeight(sheet, 1, 26)

	for col := 1; col <= 15; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 2)
		_ = f.SetCellValue(sheet, cell, strconv.Itoa(col))
		_ = f.SetCellStyle(sheet, cell, cell, styles.Header)
	}
	_ = f.SetRowHeight(sheet, 2, 20)

	headers := []string{
		"№ Шартномаи воридоти мол, кор ва хизматрасонӣ",
		"Санаи шартнома",
		"Санаи ба итмом расидани шартнома",
		"Ширкати хориҷии пардохт-гиранда",
		"Давлати пардохт-гиранда",
		"Асъор",
		"Маблағи умумии тибқи шартнома пардохтшуда",
		"Санаи пардохт (ҳисобнома-счет фактура)",
		"Маблағи пардохт",
		"Маблағи ҳуҷҷати тасдиқкунандаи воридоти мол (ГТД)",
		"Рақами тартибии ҳуҷҷати тасдиқкунандаи воридоти мол (ГТД)",
		"Фарқияти маблағи интиқолгардида бо маблағи ҲТВ мол (ГТД) (10-11)",
		"Мӯҳлати воридоти мол тибқи шартнома (сана)",
		"Мӯҳлати воридоти мол дар асл (сана)",
		"Фарқияти мӯҳлати воридоти мол тибқи шартнома ва дар асл (13-14) (рӯз)",
	}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 3)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, styles.Header)
	}
	_ = f.SetRowHeight(sheet, 3, 55)

	currentRow := 4
	for _, c := range data.Contracts {
		currentRow = writeConsolidatedBlock(f, sheet, currentRow, consolidatedBlock{
			number:         formatDocNumber(c.Number),
			date:           c.Date,
			endDate:        c.EndDate,
			foreignCompany: c.ForeignCompany,
			country:        c.Country,
			currency:       c.Currency,
			totalAmount:    c.TotalAmount,
			transfers:      c.Transfers,
		}, styles)
		for _, aa := range c.AdditionalAgreements {
			currentRow = writeConsolidatedBlock(f, sheet, currentRow, consolidatedBlock{
				number:         formatDocNumber(aa.Number),
				date:           aa.Date,
				endDate:        aa.EndDate,
				foreignCompany: aa.ForeignCompany,
				country:        aa.Country,
				currency:       aa.Currency,
				totalAmount:    aa.TotalAmount,
				transfers:      aa.Transfers,
			}, styles)
		}
	}

	minColWidths := []float64{16, 14, 16, 26, 14, 10, 18, 15, 16, 18, 26, 18, 14, 14, 16}
	for colIdx, minW := range minColWidths {
		colNum := colIdx + 1
		colLetter, _ := excelize.ColumnNumberToName(colNum)
		maxLen := 0
		for row := 4; row < currentRow; row++ {
			cell, _ := excelize.CoordinatesToCellName(colNum, row)
			val, err := f.GetCellValue(sheet, cell)
			if err != nil || val == "" {
				continue
			}
			for _, line := range strings.Split(val, "\n") {
				runes := len([]rune(strings.TrimSpace(line)))
				if runes > maxLen {
					maxLen = runes
				}
			}
		}
		w := float64(maxLen) + 4
		if w < minW {
			w = minW
		}
		if w > 50 {
			w = 50
		}
		_ = f.SetColWidth(sheet, colLetter, colLetter, w)
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
