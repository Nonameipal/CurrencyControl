package http

import (
	"activeDirectory/internal/errs"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"net/http"
)

func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, errs.ErrPasswordIsEmpty),
		errors.Is(err, errs.ErrValidationFailed),
		errors.Is(err, errs.ErrUsernameIsEmpty):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, errs.ErrAuthorization):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})

	case errors.Is(err, errs.ErrConnection):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "сервис временно недоступен"})

	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("something went wrong: %s", err.Error())})
	}
}

