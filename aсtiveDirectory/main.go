package main

import (
	_ "activeDirectory/docs" //
	"activeDirectory/internal/http"
)

// @title           Check Auth Active_Directory
// @version         1.0
// @description     API для управления клиентами, счетами, операциями и офисами банка
// @host            localhost:8083

func main() {

	if err := http.RunServer(); err != nil {
		panic(err)
	}
}
