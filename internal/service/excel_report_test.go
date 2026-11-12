package service_test

import (
	"bytes"
	"testing"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service"

	"github.com/xuri/excelize/v2"
)

func TestExcelReportGenerators(t *testing.T) {
	t.Run("Contracts Excel Generation", func(t *testing.T) {
		rows := []dto.ContractExcelRow{
			{
				RowNum:          1,
				Number:          "CNTR-001",
				Date:            "01.09.2026",
				Subject:         "Поставка оборудования",
				Amount:          50000.00,
				Currency:        "USD",
				ContractTerm:    "121",
				ReturnDays:      "30",
				DeliveryDate:    "01.11.2026",
				ContractEndDate: "31.12.2026",
				SenderName:      "ООО Поставщик",
				ReceiverName:    "ООО Рога и Копыта",
				ReceiverCountry: "Китай",
			},
		}
		data, err := service.GenerateContractsExcel(rows, "Тестовая Компания")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("generated data is empty")
		}

		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("cannot open generated excel: %v", err)
		}
		defer f.Close()

		sheet := f.GetSheetList()[0]
		titleVal, _ := f.GetCellValue(sheet, "A1")
		if titleVal != "Контракты \"Тестовая Компания\"" {
			t.Errorf("expected title 'Контракты \"Тестовая Компания\"', got %q", titleVal)
		}

		col1, _ := f.GetCellValue(sheet, "A2")
		if col1 != "№" {
			t.Errorf("expected header '№', got %q", col1)
		}

		col2, _ := f.GetCellValue(sheet, "B2")
		if col2 != "Номер контракта" {
			t.Errorf("expected header 'Номер контракта', got %q", col2)
		}

		dataNum, _ := f.GetCellValue(sheet, "B3")
		if dataNum != "CNTR-001" {
			t.Errorf("expected contract number 'CNTR-001', got %q", dataNum)
		}
	})

	t.Run("Invoices Excel Generation", func(t *testing.T) {
		rows := []dto.InvoiceExcelRow{
			{
				RowNum:   1,
				Number:   "INV-100",
				Date:     "05.09.2026",
				Currency: "USD",
				Amount:   25000.00,
			},
		}
		data, err := service.GenerateInvoicesExcel(rows, "Тестовая Компания")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("cannot open generated excel: %v", err)
		}
		defer f.Close()

		sheet := f.GetSheetList()[0]
		titleVal, _ := f.GetCellValue(sheet, "A1")
		if titleVal != "Инвойсы \"Тестовая Компания\"" {
			t.Errorf("expected title 'Инвойсы \"Тестовая Компания\"', got %q", titleVal)
		}

		col1, _ := f.GetCellValue(sheet, "A2")
		if col1 != "№" {
			t.Errorf("expected header '№', got %q", col1)
		}

		col2, _ := f.GetCellValue(sheet, "B2")
		if col2 != "Номер инвойса" {
			t.Errorf("expected header 'Номер инвойса', got %q", col2)
		}

		col4, _ := f.GetCellValue(sheet, "D2")
		if col4 != "Валюта" {
			t.Errorf("expected header 'Валюта', got %q", col4)
		}

		col5, _ := f.GetCellValue(sheet, "E2")
		if col5 != "Сумма" {
			t.Errorf("expected header 'Сумма', got %q", col5)
		}

		rowNum, _ := f.GetCellValue(sheet, "A3")
		if rowNum != "1" {
			t.Errorf("expected row number '1', got %q", rowNum)
		}

		invNum, _ := f.GetCellValue(sheet, "B3")
		if invNum != "INV-100" {
			t.Errorf("expected invoice number 'INV-100', got %q", invNum)
		}

		curr, _ := f.GetCellValue(sheet, "D3")
		if curr != "USD" {
			t.Errorf("expected currency 'USD', got %q", curr)
		}
	})

	t.Run("GTD Excel Generation", func(t *testing.T) {
		rows := []dto.GTDExcelRow{
			{
				Number:     "GTD-999",
				Date:       "10.09.2026",
				Amount:     25000.00,
				Currency:   "USD",
				HSCode:     "8471300000",
				SenderName: "Global Tech Ltd",
				Country:    "Германия",
			},
		}
		data, err := service.GenerateGTDExcel(rows, "Тестовая Компания")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("cannot open generated excel: %v", err)
		}
		defer f.Close()

		sheet := f.GetSheetList()[0]
		titleVal, _ := f.GetCellValue(sheet, "A1")
		if titleVal != "ГТД \"Тестовая Компания\"" {
			t.Errorf("expected title 'ГТД \"Тестовая Компания\"', got %q", titleVal)
		}

		sender, _ := f.GetCellValue(sheet, "F3")
		if sender != "Global Tech Ltd" {
			t.Errorf("expected sender 'Global Tech Ltd', got %q", sender)
		}
	})

	t.Run("Empty HSCode Fallback to 'Нет кода'", func(t *testing.T) {
		gtdRows := []dto.GTDExcelRow{
			{
				Number:     "GTD-EMPTY-HS",
				Date:       "01.01.2026",
				Amount:     1000.00,
				Currency:   "USD",
				HSCode:     "",
				SenderName: "Test Sender",
				Country:    "Германия",
			},
		}
		gtdData, err := service.GenerateGTDExcel(gtdRows, "Компания")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fGTD, err := excelize.OpenReader(bytes.NewReader(gtdData))
		if err != nil {
			t.Fatalf("cannot open generated excel: %v", err)
		}
		defer fGTD.Close()
		gtdSheet := fGTD.GetSheetList()[0]
		valGTD, _ := fGTD.GetCellValue(gtdSheet, "E3")
		if valGTD != "Нет кода" {
			t.Errorf("expected HS code 'Нет кода', got %q", valGTD)
		}
	})

	t.Run("Client Consolidated Excel Generation", func(t *testing.T) {
		reportData := &dto.ClientConsolidatedReportData{
			ClientName: "LLC American Company",
			Contracts: []dto.ClientConsolidatedContract{
				{
					Number:         "37732",
					Date:           "13.02.2024",
					EndDate:        "12.02.2025",
					ForeignCompany: "LLC American beef",
					Country:        "Америка",
					TotalAmount:    500000.00,
					Currency:       "USD",
					Transfers: []dto.ClientConsolidatedTransfer{
						{
							InvoiceDate:          "17.02.2024",
							InvoiceAmount:        125000.00,
							GTDAmount:            120000.00,
							GTDNumber:            "775857/229658/784475",
							DiffAmount:           5000.00,
							ContractDeliveryTerm: 180,
							ActualDeliveryTerm:   180,
							DiffDays:             "0",
						},
						{
							InvoiceDate:          "30.05.2024",
							InvoiceAmount:        125000.00,
							GTDAmount:            125000.00,
							GTDNumber:            "775859/229658/788575",
							DiffAmount:           0.00,
							ContractDeliveryTerm: 120,
							ActualDeliveryTerm:   110,
							DiffDays:             "+10",
						},
					},
					AdditionalAgreements: []dto.ClientConsolidatedAA{
						{
							Number:               "25748",
							Date:                 "12.02.2025",
							EndDate:              "01.01.2026",
							ForeignCompany:       "LLC American beef",
							Country:              "Америка",
							TotalAmount:          500000.00,
							Currency:             "RUB",
							ParentContractNumber: "37732",
							Transfers: []dto.ClientConsolidatedTransfer{
								{
									InvoiceDate:          "01.04.2025",
									InvoiceAmount:        200000.00,
									GTDAmount:            198000.00,
									GTDNumber:            "856858/253256/457856",
									DiffAmount:           2000.00,
									ContractDeliveryTerm: 60,
									ActualDeliveryTerm:   48,
									DiffDays:             "+12",
								},
								{
									InvoiceDate:          "03.07.2025",
									InvoiceAmount:        400000.00,
									GTDAmount:            0,
									GTDNumber:            "",
									DiffAmount:           400000.00,
									ContractDeliveryTerm: 30,
									ActualDeliveryTerm:   63,
									DiffDays:             "-33",
								},
							},
						},
					},
				},
			},
		}

		data, err := service.GenerateClientConsolidatedExcel(reportData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("generated data is empty")
		}

		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("cannot open generated excel: %v", err)
		}
		defer f.Close()

		sheet := f.GetSheetList()[0]
		titleVal, _ := f.GetCellValue(sheet, "A1")
		if titleVal != "Шахси ҳуқуқӣ LLC American Company" {
			t.Errorf("expected title 'Шахси ҳуқуқӣ LLC American Company', got %q", titleVal)
		}

		h1, _ := f.GetCellValue(sheet, "A2")
		if h1 != "1" {
			t.Errorf("expected header index '1', got %q", h1)
		}
		h6, _ := f.GetCellValue(sheet, "F2")
		if h6 != "6" {
			t.Errorf("expected header index '6', got %q", h6)
		}
		h15, _ := f.GetCellValue(sheet, "O2")
		if h15 != "15" {
			t.Errorf("expected header index '15', got %q", h15)
		}

		th1, _ := f.GetCellValue(sheet, "A3")
		if th1 != "№ Шартномаи воридоти мол, кор ва хизматрасонӣ" {
			t.Errorf("expected th1 '№ Шартномаи воридоти мол, кор ва хизматрасонӣ', got %q", th1)
		}
		th6, _ := f.GetCellValue(sheet, "F3")
		if th6 != "Асъор" {
			t.Errorf("expected th6 'Асъор', got %q", th6)
		}
		th12, _ := f.GetCellValue(sheet, "L3")
		if th12 != "Фарқияти маблағи интиқолгардида бо маблағи ҲТВ мол (ГТД) (10-11)" {
			t.Errorf("expected th12 with (10-11), got %q", th12)
		}
		th15, _ := f.GetCellValue(sheet, "O3")
		if th15 != "Фарқияти мӯҳлати воридоти мол тибқи шартнома ва дар асл (13-14) (рӯз)" {
			t.Errorf("expected th15 with (13-14), got %q", th15)
		}

		// Contract row
		cNum, _ := f.GetCellValue(sheet, "A4")
		if cNum != "№37732" {
			t.Errorf("expected contract number '№37732', got %q", cNum)
		}
		cCurr, _ := f.GetCellValue(sheet, "F4")
		if cCurr != "USD" {
			t.Errorf("expected contract currency 'USD', got %q", cCurr)
		}
		diffDays1, _ := f.GetCellValue(sheet, "O4")
		if diffDays1 != "0" {
			t.Errorf("expected diff days '0', got %q", diffDays1)
		}
		diffDays2, _ := f.GetCellValue(sheet, "O5")
		if diffDays2 != "+10" {
			t.Errorf("expected diff days '+10', got %q", diffDays2)
		}

		// AA block directly underneath (different currency RUB, cols A-C empty)
		for _, col := range []string{"A", "B", "C"} {
			val, _ := f.GetCellValue(sheet, col+"6")
			if val != "" {
				t.Errorf("expected %s6 to be empty for AA block, got %q", col, val)
			}
		}
		aaReceiver, _ := f.GetCellValue(sheet, "D6")
		if aaReceiver != "LLC American beef" {
			t.Errorf("expected AA receiver 'LLC American beef', got %q", aaReceiver)
		}
		aaCurr, _ := f.GetCellValue(sheet, "F6")
		if aaCurr != "RUB" {
			t.Errorf("expected AA currency 'RUB', got %q", aaCurr)
		}
		aaDiffDays1, _ := f.GetCellValue(sheet, "O6")
		if aaDiffDays1 != "+12" {
			t.Errorf("expected AA diff days '+12', got %q", aaDiffDays1)
		}
		aaNoGTDNum, _ := f.GetCellValue(sheet, "K7")
		if aaNoGTDNum != "" {
			t.Errorf("expected empty GTD number in K7, got %q", aaNoGTDNum)
		}
		aaDiffDays2, _ := f.GetCellValue(sheet, "O7")
		if aaDiffDays2 != "-33" {
			t.Errorf("expected AA diff days '-33', got %q", aaDiffDays2)
		}

		wA, _ := f.GetColWidth(sheet, "A")
		if wA < 16 {
			t.Errorf("expected col A width >= 16, got %f", wA)
		}

		dataIndiv, err := service.GenerateClientConsolidatedExcel(&dto.ClientConsolidatedReportData{
			ClientName: "Иванов Иван",
			ClientType: "individual",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		fIndiv, err := excelize.OpenReader(bytes.NewReader(dataIndiv))
		if err != nil {
			t.Fatalf("cannot open generated excel: %v", err)
		}
		defer fIndiv.Close()
		sheetIndiv := fIndiv.GetSheetList()[0]
		titleIndiv, _ := fIndiv.GetCellValue(sheetIndiv, "A1")
		if titleIndiv != "Шахси воқеӣ Иванов Иван" {
			t.Errorf("expected title 'Шахси воқеӣ Иванов Иван', got %q", titleIndiv)
		}
	})

	t.Run("Template Save and Retrieve", func(t *testing.T) {
		initialTpl, err := service.GetReportTemplate(dto.ReportTypeClientConsolidated)
		if err != nil {
			t.Fatalf("failed to get template: %v", err)
		}
		if len(initialTpl) == 0 {
			t.Fatal("template is empty")
		}

		err = service.SaveReportTemplate(dto.ReportTypeClientConsolidated, initialTpl)
		if err != nil {
			t.Fatalf("failed to save template: %v", err)
		}
	})
}
