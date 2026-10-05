package ldap

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"CurrencyControl/internal/configs"
)


type UserInfo struct {
	LastName  string
	FirstName string
	Email     string
}

type Client interface {
	Authenticate(username, password string) (*UserInfo, error)
}

type ldapClient struct {
	cfg configs.ADParams
}

func NewClient(cfg configs.ADParams) Client {
	return &ldapClient{cfg: cfg}
}

func (c *ldapClient) Authenticate(username, password string) (*UserInfo, error) {
	if username == "" {
		return nil, fmt.Errorf("имя пользователя не должно быть пустым")
	}
	if password == "" {
		return nil, fmt.Errorf("пароль не должен быть пустым")
	}
	userDN := fmt.Sprintf("%s@%s", username, c.cfg.Domain)
	conn, err := ldap.DialURL(c.cfg.Server)
	if err != nil {
		log.Printf("connection error to AD: %v", err)
		return nil, fmt.Errorf("сервер аутентификации недоступен: %w", err)
	}
	defer conn.Close()
	if err := conn.Bind(userDN, password); err != nil {
		log.Printf("authorization error for %s: %v", username, err)
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	info := c.fetchUserInfo(conn, username)
	log.Printf("user %s authenticated via AD: lastName=%s firstName=%s", username, info.LastName, info.FirstName)
	return info, nil
}

func (c *ldapClient) fetchUserInfo(conn *ldap.Conn, username string) *UserInfo {
	baseDN := c.cfg.SearchBase
	if baseDN == "" {
		baseDN = fmt.Sprintf("DC=%s", strings.ReplaceAll(c.cfg.Domain, ".", ",DC="))
	}
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1,   
		0,    
		false,
		fmt.Sprintf("(sAMAccountName=%s)", ldap.EscapeFilter(username)),
		[]string{"sn", "givenName", "mail", "displayName"},
		nil,
	)
	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Printf("LDAP search error for %s: %v", username, err)
		return &UserInfo{}
	}
	if len(sr.Entries) == 0 {
		return &UserInfo{}
	}
	entry := sr.Entries[0]
	info := &UserInfo{
		LastName:  entry.GetAttributeValue("sn"),
		FirstName: entry.GetAttributeValue("givenName"),
		Email:     entry.GetAttributeValue("mail"),
	}

	if info.LastName == "" && info.FirstName == "" {
		displayName := entry.GetAttributeValue("displayName")
		parts := strings.Fields(displayName)
		switch len(parts) {
		case 1:
			info.LastName = parts[0]
		case 2:
			info.LastName, info.FirstName = parts[0], parts[1]
		default:
			if len(parts) >= 3 {
				info.LastName, info.FirstName = parts[0], parts[1]
			}
		}
	}

	return info
}
