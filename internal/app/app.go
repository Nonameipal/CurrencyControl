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
	"CurrencyControl/internal/delivery/http/handlers/auth"
	"CurrencyControl/internal/delivery/http/handlers/counterparty"
	"CurrencyControl/internal/delivery/http/handlers/document"
	"CurrencyControl/internal/delivery/http/handlers/workflow"
	"CurrencyControl/internal/delivery/http/middleware"
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
	gormDB, err := database.InitConnection(ctx)
	if err != nil {
		return err
	}
	defer database.CloseConnection(gormDB)

	branchRepo := repository.NewBranchRepository(gormDB)
	branchSvc := service.NewBranchService(branchRepo)
	branchHandler := counterparty.NewBranchHandler(branchSvc)

	ldapClient := ldap.NewClient(configs.AppSettings.ADParams)
	authRepo := repository.NewAuthRepository(gormDB)
	authSvc := service.NewAuthService(authRepo, ldapClient)
	authHandler := auth.NewAuthHandler(authSvc, branchSvc)
	middleware.SetAuthService(authSvc)

	contractRepo := repository.NewContractRepository(gormDB)
	contractService := service.NewContractService(contractRepo)
	contractHandler := document.NewContractHandler(contractService)

	absClient := abs.NewClient(configs.AppSettings.ABSParams)
	counterpartyRepo := repository.NewCounterpartyRepository(gormDB)
	counterpartyService := service.NewCounterpartyService(counterpartyRepo, absClient)
	counterpartyHandler := counterparty.NewCounterpartyHandler(counterpartyService, absClient)

	dictHandler := counterparty.NewDictionaryHandler(gormDB)

	invoiceRepo := repository.NewInvoiceRepository(gormDB)
	invoiceSvc := service.NewInvoiceService(invoiceRepo)
	gtdRepo := repository.NewGTDRepository(gormDB)
	gtdSvc := service.NewGTDService(gtdRepo)
	gtdHandler := document.NewGTDHandler(gtdSvc)
	actRepo := repository.NewActRepository(gormDB)
	actSvc := service.NewActService(actRepo)
	actHandler := document.NewActHandler(actSvc)
	addlRepo := repository.NewAdditionalAgreementRepository(gormDB)
	addlSvc := service.NewAdditionalAgreementService(addlRepo)
	paymentOrderRepo := repository.NewPaymentOrderRepository(gormDB)
	paymentOrderSvc := service.NewPaymentOrderService(paymentOrderRepo)
	paymentOrderHandler := document.NewPaymentOrderHandler(paymentOrderSvc)
	invoiceHandler := document.NewInvoiceHandler(invoiceSvc, gtdSvc, addlSvc, paymentOrderSvc)

	auditLogRepo := repository.NewAuditLogRepository(gormDB)
	auditLogSvc := service.NewAuditLogService(auditLogRepo)
	middleware.SetAuditService(auditLogSvc)
	auditHandler := workflow.NewAuditHandler(auditLogSvc)

	reportRepo := repository.NewReportRepository(gormDB)
	reportSvc := service.NewReportService(reportRepo)
	reportHandler := workflow.NewReportHandler(reportSvc)

	trashRepo := repository.NewTrashRepository(gormDB)
	trashSvc := service.NewTrashService(trashRepo)
	trashHandler := workflow.NewTrashHandler(trashSvc)

	permRepo := repository.NewPermissionRepository(gormDB)
	permSvc := service.NewPermissionService(permRepo)
	middleware.SetPermissionService(permSvc)
	complianceHandler := workflow.NewComplianceHandler(permSvc)

	approvalRepo := repository.NewApprovalRepository(gormDB)
	approvalSvc := service.NewApprovalService(approvalRepo, auditLogSvc)
	approvalHandler := workflow.NewApprovalHandler(approvalSvc)

	gtdExtRepo := repository.NewGTDExtensionRepository(gormDB)
	gtdExtSvc := service.NewGTDExtensionService(gtdExtRepo, auditLogSvc)
	gtdExtHandler := document.NewGTDExtensionHandler(gtdExtSvc)

	docRepo := repository.NewDocumentRepository(gormDB)
	docSvc := service.NewDocumentService(docRepo)
	docHandler := document.NewDocumentHandler(docSvc)

	myDocsRepo := repository.NewMyDocumentsRepository(gormDB)
	myDocsSvc := service.NewMyDocumentsService(myDocsRepo)
	myDocsHandler := document.NewMyDocumentsHandler(myDocsSvc)

	router := delivery.InitRoutes(delivery.Handlers{
		Contract:     contractHandler,
		Company:      counterpartyHandler,
		Dict:         dictHandler,
		Invoice:      invoiceHandler,
		GTD:          gtdHandler,
		Act:          actHandler,
		Auth:         authHandler,
		Branch:       branchHandler,
		Audit:        auditHandler,
		Report:       reportHandler,
		Trash:        trashHandler,
		Compliance:   complianceHandler,
		Approval:     approvalHandler,
		PaymentOrder: paymentOrderHandler,
		GTDExt:       gtdExtHandler,
		Document:     docHandler,
		MyDocuments:  myDocsHandler,
	})

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
