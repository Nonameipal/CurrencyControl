package domain_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"CurrencyControl/internal/domain"
)

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, domain.CtxKeyLogin, "testuser")
	ctx = context.WithValue(ctx, domain.CtxKeyFirstName, "Иван")
	ctx = context.WithValue(ctx, domain.CtxKeyLastName, "Иванов")
	ctx = context.WithValue(ctx, domain.CtxKeyEmail, "ivan@example.com")

	if login := domain.GetLoginFromCtx(ctx); login != "testuser" {
		t.Errorf("GetLoginFromCtx() = %q, expected %q", login, "testuser")
	}

	ub := domain.GetUserBriefFromCtx(ctx)
	if ub.Login != "testuser" || ub.FirstName != "Иван" || ub.LastName != "Иванов" || ub.Email != "ivan@example.com" {
		t.Errorf("GetUserBriefFromCtx() = %+v, unexpected fields", ub)
	}
}

func TestOptionB_UserBrief_Serialization(t *testing.T) {
	creator := &domain.UserBrief{
		Login:     "operator1",
		FirstName: "Алишер",
		LastName:  "Каримов",
		Email:     "alisher@bank.tj",
	}
	updater := &domain.UserBrief{
		Login:     "compliance1",
		FirstName: "Сардор",
		LastName:  "Рахимов",
		Email:     "sardor@bank.tj",
	}

	// 1. Contract
	c := domain.Contract{
		ID:             1,
		ContractNumber: "CNT-001",
		CreatedBy:      "operator1",
		Creator:        creator,
		UpdatedBy:      "compliance1",
		Updater:        updater,
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal Contract failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"created_by":"operator1"`) {
		t.Errorf("Contract json missing created_by string: %s", s)
	}
	if !strings.Contains(s, `"creator":{"login":"operator1","first_name":"Алишер","last_name":"Каримов","email":"alisher@bank.tj"}`) {
		t.Errorf("Contract json missing creator UserBrief object: %s", s)
	}
	if !strings.Contains(s, `"updated_by":"compliance1"`) {
		t.Errorf("Contract json missing updated_by string: %s", s)
	}
	if !strings.Contains(s, `"updater":{"login":"compliance1","first_name":"Сардор","last_name":"Рахимов","email":"sardor@bank.tj"}`) {
		t.Errorf("Contract json missing updater UserBrief object: %s", s)
	}

	// 2. Invoice
	inv := domain.Invoice{
		ID:            2,
		InvoiceNumber: "INV-001",
		CreatedBy:     "operator1",
		Creator:       creator,
	}
	b, err = json.Marshal(inv)
	if err != nil {
		t.Fatalf("Marshal Invoice failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"created_by":"operator1"`) || !strings.Contains(s, `"creator":{`) {
		t.Errorf("Invoice json format error: %s", s)
	}

	// 3. GTD
	gtd := domain.GTD{
		ID:        3,
		GTDNumber: "GTD-001",
		CreatedBy: "operator1",
		Creator:   creator,
	}
	b, err = json.Marshal(gtd)
	if err != nil {
		t.Fatalf("Marshal GTD failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"created_by":"operator1"`) || !strings.Contains(s, `"creator":{`) {
		t.Errorf("GTD json format error: %s", s)
	}

	// 4. Additional Agreement
	aa := domain.AdditionalAgreement{
		ID:        4,
		CreatedBy: "operator1",
		Creator:   creator,
	}
	b, err = json.Marshal(aa)
	if err != nil {
		t.Fatalf("Marshal AdditionalAgreement failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"created_by":"operator1"`) || !strings.Contains(s, `"creator":{`) {
		t.Errorf("AdditionalAgreement json format error: %s", s)
	}

	// 5. PaymentOrder
	po := domain.PaymentOrder{
		ID:        5,
		CreatedBy: "operator1",
		Creator:   creator,
	}
	b, err = json.Marshal(po)
	if err != nil {
		t.Fatalf("Marshal PaymentOrder failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"created_by":"operator1"`) || !strings.Contains(s, `"creator":{`) {
		t.Errorf("PaymentOrder json format error: %s", s)
	}

	// 6. Counterparty
	cp := domain.Counterparty{
		ID:        6,
		CreatedBy: "operator1",
		Creator:   creator,
	}
	b, err = json.Marshal(cp)
	if err != nil {
		t.Fatalf("Marshal Counterparty failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"created_by":"operator1"`) || !strings.Contains(s, `"creator":{`) {
		t.Errorf("Counterparty json format error: %s", s)
	}

	// 7. Branch
	br := domain.Branch{
		ID:        7,
		Name:      "Душанбе-Центр",
		CreatedBy: "admin",
		Creator:   creator,
	}
	b, err = json.Marshal(br)
	if err != nil {
		t.Fatalf("Marshal Branch failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"created_by":"admin"`) || !strings.Contains(s, `"creator":{`) {
		t.Errorf("Branch json format error: %s", s)
	}

	// 8. AuditLog
	al := domain.AuditLog{
		ID:        8,
		UserLogin: "operator1",
		User:      creator,
		CreatedAt: time.Now(),
	}
	b, err = json.Marshal(al)
	if err != nil {
		t.Fatalf("Marshal AuditLog failed: %v", err)
	}
	s = string(b)
	if !strings.Contains(s, `"user_login":"operator1"`) || !strings.Contains(s, `"user":{`) {
		t.Errorf("AuditLog json format error: %s", s)
	}
}
