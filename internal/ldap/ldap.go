package ldap

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"

	"CurrencyControl/internal/configs"
)

type Client interface {
	Authenticate(username, password string) (bool, error)
}

type ldapClient struct {
	cfg configs.ADParams
}

func NewClient(cfg configs.ADParams) Client {
	return &ldapClient{cfg: cfg}
}

func (c *ldapClient) Authenticate(username, password string) (bool, error) {
	if username == "" {
		return false, fmt.Errorf("имя пользователя не должно быть пустым")
	}
	if password == "" {
		return false, fmt.Errorf("пароль не должен быть пустым")
	}

	userDN := fmt.Sprintf("%s@%s", username, c.cfg.Domain)

	conn, err := ldap.DialURL(c.cfg.Server)
	if err != nil {
		log.Printf("connection error to AD: %v", err)
		return false, fmt.Errorf("ошибка подключения к AD: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(userDN, password); err != nil {
		log.Printf("authorization error for %s: %v", username, err)
		return false, fmt.Errorf("неверный логин или пароль")
	}

	log.Printf("user %s authenticated via AD", username)
	return true, nil
}
