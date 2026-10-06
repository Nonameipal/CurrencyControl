package domain_test

import (
	"testing"
	"time"

	"CurrencyControl/internal/domain"
)

func TestFormatRussianDays(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{1, "1 день"},
		{2, "2 дня"},
		{3, "3 дня"},
		{4, "4 дня"},
		{5, "5 дней"},
		{11, "11 дней"},
		{12, "12 дней"},
		{14, "14 дней"},
		{19, "19 дней"},
		{20, "20 дней"},
		{21, "21 день"},
		{22, "22 дня"},
		{25, "25 дней"},
		{-5, "5 дней"},
		{-1, "1 день"},
	}

	for _, tt := range tests {
		got := domain.FormatRussianDays(tt.n)
		if got != tt.expected {
			t.Errorf("FormatRussianDays(%d) = %q, expected %q", tt.n, got, tt.expected)
		}
	}
}

func TestCalculateDeliveryComparison(t *testing.T) {
	// Case 1: Early delivery (5 days early)
	deadline1 := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	actual1 := time.Date(2026, 10, 10, 14, 30, 0, 0, time.UTC)
	diffDays, status, notice := domain.CalculateDeliveryComparison(domain.DocumentTypeGTD, actual1, &deadline1)

	if diffDays != -5 {
		t.Errorf("expected diffDays -5, got %d", diffDays)
	}
	if status != "early" {
		t.Errorf("expected status 'early', got %s", status)
	}
	expectedNotice1 := "ГТД предоставлена на 5 дней раньше установленного срока поставки (план: 15.10.2026, факт: 10.10.2026)"
	if notice != expectedNotice1 {
		t.Errorf("expected notice %q, got %q", expectedNotice1, notice)
	}

	// Case 2: On-time delivery (exact date)
	deadline2 := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	actual2 := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	diffDays2, status2, notice2 := domain.CalculateDeliveryComparison(domain.DocumentTypeGTD, actual2, &deadline2)

	if diffDays2 != 0 {
		t.Errorf("expected diffDays 0, got %d", diffDays2)
	}
	if status2 != "on_time" {
		t.Errorf("expected status 'on_time', got %s", status2)
	}
	expectedNotice2 := "ГТД предоставлена точно в установленный срок поставки (15.10.2026)"
	if notice2 != expectedNotice2 {
		t.Errorf("expected notice %q, got %q", expectedNotice2, notice2)
	}

	// Case 3: Overdue delivery (3 days late)
	deadline3 := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	actual3 := time.Date(2026, 10, 18, 16, 45, 0, 0, time.UTC)
	diffDays3, status3, notice3 := domain.CalculateDeliveryComparison(domain.DocumentTypeGTD, actual3, &deadline3)

	if diffDays3 != 3 {
		t.Errorf("expected diffDays 3, got %d", diffDays3)
	}
	if status3 != "overdue" {
		t.Errorf("expected status 'overdue', got %s", status3)
	}
	expectedNotice3 := "Внимание! ГТД предоставлена на 3 дня позже установленного срока поставки (план: 15.10.2026, факт: 18.10.2026). Просрочка: 3 дня"
	if notice3 != expectedNotice3 {
		t.Errorf("expected notice %q, got %q", expectedNotice3, notice3)
	}

	// Case 4: Document type Act (акт выполненных работ)
	deadline4 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	actual4 := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	diffDays4, status4, notice4 := domain.CalculateDeliveryComparison(domain.DocumentTypeAct, actual4, &deadline4)

	if diffDays4 != -2 {
		t.Errorf("expected diffDays -2, got %d", diffDays4)
	}
	if status4 != "early" {
		t.Errorf("expected status 'early', got %s", status4)
	}
	expectedNotice4 := "Акт выполненных работ предоставлен на 2 дня раньше установленного срока предоставления услуг (план: 01.11.2026, факт: 30.10.2026)"
	if notice4 != expectedNotice4 {
		t.Errorf("expected notice %q, got %q", expectedNotice4, notice4)
	}

	// Case 5: No deadline specified
	actual5 := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	diffDays5, status5, notice5 := domain.CalculateDeliveryComparison(domain.DocumentTypeGTD, actual5, nil)

	if diffDays5 != 0 {
		t.Errorf("expected diffDays 0, got %d", diffDays5)
	}
	if status5 != "unknown" {
		t.Errorf("expected status 'unknown', got %s", status5)
	}
	if notice5 == "" {
		t.Errorf("expected non-empty notice for unknown deadline")
	}
}

