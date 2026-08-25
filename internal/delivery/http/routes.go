package http

import (
	"net/http"
	"github.com/gorilla/mux"
)

func InitRoutes(dashboardHandler *ContractHandler) http.Handler {
	r := mux.NewRouter()

	api := r.PathPrefix("/api").Subrouter()
	
	api.Use(func(next http.Handler) http.Handler {
		return AuthMiddleware(next)
	})

	api.HandleFunc("/dashboard", dashboardHandler.Dashboard).Methods(http.MethodGet)

	return r
}