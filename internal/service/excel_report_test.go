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
				Number:          "CNTR-001",
				Date:            "01.09.2026",
				Subject:         "Поставка оборудования",
				Amount:          50000.00,
				Currency:        "USD",
				ReturnDays:      "30",
				DeliveryDate:    "01.11.2026",
				ContractEndDate: "31.12.2026",
				ReceiverName:    "ООО Рога и Копыта",
				ReceiverAccount: "20202840123456789012",
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
		if col1 != "Номер" {
			t.Errorf("expected header 'Номер', got %q", col1)
		}

		dataNum, _ := f.GetCellValue(sheet, "A3")
		if dataNum != "CNTR-001" {
			t.Errorf("expected contract number 'CNTR-001', got %q", dataNum)
		}
	})

	t.Run("Invoices Excel Generation", func(t *testing.T) {
		rows := []dto.InvoiceExcelRow{
			{
				Number:         "INV-100",
				Date:           "05.09.2026",
				Amount:         25000.00,
				Currency:       "USD",
				HSCode:         "8471300000",
				PaymentPurpose: "Оплата за серверы",
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

		hsCode, _ := f.GetCellValue(sheet, "E3")
		if hsCode != "8471300000" {
			t.Errorf("expected HS code '8471300000', got %q", hsCode)
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

	t.Run("Additional Agreements Excel Generation", func(t *testing.T) {
		rows := []dto.AAExcelRow{
			{
				Number:       "AA-01",
				DocType:      "Доп. соглашение",
				Date:         "12.09.2026",
				ContractNum:  "CNTR-001",
				Amount:       10000.00,
				Currency:     "USD",
				DeliveryDate: "15.11.2026",
				ReturnDays:   "30",
				ExtendDateTo: "15.01.2027",
				Subject:      "Продление сроков",
			},
		}
		data, err := service.GenerateAAExcel(rows, "Тестовая Компания")
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
		if titleVal != "Дополнительные соглашения \"Тестовая Компания\"" {
			t.Errorf("expected title, got %q", titleVal)
		}

		docType, _ := f.GetCellValue(sheet, "B3")
		if docType != "Доп. соглашение" {
			t.Errorf("expected docType 'Доп. соглашение', got %q", docType)
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
							ForeignCompany:       "LLC American Bread",
							Country:              "Америка",
							TotalAmount:          1000000.00,
							Currency:             "USD",
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
									GTDAmount:            400000.00,
									GTDNumber:            "652145/558965/123356",
									DiffAmount:           0.00,
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
		if titleVal != "Клиент: LLC American Company" {
			t.Errorf("expected title 'Клиент: LLC American Company', got %q", titleVal)
		}

		h1, _ := f.GetCellValue(sheet, "A2")
		if h1 != "1" {
			t.Errorf("expected header index '1', got %q", h1)
		}
		h13, _ := f.GetCellValue(sheet, "O2")
		if h13 != "13" {
			t.Errorf("expected header index '13', got %q", h13)
		}

		th1, _ := f.GetCellValue(sheet, "A3")
		if th1 != "№ Шартномаи воридоти мол, кор ва хизматрасонӣ" {
			t.Errorf("expected th1 '№ Шартномаи воридоти мол, кор ва хизматрасонӣ', got %q", th1)
		}

		// Contract row
		cNum, _ := f.GetCellValue(sheet, "A4")
		if cNum != "№37732" {
			t.Errorf("expected contract number '№37732', got %q", cNum)
		}
		diffDays1, _ := f.GetCellValue(sheet, "N4")
		if diffDays1 != "0" {
			t.Errorf("expected diff days '0', got %q", diffDays1)
		}
		diffDays2, _ := f.GetCellValue(sheet, "N5")
		if diffDays2 != "+10" {
			t.Errorf("expected diff days '+10', got %q", diffDays2)
		}

		// AA row directly underneath
		aaNum, _ := f.GetCellValue(sheet, "A6")
		if aaNum != "№25748" {
			t.Errorf("expected AA number '№25748', got %q", aaNum)
		}
		aaNote, _ := f.GetCellValue(sheet, "O6")
		if aaNote != "Шартномаи иловагӣ ба шартномаи №37732" {
			t.Errorf("expected AA note 'Шартномаи иловагӣ ба шартномаи №37732', got %q", aaNote)
		}
		aaDiffDays1, _ := f.GetCellValue(sheet, "N6")
		if aaDiffDays1 != "+12" {
			t.Errorf("expected AA diff days '+12', got %q", aaDiffDays1)
		}
		aaDiffDays2, _ := f.GetCellValue(sheet, "N7")
		if aaDiffDays2 != "-33" {
			t.Errorf("expected AA diff days '-33', got %q", aaDiffDays2)
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