func TestGTDModelFields(t *testing.T) {
	now := time.Now()
	currency := "USD"
	docPath := "uploads/gtd/gtd_001.pdf"

	gtd := domain.GTD{
		ID:                 1,
		ContractID:         10,
		InvoiceID:          20,
		DocumentType:       domain.DocumentTypeGTD,
		GTDNumber:          "10001010/110926/0012345",
		GTDDate:            &now,
		GTDAmount:          75000.50,
		GTDCurrency:        &currency,
		ClosesAmount:       75000.50,
		HSCode:             "8471300000",
		DestinationCountry: "Таджикистан",
		InvoiceNumber:      "INV-2026-001",
		DocumentPath:       &docPath,
		SubmissionDate:     &now,
		DeliveryDeadline:   &now,
		DaysDifference:     0,
		DeliveryStatus:     "on_time",
		DeliveryNotice:     "ГТД предоставлена точно в установленный срок поставки",
		CreatedBy:          "operator_tj",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if gtd.GTDNumber != "10001010/110926/0012345" {
		t.Fatalf("expected GTDNumber '10001010/110926/0012345', got %s", gtd.GTDNumber)
	}
	if gtd.DaysDifference != 0 || gtd.DeliveryStatus != "on_time" {
		t.Fatalf("expected 0 days diff and on_time status")
	}
}

func TestDefaultHSCode(t *testing.T) {
	if domain.DefaultHSCode != "Нет кода" {
		t.Fatalf("expected DefaultHSCode 'Нет кода', got %s", domain.DefaultHSCode)
	}
}

func TestAdditionalAgreementRemainingAmount(t *testing.T) {
	amount := 50000.0
	currency := "EUR"
	ag := domain.AdditionalAgreement{
		ContractID:      1,
		ForeignAmount:   &amount,
		ForeignCurrency: &currency,
	}

	ag.Normalize()

	if ag.RemainingAmount != 50000.0 {
		t.Errorf("expected RemainingAmount to be 50000.0, got %f", ag.RemainingAmount)
	}
	if ag.Amount == nil || *ag.Amount != 50000.0 {
		t.Errorf("expected Amount to be 50000.0, got %v", ag.Amount)
	}
	if ag.Currency == nil || *ag.Currency != "EUR" {
		t.Errorf("expected Currency to be 'EUR', got %v", ag.Currency)
	}
}

func TestInvoiceWithAdditionalAgreementID(t *testing.T) {
	addlID := int64(42)
	inv := domain.Invoice{
		ContractID:            10,
		AdditionalAgreementID: &addlID,
		InvoiceNumber:         "INV-EUR-01",
		InvoiceDate:           time.Now(),
		Amount:                12000.0,
		Currency:              "EUR",
	}

	if inv.AdditionalAgreementID == nil || *inv.AdditionalAgreementID != 42 {
		t.Errorf("expected AdditionalAgreementID 42, got %v", inv.AdditionalAgreementID)
	}
	if inv.Currency != "EUR" {
		t.Errorf("expected Currency 'EUR', got %s", inv.Currency)
	}
}

func TestGTDWithAdditionalAgreementID(t *testing.T) {
	addlID := int64(42)
	curr := "EUR"
	gtd := domain.GTD{
		ContractID:            10,
		AdditionalAgreementID: &addlID,
		InvoiceID:             5,
		GTDNumber:             "GTD-001",
		GTDCurrency:           &curr,
		ClosesAmount:          12000.0,
	}

	if gtd.AdditionalAgreementID == nil || *gtd.AdditionalAgreementID != 42 {
		t.Errorf("expected AdditionalAgreementID 42, got %v", gtd.AdditionalAgreementID)
	}
	if *gtd.GTDCurrency != "EUR" {
		t.Errorf("expected GTDCurrency 'EUR', got %s", *gtd.GTDCurrency)
	}
}
