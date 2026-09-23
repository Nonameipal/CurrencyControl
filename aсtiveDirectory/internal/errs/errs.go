package errs

import "errors"

var (
	ErrUsernameIsEmpty  = errors.New("имя пользователя не должен быть пустым")
	ErrPasswordIsEmpty  = errors.New("пароль пользователя не должен быть пустым")
	ErrAuthorization    = errors.New("ошибка авторизации")
	ErrConnection       = errors.New("ошибка подключения к AD")
	ErrValidationFailed = errors.New("validation failed")
)
