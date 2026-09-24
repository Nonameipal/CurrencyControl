package service

import (
	"context"
	"strings"
	"testing"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

// ───────────────────────────── Contract mock ─────────────────────────────

type mockContractRepoForLock struct {
	contract domain.Contract
}

func (m *mockContractRepoForLock) Create(ctx context.Context, c domain.Contract) (domain.Contract, error) {
	return c, nil
}
func (m *mockContractRepoForLock) GetByID(ctx context.Context, id int64) (domain.Contract, error) {
	return m.contract, nil
}
func (m *mockContractRepoForLock) GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	return nil, nil
}
func (m *mockContractRepoForLock) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
	return nil, 0, nil
}
func (m *mockContractRepoForLock) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	return nil, nil
}
func (m *mockContractRepoForLock) RestoreContract(ctx context.Context, id int64) error { return nil }
func (m *mockContractRepoForLock) SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	return nil, nil
}
func (m *mockContractRepoForLock) CheckCountry(ctx context.Context, name string) (bool, error) {
	return true, nil
}
func (m *mockContractRepoForLock) CheckCurrency(ctx context.Context, code string) (bool, error) {
	return true, nil
}
func (m *mockContractRepoForLock) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
	return nil, nil
}
func (m *mockContractRepoForLock) Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error) {
	return c, nil
}
func (m *mockContractRepoForLock) SoftDelete(ctx context.Context, id int64) error { return nil }
func (m *mockContractRepoForLock) ResetApprovalStatus(ctx context.Context, id int64) error {
	return nil
}

func TestContract_UnapprovedLocks(t *testing.T) {
	repo := &mockContractRepoForLock{
		contract: domain.Contract{
			ID:             1,
			ApprovalStatus: domain.ApprovalStatusPendingCurrencyControl,
		},
	}
	svc := NewContractService(repo)

	// pending_currency_control → редактирование запрещено
	_, err := svc.Update(context.Background(), 1, domain.Contract{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "валютного контроля") {
		t.Errorf("expected update to fail on pending_currency_control contract, got %v", err)
	}

	// pending_currency_control → удаление запрещено
	err = svc.SoftDelete(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "валютного контроля") {
		t.Errorf("expected soft delete to fail on pending_currency_control contract, got %v", err)
	}

	// pending_compliance → редактирование запрещено
	repo.contract.ApprovalStatus = domain.ApprovalStatusPendingCompliance
	_, err = svc.Update(context.Background(), 1, domain.Contract{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "комплаенс") {
		t.Errorf("expected update to fail on pending_compliance contract, got %v", err)
	}

	// revision_required, wrong creator → запрещено
	repo.contract.ApprovalStatus = domain.ApprovalStatusRevisionRequired
	repo.contract.CreatedBy = "alice"
	_, err = svc.Update(context.Background(), 1, domain.Contract{ID: 1, UpdatedBy: "bob"})
	if err == nil || !strings.Contains(err.Error(), "создатель") {
		t.Errorf("expected update to fail when non-creator edits revision_required contract, got %v", err)
	}

	// revision_required, correct creator → разрешено
	_, err = svc.Update(context.Background(), 1, domain.Contract{ID: 1, CreatedBy: "alice", UpdatedBy: "alice"})
	if err != nil {
		t.Errorf("expected update to succeed for creator on revision_required contract, got %v", err)
	}

	// rejected → запрещено
	repo.contract.ApprovalStatus = domain.ApprovalStatusRejectedCurrencyControl
	_, err = svc.Update(context.Background(), 1, domain.Contract{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "корзину") {
		t.Errorf("expected update to fail on rejected contract, got %v", err)
	}
	err = svc.SoftDelete(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "корзине") {
		t.Errorf("expected soft delete to fail on rejected contract, got %v", err)
	}

	// approved → разрешено
	repo.contract.ApprovalStatus = domain.ApprovalStatusApproved
	_, err = svc.Update(context.Background(), 1, domain.Contract{ID: 1})
	if err != nil {
		t.Errorf("expected update to succeed on approved contract, got %v", err)
	}
	err = svc.SoftDelete(context.Background(), 1)
	if err != nil {
		t.Errorf("expected soft delete to succeed on approved contract, got %v", err)
	}
}

// ───────────────────────────── Invoice mock ──────────────────────────────

type mockInvoiceRepoForLock struct {
	inv domain.Invoice
}

func (m *mockInvoiceRepoForLock) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	return nil, nil
}
func (m *mockInvoiceRepoForLock) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.InvoiceWithDetails, error) {
	return nil, nil
}
func (m *mockInvoiceRepoForLock) Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error) {
	return inv, nil
}
func (m *mockInvoiceRepoForLock) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	return m.inv, nil
}
func (m *mockInvoiceRepoForLock) Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error) {
	return inv, nil
}
func (m *mockInvoiceRepoForLock) SoftDelete(ctx context.Context, id int64) error { return nil }
func (m *mockInvoiceRepoForLock) ResetApprovalStatus(ctx context.Context, id int64) error {
	return nil
}

