package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/infrostucture/database"
	"CurrencyControl/internal/repository"
	"CurrencyControl/internal/service/ports"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getTestDB(t *testing.T) *gorm.DB {
	_ = godotenv.Load("../../.env")
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DATABASE")
	if dbname == "" {
		dbname = "currency_control"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC", host, port, user, pass, dbname)
	rawDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Postgres is not reachable: %v", err)
		return nil
	}
	sqlDB, err := rawDB.DB()
	if err != nil || sqlDB.Ping() != nil {
		t.Skipf("Postgres ping failed: %v", err)
		return nil
	}

	db, err := database.InitGormDB(dsn)
	if err != nil {
		t.Fatalf("InitGormDB failed: %v", err)
	}
	return db
}

func TestDatabaseOperations_EndToEnd(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	contractRepo := repository.NewContractRepository(db)
	aaRepo := repository.NewAdditionalAgreementRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	gtdRepo := repository.NewGTDRepository(db)
	poRepo := repository.NewPaymentOrderRepository(db)
	trashRepo := repository.NewTrashRepository(db)
	reportRepo := repository.NewReportRepository(db)
	approvalRepo := repository.NewApprovalRepository(db)

	now := time.Now().UTC()
	contractDate := now.AddDate(0, -1, 0)
	deliveryDate := now.AddDate(0, 3, 0)
	endDate := now.AddDate(0, 6, 0)

	contractNum := fmt.Sprintf("TEST-CNT-%d", now.UnixNano())
	createdContract, err := contractRepo.Create(ctx, domain.Contract{
		ContractNumber:   contractNum,
		ContractDate:     contractDate,
		DeliveryDate:     deliveryDate,
		ContractEndDate:  &endDate,
		TotalAmount:      10000,
		ContractCurrency: "USD",
		Subject:          "Тестовый контракт",
		Status:           domain.ContractStatusActive,
	})
	if err != nil {
		t.Fatalf("Create contract failed: %v", err)
	}
	defer func() {
		_ = db.Unscoped().Where("contract_id = ?", createdContract.ID).Delete(&domain.PaymentOrder{}).Error
		_ = db.Unscoped().Where("contract_id = ?", createdContract.ID).Delete(&domain.GTD{}).Error
		_ = db.Unscoped().Where("contract_id = ?", createdContract.ID).Delete(&domain.Invoice{}).Error
		_ = db.Unscoped().Where("contract_id = ?", createdContract.ID).Delete(&domain.AdditionalAgreement{}).Error
		_ = db.Unscoped().Where("id = ?", createdContract.ID).Delete(&domain.Contract{}).Error
	}()

	fetchedContract, err := contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract failed: %v", err)
	}
	if fetchedContract.TotalAmount != 10000 || fetchedContract.RemainingAmount != 10000 {
		t.Fatalf("expected total 10000 and rem 10000, got total=%.2f rem=%.2f", fetchedContract.TotalAmount, fetchedContract.RemainingAmount)
	}
	if fetchedContract.ApprovalStatus != domain.ApprovalStatusApproved {
		t.Fatalf("expected contract to be approved immediately, got %s", fetchedContract.ApprovalStatus)
	}

	usdAmount := 2500.0
	usdCurrency := "USD"
	aaNumUSD := fmt.Sprintf("AA-USD-%d", now.UnixNano())
	createdAAUSD, err := aaRepo.Create(ctx, domain.AdditionalAgreement{
		ContractID:      createdContract.ID,
		AgreementNumber: &aaNumUSD,
		AgreementDate:   &contractDate,
		ForeignAmount:   &usdAmount,
		ForeignCurrency: &usdCurrency,
		Status:          domain.AgreementStatusActive,
		ApprovalStatus:  domain.ApprovalStatusApproved,
	})
	if err != nil {
		t.Fatalf("Create AA (USD) failed: %v", err)
	}

	fetchedContract, err = contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract after USD AA failed: %v", err)
	}
	if fetchedContract.TotalAmount != 12500 || fetchedContract.RemainingAmount != 12500 {
		t.Fatalf("expected total 12500, got total=%.2f rem=%.2f", fetchedContract.TotalAmount, fetchedContract.RemainingAmount)
	}

	eurAmount := 5000.0
	eurCurrency := "EUR"
	aaNumEUR := fmt.Sprintf("AA-EUR-%d", now.UnixNano())
	createdAAEUR, err := aaRepo.Create(ctx, domain.AdditionalAgreement{
		ContractID:      createdContract.ID,
		AgreementNumber: &aaNumEUR,
		AgreementDate:   &contractDate,
		ForeignAmount:   &eurAmount,
		ForeignCurrency: &eurCurrency,
		Status:          domain.AgreementStatusActive,
		ApprovalStatus:  domain.ApprovalStatusApproved,
	})
	if err != nil {
		t.Fatalf("Create AA (EUR) failed: %v", err)
	}

	fetchedContract, err = contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract after EUR AA failed: %v", err)
	}
	if len(fetchedContract.AdditionalCurrencies) == 0 {
		t.Fatalf("expected EUR in AdditionalCurrencies, but list is empty")
	}
	if fetchedContract.AdditionalCurrencies[0].Currency != "EUR" || fetchedContract.AdditionalCurrencies[0].RemainingAmount != 5000 {
		t.Fatalf("expected EUR with 5000 remaining, got %+v", fetchedContract.AdditionalCurrencies[0])
	}

	invNumRUB := fmt.Sprintf("INV-RUB-%d", now.UnixNano())
	_, err = invoiceRepo.Create(ctx, domain.Invoice{
		ContractID:    createdContract.ID,
		InvoiceNumber: invNumRUB,
		InvoiceDate:   now,
		Amount:        1000,
		Currency:      "RUB",
	}, "USD")
	if err == nil {
		t.Fatalf("expected error creating invoice in unconfigured currency RUB, but got nil")
	}

	invNumEUR := fmt.Sprintf("INV-EUR-%d", now.UnixNano())
	createdInvEUR, err := invoiceRepo.Create(ctx, domain.Invoice{
		ContractID:    createdContract.ID,
		InvoiceNumber: invNumEUR,
		InvoiceDate:   now,
		Amount:        2000,
		Currency:      "EUR",
		ApprovalStatus: domain.ApprovalStatusApproved,
	}, "USD")
	if err != nil {
		t.Fatalf("Create EUR invoice failed: %v", err)
	}

	fetchedContract, err = contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract after EUR invoice failed: %v", err)
	}
	if len(fetchedContract.AdditionalCurrencies) == 0 || fetchedContract.AdditionalCurrencies[0].RemainingAmount != 3000 {
		t.Fatalf("expected EUR remaining 3000, got %+v", fetchedContract.AdditionalCurrencies)
	}

	invNumUSD := fmt.Sprintf("INV-USD-%d", now.UnixNano())
	createdInvUSD, err := invoiceRepo.Create(ctx, domain.Invoice{
		ContractID:    createdContract.ID,
		InvoiceNumber: invNumUSD,
		InvoiceDate:   now,
		Amount:        4000,
		Currency:      "USD",
		ApprovalStatus: domain.ApprovalStatusApproved,
	}, "USD")
	if err != nil {
		t.Fatalf("Create USD invoice failed: %v", err)
	}

	fetchedContract, err = contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract after USD invoice failed: %v", err)
	}
	if fetchedContract.RemainingAmount != 8500 {
		t.Fatalf("expected contract USD remaining 8500, got %.2f", fetchedContract.RemainingAmount)
	}

	poNum := fmt.Sprintf("PO-%d", now.UnixNano())
	createdPO, err := poRepo.Create(ctx, domain.PaymentOrder{
		InvoiceID:          createdInvUSD.ID,
		OperationDate:      now,
		PaymentOrderNumber: poNum,
		Amount:             2500,
		Currency:           "USD",
		Payer:              "Test Payer",
		ReceiverName:       "Test Receiver",
		ReceiverBank:       "Test Bank",
		PaymentPurpose:     "Оплата",
		ReceiverCountry:    "USA",
		SenderName:         "Sender",
		SenderBank:         "Bank",
		SenderCountry:      "TJ",
		ValueDate:          now,
		CreatedBy:          "test_user",
	})
	if err != nil {
		t.Fatalf("Create PaymentOrder failed: %v", err)
	}
	if createdPO.ApprovalStatus != domain.ApprovalStatusPendingCurrencyControl {
		t.Fatalf("expected PO to be pending_currency_control, got %s", createdPO.ApprovalStatus)
	}

	_, err = approvalRepo.SetCurrencyControlDecision(ctx, "payment_order", createdPO.ID, "accepted", "ok", "controller1")
	if err != nil {
		t.Fatalf("SetCurrencyControlDecision PO failed: %v", err)
	}
	_, err = approvalRepo.SetComplianceDecision(ctx, "payment_order", createdPO.ID, "approve", "ok", "compliance1")
	if err != nil {
		t.Fatalf("SetComplianceDecision PO failed: %v", err)
	}

	gtdNum := fmt.Sprintf("GTD-%d", now.UnixNano())
	gtdCurr := "USD"
	createdGTD, err := gtdRepo.Create(ctx, domain.GTD{
		ContractID:     createdContract.ID,
		InvoiceID:      createdInvUSD.ID,
		GTDNumber:      gtdNum,
		GTDCurrency:    &gtdCurr,
		GTDDate:        &now,
		GTDAmount:      2500,
		ClosesAmount:   2500,
		ApprovalStatus: domain.ApprovalStatusApproved,
	})
	if err != nil {
		t.Fatalf("Create GTD failed: %v", err)
	}

	invoicesWithDetails, err := invoiceRepo.GetByContractID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByContractID invoices failed: %v", err)
	}
	if len(invoicesWithDetails) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(invoicesWithDetails))
	}

	pos, err := poRepo.GetByContractID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByContractID POs failed: %v", err)
	}
	if len(pos) != 1 || pos[0].ID != createdPO.ID {
		t.Fatalf("expected 1 PO, got %d", len(pos))
	}

	gtds, err := gtdRepo.GetByContractID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByContractID GTDs failed: %v", err)
	}
	if len(gtds) != 1 || gtds[0].ID != createdGTD.ID {
		t.Fatalf("expected 1 GTD, got %d", len(gtds))
	}

	_ = createdAAUSD
	_ = createdAAEUR

	err = poRepo.SoftDelete(ctx, createdPO.ID)
	if err != nil {
		t.Fatalf("SoftDelete PO failed: %v", err)
	}

	trashItems, _, err := trashRepo.GetTrashItems(ctx, dto.TrashFilter{EntityType: "payment_order"})
	if err != nil {
		t.Fatalf("GetTrashItems PO failed: %v", err)
	}
	foundInTrash := false
	for _, it := range trashItems {
		if it.ID == createdPO.ID {
			foundInTrash = true
			break
		}
	}
	if !foundInTrash {
		t.Fatalf("deleted PO not found in trash")
	}

	err = trashRepo.RestoreItem(ctx, "payment_order", createdPO.ID)
	if err != nil {
		t.Fatalf("RestoreItem PO failed: %v", err)
	}

	err = invoiceRepo.SoftDelete(ctx, createdInvEUR.ID)
	if err != nil {
		t.Fatalf("SoftDelete EUR invoice failed: %v", err)
	}

	fetchedContract, err = contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract after deleting EUR invoice failed: %v", err)
	}
	if len(fetchedContract.AdditionalCurrencies) == 0 || fetchedContract.AdditionalCurrencies[0].RemainingAmount != 5000 {
		t.Fatalf("expected EUR remaining restored to 5000, got %+v", fetchedContract.AdditionalCurrencies)
	}

	err = trashRepo.RestoreItem(ctx, "invoice", createdInvEUR.ID)
	if err != nil {
		t.Fatalf("RestoreItem EUR invoice failed: %v", err)
	}

	fetchedContract, err = contractRepo.GetByID(ctx, createdContract.ID)
	if err != nil {
		t.Fatalf("GetByID contract after restoring EUR invoice failed: %v", err)
	}
	if len(fetchedContract.AdditionalCurrencies) == 0 || fetchedContract.AdditionalCurrencies[0].RemainingAmount != 3000 {
		t.Fatalf("expected EUR remaining back to 3000, got %+v", fetchedContract.AdditionalCurrencies)
	}

	err = gtdRepo.SoftDelete(ctx, createdGTD.ID)
	if err != nil {
		t.Fatalf("SoftDelete GTD failed: %v", err)
	}
	err = trashRepo.RestoreItem(ctx, "gtd", createdGTD.ID)
	if err != nil {
		t.Fatalf("RestoreItem GTD failed: %v", err)
	}

	reportRes, err := reportRepo.GetContractsReport(ctx, ports.ReportFilter{})
	if err != nil {
		t.Fatalf("GetContractsReport failed: %v", err)
	}
	if reportRes.TotalContracts == 0 {
		t.Fatalf("expected at least 1 contract in report")
	}
}
