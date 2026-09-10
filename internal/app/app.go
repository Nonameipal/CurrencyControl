package app

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
	db, err := database.InitConnection(ctx)
	if err != nil {
		return err
	}
	defer database.CloseConnection(db)

	ldapClient := ldap.NewClient(configs.AppSettings.ADParams)
	authRepo := repository.NewAuthRepository(db)
	authSvc := service.NewAuthService(authRepo, ldapClient)
	authHandler := delivery.NewAuthHandler(authSvc)
	delivery.SetAuthService(authSvc) 

	contractRepo := repository.NewContractRepository(db)
	contractService := service.NewContractService(contractRepo)
	contractHandler := delivery.NewContractHandler(contractService)

	absClient := abs.NewClient(configs.AppSettings.ABSParams)
	counterpartyRepo := repository.NewCounterpartyRepository(db)
	counterpartyService := service.NewCounterpartyService(counterpartyRepo, absClient)
	counterpartyHandler := delivery.NewCounterpartyHandler(counterpartyService)

	dictHandler := delivery.NewDictionaryHandler(db)

	invoiceRepo := repository.NewInvoiceRepository(db)
	invoiceSvc := service.NewInvoiceService(invoiceRepo)
	gtdRepo := repository.NewGTDRepository(db)
	gtdSvc := service.NewGTDService(gtdRepo)
	addlRepo := repository.NewAdditionalAgreementRepository(db)
	addlSvc := service.NewAdditionalAgreementService(addlRepo)
	invoiceHandler := delivery.NewInvoiceHandler(invoiceSvc, gtdSvc, addlSvc)

	branchRepo := repository.NewBranchRepository(db)
	branchSvc := service.NewBranchService(branchRepo)
	branchHandler := delivery.NewBranchHandler(branchSvc)

	auditLogRepo := repository.NewAuditLogRepository(db)
	auditLogSvc := service.NewAuditLogService(auditLogRepo)
	delivery.SetAuditService(auditLogSvc)
	auditHandler := delivery.NewAuditHandler(auditLogSvc)

	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	reportHandler := delivery.NewReportHandler(reportSvc)

	router := delivery.InitRoutes(
		contractHandler,
		counterpartyHandler,
		dictHandler,
		invoiceHandler,
		authHandler,
		branchHandler,
		auditHandler,
		reportHandler,
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