package http

import (
	"activeDirectory/internal/job"
	"activeDirectory/models"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"net/http"
)

// AuthenticateUser godoc
// @Summary      Авторизация пользователя
// @Description  Проверяет учётные данные пользователя через Active Directory и возвращает ФИО
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        account  body      models.RequestUser  true  "Учётные данные пользователя"
// @Success      201      {object}  models.UserInfo
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      409      {object}  models.ErrorResponse
// @Router       /auth/login [post]
func AuthenticateUser(c *gin.Context) {
	var user models.RequestUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	cfg := job.NewADConfig()

	result, err := job.Authenticate(cfg, user.Username, user.Password)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

