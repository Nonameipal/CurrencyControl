package service

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/pkg/excel"

	"github.com/xuri/excelize/v2"
)



var contractColumns = []excel.ColumnDef[dto.ContractExcelRow]{
	{Header: "Номер", Width: 16, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.Number }},
	{Header: "Дата", Width: 13, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.Date }},
	{Header: "Предмет", Width: 25, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.Subject }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.ContractExcelRow) any { return r.Amount }},
	{Header: "Валюта", Width: 10, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.Currency }},
	{Header: "Срок возврата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.ReturnDate }},
	{Header: "Срок поставки", Width: 14, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.DeliveryDate }},
	{Header: "Дата окончании контракта", Width: 24, Type: excel.CellDate, GetValue: func(r dto.ContractExcelRow) any { return r.ContractEndDate }},
	{Header: "Наименование получателя", Width: 24, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.ReceiverName }},
	{Header: "Счет получателя", Width: 20, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.ReceiverAccount }},
	{Header: "Страна получателя", Width: 18, Type: excel.CellText, GetValue: func(r dto.ContractExcelRow) any { return r.ReceiverCountry }},
}

var invoiceColumns = []excel.ColumnDef[dto.InvoiceExcelRow]{
	{Header: "Номер", Width: 18, Type: excel.CellText, GetValue: func(r dto.InvoiceExcelRow) any { return r.Number }},
	{Header: "Дата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.InvoiceExcelRow) any { return r.Date }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.InvoiceExcelRow) any { return r.Amount }},
	{Header: "Валюта", Width: 10, Type: excel.CellText, GetValue: func(r dto.InvoiceExcelRow) any { return r.Currency }},
	{Header: "HS CODE", Width: 14, Type: excel.CellText, GetValue: func(r dto.InvoiceExcelRow) any { return r.HSCode }},
	{Header: "Назначение оплаты товар/услуга", Width: 35, Type: excel.CellText, GetValue: func(r dto.InvoiceExcelRow) any { return r.PaymentPurpose }},
}

var gtdColumns = []excel.ColumnDef[dto.GTDExcelRow]{
	{Header: "Номер ГТД", Width: 20, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.Number }},
	{Header: "Дата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.GTDExcelRow) any { return r.Date }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.GTDExcelRow) any { return r.Amount }},
	{Header: "Валюта", Width: 10, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.Currency }},
	{Header: "HS CODE", Width: 14, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.HSCode }},
	{Header: "Наименование отправителя", Width: 28, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.SenderName }},
	{Header: "Страна отправителя", Width: 20, Type: excel.CellText, GetValue: func(r dto.GTDExcelRow) any { return r.Country }},
}

var aaColumns = []excel.ColumnDef[dto.AAExcelRow]{
	{Header: "Номер", Width: 16, Type: excel.CellText, GetValue: func(r dto.AAExcelRow) any { return r.Number }},
	{Header: "Тип документа", Width: 18, Type: excel.CellText, GetValue: func(r dto.AAExcelRow) any { return r.DocType }},
	{Header: "Дата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.AAExcelRow) any { return r.Date }},
	{Header: "Номер контракта", Width: 18, Type: excel.CellText, GetValue: func(r dto.AAExcelRow) any { return r.ContractNum }},
	{Header: "Сумма", Width: 16, Type: excel.CellAmount, GetValue: func(r dto.AAExcelRow) any { return r.Amount }},
	{Header: "Валюта", Width: 10, Type: excel.CellText, GetValue: func(r dto.AAExcelRow) any { return r.Currency }},
	{Header: "Срок поставки", Width: 14, Type: excel.CellDate, GetValue: func(r dto.AAExcelRow) any { return r.DeliveryDate }},
	{Header: "Срок возврата", Width: 14, Type: excel.CellDate, GetValue: func(r dto.AAExcelRow) any { return r.ReturnDate }},
	{Header: "Продление до", Width: 14, Type: excel.CellDate, GetValue: func(r dto.AAExcelRow) any { return r.ExtendDateTo }},
	{Header: "Предмет", Width: 28, Type: excel.CellText, GetValue: func(r dto.AAExcelRow) any { return r.Subject }},
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
		ReportType: dto.ReportTypeContracts,
		SheetName:  "Контракты",
		Title:      fmt.Sprintf("Контракты \"%s\"", clientName),
		Columns:    contractColumns,
		Rows:       rows,
	})
}

