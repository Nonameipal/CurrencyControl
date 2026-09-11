package domain_test

import (
	"testing"
	"time"

	"CurrencyControl/internal/domain"
)

func TestInvoiceModelFields(t *testing.T) {
	now := time.Now()
	docPath := "uploads/invoices/inv1.pdf"
	docName := "inv1.pdf"

	inv := domain.Invoice{
		ID:                   10,
		ContractID:           100,
		InvoiceNumber:        "INV-2026-001",
		InvoiceName:          "Оплата за поставку оборудования",
		InvoiceDate:          now,
		Amount:               50000.00,
		Currency:             "USD",
		HSCode:               "8471300000", // ТН ВЭД (HS CODE)
		DeductAmount:         50000.00,
		DocumentPath:         &docPath,
		OriginalDocumentName: &docName,
		CreatedBy:            "operator1",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if inv.InvoiceNumber != "INV-2026-001" {
		t.Fatalf("expected invoice number INV-2026-001, got %s", inv.InvoiceNumber)
	}
	if inv.HSCode != "8471300000" {
		t.Fatalf("expected HS Code 8471300000, got %s", inv.HSCode)
	}
	if inv.Amount != 50000.00 || inv.Currency != "USD" {
		t.Fatalf("unexpected amount or currency")
	}
	if inv.CreatedBy != "operator1" {
		t.Fatalf("expected created_by operator1, got %s", inv.CreatedBy)
	}
	if inv.CreatedAt.IsZero() || inv.UpdatedAt.IsZero() {
		t.Fatalf("expected non-zero created_at and updated_at")
	}
}
