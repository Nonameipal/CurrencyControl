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

	createdPO, err := poRepo.Create(ctx, domain.PaymentOrder{
		InvoiceID:       createdInvUSD.ID,
		ValueDate:       now,
		Amount:          2500,
		Currency:        "USD",
		SenderName:      "Sender",
		ReceiverName:    "Test Receiver",
		ReceiverBank:    "Test Bank",
		ReceiverCountry: "USA",
		PaymentPurpose:  "Оплата",
		CreatedBy:       "test_user",
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
	createdGTD, err := gtdRepo.Create(ctx, domain.GTD{
		ContractID:     createdContract.ID,
		InvoiceID:      createdInvUSD.ID,
		GTDNumber:      gtdNum,
		Currency:       "USD",
		GTDDate:        now,
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

func TestInvoice_SharedLimit_ActAndGTD(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	contractRepo := repository.NewContractRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	gtdRepo := repository.NewGTDRepository(db)
	actRepo := repository.NewActRepository(db)
	approvalRepo := repository.NewApprovalRepository(db)

	now := time.Now().UTC()
	contractDate := now.AddDate(0, -1, 0)
	deliveryDate := now.AddDate(0, 3, 0)

	contractNum := fmt.Sprintf("SHARED-CNT-%d", now.UnixNano())
	cnt, err := contractRepo.Create(ctx, domain.Contract{
		ContractNumber:   contractNum,
		ContractDate:     contractDate,
		DeliveryDate:     deliveryDate,
		TotalAmount:      10000,
		ContractCurrency: "USD",
		Subject:          "Контракт с актами и ГТД",
		Status:           domain.ContractStatusActive,
	})
	if err != nil {
		t.Fatalf("Create contract failed: %v", err)
	}
	defer func() {
		_ = db.Unscoped().Where("contract_id = ?", cnt.ID).Delete(&domain.GTD{}).Error
		_ = db.Unscoped().Where("contract_id = ?", cnt.ID).Delete(&domain.Act{}).Error
		_ = db.Unscoped().Where("contract_id = ?", cnt.ID).Delete(&domain.Invoice{}).Error
		_ = db.Unscoped().Where("id = ?", cnt.ID).Delete(&domain.Contract{}).Error
	}()

	invNum := fmt.Sprintf("SHARED-INV-%d", now.UnixNano())
	inv, err := invoiceRepo.Create(ctx, domain.Invoice{
		ContractID:    cnt.ID,
		InvoiceNumber: invNum,
		InvoiceDate:   now,
		Amount:        5000,
		Currency:      "USD",
	}, "USD")
	if err != nil {
		t.Fatalf("Create invoice 5000 USD failed: %v", err)
	}

	actNum := fmt.Sprintf("ACT-3K-%d", now.UnixNano())
	createdAct, err := actRepo.Create(ctx, domain.Act{
		ContractID: cnt.ID,
		InvoiceID:  inv.ID,
		ActNumber:  actNum,
		ActDate:    now,
		ActAmount:  3000,
	})
	if err != nil {
		t.Fatalf("expected Act for 3000 to succeed, got %v", err)
	}

	gtdNumPass := fmt.Sprintf("GTD-5K-%d", now.UnixNano())
	createdGTD, err := gtdRepo.Create(ctx, domain.GTD{
		ContractID:   cnt.ID,
		InvoiceID:    inv.ID,
		GTDNumber:    gtdNumPass,
		GTDDate:      now,
		GTDAmount:    5000,
		ClosesAmount: 5000,
	})
	if err != nil {
		t.Fatalf("expected GTD for 5000 to succeed because Act does not deduct from invoice, got %v", err)
	}

	gtdNumOverflow := fmt.Sprintf("GTD-EXTRA-%d", now.UnixNano())
	_, err = gtdRepo.Create(ctx, domain.GTD{
		ContractID:   cnt.ID,
		InvoiceID:    inv.ID,
		GTDNumber:    gtdNumOverflow,
		GTDDate:      now,
		GTDAmount:    100,
		ClosesAmount: 100,
	})
	if err == nil {
		t.Fatalf("expected extra document to be rejected because invoice is already 100%% closed (5000/5000)")
	}

	_, err = approvalRepo.SetCurrencyControlDecision(ctx, "gtd", createdGTD.ID, "accepted", "ok", "controller1")
	if err != nil {
		t.Fatalf("SetCurrencyControlDecision for GTD failed: %v", err)
	}
	_, err = approvalRepo.SetComplianceDecision(ctx, "gtd", createdGTD.ID, "approve", "ok", "compliance1")
	if err != nil {
		t.Fatalf("SetComplianceDecision for GTD failed: %v", err)
	}

	gtdDetail, err := approvalRepo.GetApprovalDetail(ctx, "gtd", createdGTD.ID)
	if err != nil {
		t.Fatalf("GetApprovalDetail for GTD failed: %v", err)
	}
	if gtdDetail.EntityType != "gtd" {
		t.Errorf("expected gtdDetail.EntityType to be 'gtd', got '%s'", gtdDetail.EntityType)
	}
	if gtdDetail.ApprovalStatus != domain.ApprovalStatusApproved {
		t.Errorf("expected gtd to be approved, got '%s'", gtdDetail.ApprovalStatus)
	}

	// Verify Act exists directly without approval
	fetchedAct, err := actRepo.GetByID(ctx, createdAct.ID)
	if err != nil || fetchedAct == nil {
		t.Fatalf("GetByID for Act failed: %v", err)
	}
	if fetchedAct.ActNumber != actNum {
		t.Errorf("expected act number %s, got %s", actNum, fetchedAct.ActNumber)
	}
}

func TestContract_CRUD_DB_Lifecycle(t *testing.T) {
	db := getTestDB(t)

	ctx := context.Background()
	contractRepo := repository.NewContractRepository(db)
	now := time.Now().Truncate(24 * time.Hour)
	delivery := now.Add(30 * 24 * time.Hour)
	endDate := now.Add(60 * 24 * time.Hour)
	returnDays := 30

	cNum := fmt.Sprintf("CNT-CRUD-%d", now.UnixNano())
	created, err := contractRepo.Create(ctx, domain.Contract{
		ContractNumber:   cNum,
		ContractDate:     now,
		DeliveryDate:     delivery,
		ContractEndDate:  &endDate,
		ReturnDays:       &returnDays,
		TotalAmount:      10000,
		ContractCurrency: "USD",
		Subject:          "Поставка оборудования",
		SenderName:       "Test Sender LLC",
		ReceiverName:     "Test Receiver CJSC",
		ReceiverBank:     "Test Bank",
		ReceiverCountry:  "USA",
		CreatedBy:        "operator1",
	})
	if err != nil {
		t.Fatalf("Create contract failed: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created contract ID > 0")
	}

	// 1. Read by ID
	fetched, err := contractRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID contract failed: %v", err)
	}
	if fetched.ContractNumber != cNum {
		t.Errorf("expected contract number %s, got %s", cNum, fetched.ContractNumber)
	}
	if fetched.ReturnDays == nil || *fetched.ReturnDays != 30 {
		t.Errorf("expected return_days 30, got %v", fetched.ReturnDays)
	}
	if fetched.SenderName != "Test Sender LLC" {
		t.Errorf("expected sender_name 'Test Sender LLC', got '%s'", fetched.SenderName)
	}

	// 2. Update (Edit)
	newReturnDays := 45
	newSubject := "Обновленный предмет поставки"
	newSender := "Updated Sender LLC"
	fetched.ReturnDays = &newReturnDays
	fetched.Subject = newSubject
	fetched.SenderName = newSender
	fetched.TotalAmount = 15000
	fetched.UpdatedBy = "controller1"

	updated, err := contractRepo.Update(ctx, created.ID, fetched)
	if err != nil {
		t.Fatalf("Update contract failed: %v", err)
	}
	if updated.Subject != newSubject {
		t.Errorf("expected subject '%s', got '%s'", newSubject, updated.Subject)
	}
	if updated.SenderName != newSender {
		t.Errorf("expected sender_name '%s', got '%s'", newSender, updated.SenderName)
	}
	if updated.ReturnDays == nil || *updated.ReturnDays != 45 {
		t.Errorf("expected return_days 45, got %v", updated.ReturnDays)
	}

	// 3. Soft Delete
	delCtx := context.WithValue(ctx, domain.CtxKeyLogin, "compliance_admin")
	err = contractRepo.SoftDelete(delCtx, created.ID)
	if err != nil {
		t.Fatalf("SoftDelete contract failed: %v", err)
	}

	// 4. Verify cannot be found via normal GetByID
	_, err = contractRepo.GetByID(ctx, created.ID)
	if err == nil {
		t.Fatalf("expected error after SoftDelete, got nil")
	}

	// 5. Verify row still exists in DB with deleted_at IS NOT NULL
	var deletedRow domain.Contract
	dbErr := db.Unscoped().First(&deletedRow, created.ID).Error
	if dbErr != nil {
		t.Fatalf("expected unscoped contract row to exist in DB, got error %v", dbErr)
	}
	if !deletedRow.DeletedAt.Valid {
		t.Fatalf("expected DeletedAt to be valid (soft delete), got %v", deletedRow.DeletedAt)
	}
}

func TestInvoice_CRUD_DB_Lifecycle(t *testing.T) {
	db := getTestDB(t)

	ctx := context.Background()
	contractRepo := repository.NewContractRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)

	now := time.Now().Truncate(24 * time.Hour)
	delivery := now.Add(60 * 24 * time.Hour)
	endDate := now.Add(90 * 24 * time.Hour)
	returnDays := 30

	cNum := fmt.Sprintf("CNT-INV-LIFE-%d", now.UnixNano())
	cnt, err := contractRepo.Create(ctx, domain.Contract{
		ContractNumber:   cNum,
		ContractDate:     now,
		DeliveryDate:     delivery,
		ContractEndDate:  &endDate,
		ReturnDays:       &returnDays,
		TotalAmount:      10000,
		ContractCurrency: "USD",
		Subject:          "Для проверки инвойсов",
		SenderName:       "Поставщик",
		ReceiverName:     "Покупатель",
		ReceiverBank:     "Банк",
		ReceiverCountry:  "USA",
		CreatedBy:        "operator1",
	})
	if err != nil {
		t.Fatalf("Create contract failed: %v", err)
	}
	defer func() {
		_ = db.Unscoped().Where("contract_id = ?", cnt.ID).Delete(&domain.Invoice{}).Error
		_ = db.Unscoped().Where("id = ?", cnt.ID).Delete(&domain.Contract{}).Error
	}()

	// 1. Create Invoice
	invNum := fmt.Sprintf("INV-LIFE-%d", now.UnixNano())
	docPath := "uploads/invoices/test.pdf"
	createdInv, err := invoiceRepo.Create(ctx, domain.Invoice{
		ContractID:    cnt.ID,
		InvoiceNumber: invNum,
		InvoiceDate:   now,
		Amount:        3000,
		Currency:      "USD",
		DocumentPath:  &docPath,
		CreatedBy:     "operator1",
	}, "USD")
	if err != nil {
		t.Fatalf("Create invoice failed: %v", err)
	}
	if createdInv.ID == 0 {
		t.Fatalf("expected created invoice ID > 0")
	}

	// 2. Read by ID
	fetchedInv, err := invoiceRepo.GetByID(ctx, createdInv.ID)
	if err != nil {
		t.Fatalf("GetByID invoice failed: %v", err)
	}
	if fetchedInv.InvoiceNumber != invNum {
		t.Errorf("expected invoice number %s, got %s", invNum, fetchedInv.InvoiceNumber)
	}
	if fetchedInv.Amount != 3000 {
		t.Errorf("expected amount 3000, got %f", fetchedInv.Amount)
	}
	if fetchedInv.Currency != "USD" {
		t.Errorf("expected currency 'USD', got '%s'", fetchedInv.Currency)
	}

	// Verify contract remaining amount updated: 10000 - 3000 = 7000
	fetchedContract, err := contractRepo.GetByID(ctx, cnt.ID)
	if err != nil {
		t.Fatalf("GetByID contract failed: %v", err)
	}
	if fetchedContract.RemainingAmount != 7000 {
		t.Errorf("expected remaining_amount 7000, got %f", fetchedContract.RemainingAmount)
	}

	// 3. Update (Edit) Invoice
	newInvNum := invNum + "-EDIT"
	fetchedInv.InvoiceNumber = newInvNum
	fetchedInv.Amount = 4500
	fetchedInv.UpdatedBy = "compliance1"

	updatedInv, err := invoiceRepo.Update(ctx, fetchedInv.ID, fetchedInv)
	if err != nil {
		t.Fatalf("Update invoice failed: %v", err)
	}
	if updatedInv.InvoiceNumber != newInvNum {
		t.Errorf("expected updated invoice number %s, got %s", newInvNum, updatedInv.InvoiceNumber)
	}
	if updatedInv.Amount != 4500 {
		t.Errorf("expected updated amount 4500, got %f", updatedInv.Amount)
	}

	// Verify contract remaining amount updated: 10000 - 4500 = 5500
	fetchedContract, err = contractRepo.GetByID(ctx, cnt.ID)
	if err != nil {
		t.Fatalf("GetByID contract failed: %v", err)
	}
	if fetchedContract.RemainingAmount != 5500 {
		t.Errorf("expected remaining_amount 5500, got %f", fetchedContract.RemainingAmount)
	}

	// 4. Soft Delete Invoice
	delCtx := context.WithValue(ctx, domain.CtxKeyLogin, "compliance_admin")
	err = invoiceRepo.SoftDelete(delCtx, createdInv.ID)
	if err != nil {
		t.Fatalf("SoftDelete invoice failed: %v", err)
	}

	// 5. Verify invoice not found in normal queries
	_, err = invoiceRepo.GetByID(ctx, createdInv.ID)
	if err == nil {
		t.Fatalf("expected error after SoftDelete, got nil")
	}

	// Verify contract remaining amount restored: 10000
	fetchedContract, err = contractRepo.GetByID(ctx, cnt.ID)
	if err != nil {
		t.Fatalf("GetByID contract failed: %v", err)
	}
	if fetchedContract.RemainingAmount != 10000 {
		t.Errorf("expected remaining_amount restored to 10000, got %f", fetchedContract.RemainingAmount)
	}

	// Verify row still exists in DB with deleted_at IS NOT NULL
	var deletedInv domain.Invoice
	dbErr := db.Unscoped().First(&deletedInv, createdInv.ID).Error
	if dbErr != nil {
		t.Fatalf("expected unscoped invoice row to exist in DB, got error %v", dbErr)
	}
	if !deletedInv.DeletedAt.Valid {
		t.Fatalf("expected DeletedAt to be valid (soft delete), got %v", deletedInv.DeletedAt)
	}
}
