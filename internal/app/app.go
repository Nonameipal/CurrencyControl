package app

import (
	"context"
	"net/http"
	"time"

	"CurrencyControl/internal/configs"
	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/infrostucture/database"
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

	contractRepo := repository.NewContractRepository(db)
	contractService := service.NewContractService(contractRepo)
	contractHandler := delivery.NewContractHandler(contractService)

	counterpartyRepo := repository.NewCounterpartyRepository(db)
	counterpartyService := service.NewCounterpartyService(counterpartyRepo)
	counterpartyHandler := delivery.NewCounterpartyHandler(counterpartyService)

	dictHandler := delivery.NewDictionaryHandler(db)

	invoiceRepo := repository.NewInvoiceRepository(db)
	invoiceSvc := service.NewInvoiceService(invoiceRepo)
	gtdRepo := repository.NewGTDRepository(db)
	gtdSvc := service.NewGTDService(gtdRepo)
	addlRepo := repository.NewAdditionalAgreementRepository(db)
	addlSvc := service.NewAdditionalAgreementService(addlRepo)
	invoiceHandler := delivery.NewInvoiceHandler(invoiceSvc, gtdSvc, addlSvc)

	router := delivery.InitRoutes(contractHandler, counterpartyHandler, dictHandler, invoiceHandler)

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