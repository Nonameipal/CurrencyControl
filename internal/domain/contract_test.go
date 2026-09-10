package domain_test

import (
	"testing"
	"time"

	"CurrencyControl/internal/domain"
)

func TestContractModelFields(t *testing.T) {
	now := time.Now()
	retDate := now.AddDate(0, 3, 0)
	c := domain.Contract{
		ID:             1,
		ContractNumber: "CNT-001",
		ContractDate:   now,
		Subject:        "Поставка оборудования и пусконаладочные работы",
		DeliveryDate:   now.AddDate(0, 1, 0),
		ReturnTermDays: 90,
		ReturnDate:     &retDate,
		TotalAmount:    150000.00,
		ContractCurrency: "USD",
		CreatedBy:      "operator_tj",
	}

	if c.ID != 1 || c.ContractNumber != "CNT-001" {
		t.Fatalf("unexpected contract fields: %+v", c)
	}
	if c.ReturnDate == nil || !c.ReturnDate.Equal(retDate) {
		t.Fatalf("expected return date %v, got %v", retDate, c.ReturnDate)
	}
}

func TestGTDDocumentTypes(t *testing.T) {
	gtd := domain.GTD{
		DocumentType: domain.DocumentTypeGTD,
		GTDNumber:    "01001/010126/0012345",
	}
	if gtd.DocumentType != "gtd" {
		t.Fatalf("expected gtd, got %s", gtd.DocumentType)
	}

	act := domain.GTD{
		DocumentType: domain.DocumentTypeAct,
		GTDNumber:    "АКТ-78",
	}
	if act.DocumentType != "act" {
		t.Fatalf("expected act, got %s", act.DocumentType)
	}
}

func TestAdditionalAgreementDocTypes(t *testing.T) {
	ag := domain.AdditionalAgreement{
		DocType: domain.DocTypeAdditionalAgreement,
	}
	if ag.DocType != "additional_agreement" {
		t.Fatalf("expected additional_agreement, got %s", ag.DocType)
	}

	spec := domain.AdditionalAgreement{
		DocType: domain.DocTypeSpecification,
	}
	if spec.DocType != "specification" {
		t.Fatalf("expected specification, got %s", spec.DocType)
	}

	app := domain.AdditionalAgreement{
		DocType: domain.DocTypeAppendix,
	}
	if app.DocType != "appendix" {
		t.Fatalf("expected appendix, got %s", app.DocType)
	}
}