func TestInvoice_UnapprovedLocks(t *testing.T) {
	repo := &mockInvoiceRepoForLock{
		inv: domain.Invoice{
			ID:             1,
			ApprovalStatus: domain.ApprovalStatusPendingCompliance,
		},
	}
	svc := NewInvoiceService(repo)

	// pending_compliance → редактирование запрещено
	_, err := svc.Update(context.Background(), 1, domain.Invoice{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "комплаенс") {
		t.Errorf("expected update to fail on pending invoice, got %v", err)
	}

	// pending_compliance → удаление запрещено
	err = svc.SoftDelete(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "комплаенс") {
		t.Errorf("expected soft delete to fail on pending invoice, got %v", err)
	}

	// approved → разрешено
	repo.inv.ApprovalStatus = domain.ApprovalStatusApproved
	_, err = svc.Update(context.Background(), 1, domain.Invoice{ID: 1})
	if err != nil {
		t.Errorf("expected update to succeed on approved invoice, got %v", err)
	}
	err = svc.SoftDelete(context.Background(), 1)
	if err != nil {
		t.Errorf("expected soft delete to succeed on approved invoice, got %v", err)
	}
}

// ───────────────────────────── GTD mock ─────────────────────────────────

type mockGTDRepoForLock struct {
	gtd domain.GTD
}

func (m *mockGTDRepoForLock) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	return g, nil
}
func (m *mockGTDRepoForLock) GetByID(ctx context.Context, id int64) (*domain.GTD, error) {
	return &m.gtd, nil
}
func (m *mockGTDRepoForLock) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	return &m.gtd, nil
}
func (m *mockGTDRepoForLock) GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error) {
	return nil, nil
}
func (m *mockGTDRepoForLock) GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error) {
	return nil, nil
}
func (m *mockGTDRepoForLock) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.GTD, error) {
	return nil, nil
}
func (m *mockGTDRepoForLock) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	return g, nil
}
func (m *mockGTDRepoForLock) SoftDelete(ctx context.Context, id int64) error { return nil }
func (m *mockGTDRepoForLock) ResetApprovalStatus(ctx context.Context, id int64) error { return nil }

func TestGTD_UnapprovedLocks(t *testing.T) {
	repo := &mockGTDRepoForLock{
		gtd: domain.GTD{
			ID:             1,
			ApprovalStatus: domain.ApprovalStatusPendingCurrencyControl,
		},
	}
	svc := NewGTDService(repo)

	// pending_currency_control → редактирование запрещено
	_, err := svc.Update(context.Background(), 1, domain.GTD{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "валютного контроля") {
		t.Errorf("expected update to fail on pending GTD, got %v", err)
	}

	// pending_currency_control → удаление запрещено
	err = svc.SoftDelete(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "валютного контроля") {
		t.Errorf("expected soft delete to fail on pending GTD, got %v", err)
	}

	// approved → разрешено
	repo.gtd.ApprovalStatus = domain.ApprovalStatusApproved
	_, err = svc.Update(context.Background(), 1, domain.GTD{ID: 1})
	if err != nil {
		t.Errorf("expected update to succeed on approved GTD, got %v", err)
	}
	err = svc.SoftDelete(context.Background(), 1)
	if err != nil {
		t.Errorf("expected soft delete to succeed on approved GTD, got %v", err)
	}
}

// ───────────────────────── AdditionalAgreement mock ─────────────────────

type mockAddlRepoForLock struct {
	ag domain.AdditionalAgreement
}

func (m *mockAddlRepoForLock) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	return ag, nil
}
func (m *mockAddlRepoForLock) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	return nil, nil
}
func (m *mockAddlRepoForLock) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	return m.ag, nil
}
func (m *mockAddlRepoForLock) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	return ag, nil
}
func (m *mockAddlRepoForLock) SoftDelete(ctx context.Context, id int64) error { return nil }
func (m *mockAddlRepoForLock) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	return nil
}
func (m *mockAddlRepoForLock) ResetApprovalStatus(ctx context.Context, id int64) error { return nil }

func TestAdditionalAgreement_UnapprovedLocks(t *testing.T) {
	repo := &mockAddlRepoForLock{
		ag: domain.AdditionalAgreement{
			ID:             1,
			ApprovalStatus: domain.ApprovalStatusPendingCompliance,
		},
	}
	svc := NewAdditionalAgreementService(repo)

	// pending_compliance → редактирование запрещено
	_, err := svc.Update(context.Background(), 1, domain.AdditionalAgreement{ID: 1})
	if err == nil || !strings.Contains(err.Error(), "комплаенс") {
		t.Errorf("expected update to fail on pending additional agreement, got %v", err)
	}

	// pending_compliance → удаление запрещено
	err = svc.SoftDelete(context.Background(), 1)
	if err == nil || !strings.Contains(err.Error(), "комплаенс") {
		t.Errorf("expected soft delete to fail on pending additional agreement, got %v", err)
	}

	// approved → разрешено
	repo.ag.ApprovalStatus = domain.ApprovalStatusApproved
	_, err = svc.Update(context.Background(), 1, domain.AdditionalAgreement{ID: 1})
	if err != nil {
		t.Errorf("expected update to succeed on approved additional agreement, got %v", err)
	}
	err = svc.SoftDelete(context.Background(), 1)
	if err != nil {
		t.Errorf("expected soft delete to succeed on approved additional agreement, got %v", err)
	}
}
