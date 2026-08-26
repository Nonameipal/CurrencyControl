package http

import (
	"net/http"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	
	_ "CurrencyControl/docs" // Важно для инициализации Swagger
)

func InitRoutes(dashboardHandler *ContractHandler, companyHandler *CounterpartyHandler) http.Handler {
	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	
	api.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(next)
	})

	api.HandleFunc("/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)
	
	// Роут для создания компании (ҶДММ)
	api.HandleFunc("/companies", companyHandler.Create).Methods(http.MethodPost)

	// Роут для создания контракта (с загрузкой PDF)
	api.HandleFunc("/contracts", dashboardHandler.Create).Methods(http.MethodPost)

	// Роут для интерфейса Swagger
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return r
}