func GenerateInvoicesExcel(rows []dto.InvoiceExcelRow, clientName string) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.InvoiceExcelRow]{
		ReportType: dto.ReportTypeInvoices,
		SheetName:  "Инвойсы",
		Title:      fmt.Sprintf("Инвойсы \"%s\"", clientName),
		Columns:    invoiceColumns,
		Rows:       rows,
	})
}

func GenerateGTDExcel(rows []dto.GTDExcelRow, clientName string) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.GTDExcelRow]{
		ReportType: dto.ReportTypeGTD,
		SheetName:  "ГТД",
		Title:      fmt.Sprintf("ГТД \"%s\"", clientName),
		Columns:    gtdColumns,
		Rows:       rows,
	})
}

func GenerateAAExcel(rows []dto.AAExcelRow, clientName string) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.AAExcelRow]{
		ReportType: dto.ReportTypeAdditionalAgreements,
		SheetName:  "Доп. соглашения",
		Title:      fmt.Sprintf("Дополнительные соглашения \"%s\"", clientName),
		Columns:    aaColumns,
		Rows:       rows,
	})
}

func GenerateClientsExcel(rows []dto.ClientExcelRow) ([]byte, error) {
	return excel.GenerateTable(excel.TableConfig[dto.ClientExcelRow]{
		ReportType: dto.ReportTypeClients,
		SheetName:  "Клиенты",
		Title:      "Отчет по клиентам банка",
		Columns:    clientColumns,
		Rows:       rows,
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

func GenerateClientConsolidatedExcel(data *dto.ClientConsolidatedReportData) ([]byte, error) {
	f, sheet := excel.OpenFileOrNew(dto.ReportTypeClientConsolidated, "Умуми")
	defer f.Close()

	styles := excel.InitStyles(f)

	if data == nil {
		data = &dto.ClientConsolidatedReportData{ClientName: ""}
	}

	titleText := "Клиент"
	if strings.TrimSpace(data.ClientName) != "" {
		titleText = fmt.Sprintf("Клиент: %s", data.ClientName)
	}
	_ = f.MergeCell(sheet, "A1", "O1")
	_ = f.SetCellValue(sheet, "A1", titleText)
	_ = f.SetCellStyle(sheet, "A1", "O1", styles.Title)
	_ = f.SetRowHeight(sheet, 1, 26)

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
		_ = f.SetCellStyle(sheet, cell, cell, styles.Header)
	}
	_ = f.SetRowHeight(sheet, 2, 20)

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
		_ = f.SetCellStyle(sheet, cell, cell, styles.Header)
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

			_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), styles.LeftWrap)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), styles.Amount)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), styles.Amount)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), styles.Amount)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowIdx), fmt.Sprintf("J%d", rowIdx), styles.CenterWrap)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("K%d", rowIdx), fmt.Sprintf("K%d", rowIdx), styles.Amount)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("L%d", rowIdx), fmt.Sprintf("L%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("M%d", rowIdx), fmt.Sprintf("M%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("N%d", rowIdx), fmt.Sprintf("N%d", rowIdx), styles.Center)
			_ = f.SetCellStyle(sheet, fmt.Sprintf("O%d", rowIdx), fmt.Sprintf("O%d", rowIdx), styles.CenterWrap)
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

				_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), styles.LeftWrap)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), styles.Amount)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), styles.Amount)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), styles.Amount)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowIdx), fmt.Sprintf("J%d", rowIdx), styles.CenterWrap)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("K%d", rowIdx), fmt.Sprintf("K%d", rowIdx), styles.Amount)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("L%d", rowIdx), fmt.Sprintf("L%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("M%d", rowIdx), fmt.Sprintf("M%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("N%d", rowIdx), fmt.Sprintf("N%d", rowIdx), styles.Center)
				_ = f.SetCellStyle(sheet, fmt.Sprintf("O%d", rowIdx), fmt.Sprintf("O%d", rowIdx), styles.CenterWrap)
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
	excel.EnsureTemplateDir()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("некорректный файл Excel: %w", err)
	}
	defer f.Close()

	path := excel.GetTemplatePath(reportType)
	return os.WriteFile(path, data, 0644)
}

func GetReportTemplate(reportType string) ([]byte, error) {
	excel.EnsureTemplateDir()
	path := excel.GetTemplatePath(reportType)
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
