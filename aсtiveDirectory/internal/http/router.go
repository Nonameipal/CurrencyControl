package http

import (
	"fmt"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RunServer() error {
	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.POST("/auth/login", AuthenticateUser)
	fmt.Println()
	fmt.Println("http://localhost:8083/auth/login")
	fmt.Println("http://localhost:8083/swagger/index.html")

	return r.Run(":8083")
}

