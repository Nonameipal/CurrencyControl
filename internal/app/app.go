package app

// @title Currency Control API
// @version 1.0
// @description API сервера для системы валютного контроля.
// @host localhost:8088
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Введите Bearer <access_token> (полученный при /auth/login)

import (
	"context"
	"net/http"
	"time"

	"CurrencyControl/internal/abs"
	"CurrencyControl/internal/configs"
	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/infrostucture/database"
	"CurrencyControl/internal/ldap"
	"CurrencyControl/internal/logger"
	"CurrencyControl/internal/repository"
	"CurrencyControl/internal/service"
)

func Run() error {
	log := logger.GetLogger()
	if err := configs.Load(); err != nil {
		return err
	}

	ctx := context.Background()
	db, gormDB, err := database.InitConnection(ctx)
	if err != nil {
		return err
	}
	defer database.CloseConnection(db)

	branchRepo := repository.NewBranchRepository(gormDB)
	branchSvc := service.NewBranchService(branchRepo)
	branchHandler := delivery.NewBranchHandler(branchSvc)

	ldapClient := ldap.NewClient(configs.AppSettings.ADParams)
	authRepo := repository.NewAuthRepository(gormDB)
	authSvc := service.NewAuthService(authRepo, ldapClient)
	authHandler := delivery.NewAuthHandler(authSvc, branchSvc)
	delivery.SetAuthService(authSvc) 

	contractRepo := repository.NewContractRepository(gormDB)
	contractService := service.NewContractService(contractRepo)
	contractHandler := delivery.NewContractHandler(contractService)

	absClient := abs.NewClient(configs.AppSettings.ABSParams)
	counterpartyRepo := repository.NewCounterpartyRepository(gormDB)
	counterpartyService := service.NewCounterpartyService(counterpartyRepo, absClient)
	counterpartyHandler := delivery.NewCounterpartyHandler(counterpartyService)

	dictHandler := delivery.NewDictionaryHandler(gormDB)

	invoiceRepo := repository.NewInvoiceRepository(gormDB)
	invoiceSvc := service.NewInvoiceService(invoiceRepo)
	gtdRepo := repository.NewGTDRepository(gormDB)
	gtdSvc := service.NewGTDService(gtdRepo)
	addlRepo := repository.NewAdditionalAgreementRepository(gormDB)
	addlSvc := service.NewAdditionalAgreementService(addlRepo)
	paymentOrderRepo := repository.NewPaymentOrderRepository(gormDB)
	paymentOrderSvc := service.NewPaymentOrderService(paymentOrderRepo)
	paymentOrderHandler := delivery.NewPaymentOrderHandler(paymentOrderSvc)
	invoiceHandler := delivery.NewInvoiceHandler(invoiceSvc, gtdSvc, addlSvc, paymentOrderSvc)

	auditLogRepo := repository.NewAuditLogRepository(gormDB)
	auditLogSvc := service.NewAuditLogService(auditLogRepo)
	delivery.SetAuditService(auditLogSvc)
	auditHandler := delivery.NewAuditHandler(auditLogSvc)

	reportRepo := repository.NewReportRepository(gormDB)
	reportSvc := service.NewReportService(reportRepo)
	reportHandler := delivery.NewReportHandler(reportSvc)

	trashRepo := repository.NewTrashRepository(gormDB)
	trashSvc := service.NewTrashService(trashRepo)
	trashHandler := delivery.NewTrashHandler(trashSvc)

	permRepo := repository.NewPermissionRepository(gormDB)
	permSvc := service.NewPermissionService(permRepo)
	delivery.SetPermissionService(permSvc)
	complianceHandler := delivery.NewComplianceHandler(permSvc)

	approvalRepo := repository.NewApprovalRepository(gormDB)
	approvalSvc := service.NewApprovalService(approvalRepo, auditLogSvc)
	approvalHandler := delivery.NewApprovalHandler(approvalSvc)

	gtdExtRepo := repository.NewGTDExtensionRepository(gormDB)
	gtdExtSvc := service.NewGTDExtensionService(gtdExtRepo, auditLogSvc)
	gtdExtHandler := delivery.NewGTDExtensionHandler(gtdExtSvc)

	router := delivery.InitRoutes(
		contractHandler,
		counterpartyHandler,
		dictHandler,
		invoiceHandler,
		authHandler,
		branchHandler,
		auditHandler,
		reportHandler,
		trashHandler,
		complianceHandler,
		approvalHandler,
		paymentOrderHandler,
		gtdExtHandler,
	)

	server := &http.Server{
		Addr:         ":" + configs.AppSettings.AppParams.PortRun,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Info().Str("address", server.Addr).Msg("server listening")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}