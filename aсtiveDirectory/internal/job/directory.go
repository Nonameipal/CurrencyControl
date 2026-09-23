package job

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"activeDirectory/internal/errs"
	"activeDirectory/models"
)

type ADConfig struct {
	Domain     string
	Server     string
	SearchBase string
}

func Authenticate(cfg ADConfig, username, password string) (*models.UserInfo, error) {
	if username == "" {
		return nil, errs.ErrUsernameIsEmpty
	}
	if password == "" {
		return nil, errs.ErrPasswordIsEmpty
	}

	userDN := fmt.Sprintf("%s@%s", username, cfg.Domain)

	conn, err := ldap.DialURL(cfg.Server)
	if err != nil {
		log.Printf("connection error to AD: %v", err)
		return nil, errs.ErrConnection
	}
	defer conn.Close()

	if err := conn.Bind(userDN, password); err != nil {
		log.Printf("authorization error for %s: %v", username, err)
		return nil, errs.ErrAuthorization
	}

	// Берём baseDN из конфига (AD_SEARCH_BASE)
	baseDN := cfg.SearchBase
	if baseDN == "" {
		baseDN = fmt.Sprintf("DC=%s", strings.ReplaceAll(cfg.Domain, ".", ",DC="))
	}

	// Запрашиваем все нужные атрибуты одним запросом
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1,     // максимум 1 результат
		0,     // без таймаута
		false,
		fmt.Sprintf("(sAMAccountName=%s)", ldap.EscapeFilter(username)),
		[]string{
			"displayName", // "Иванов Иван Иванович" — основной источник ФИО
			"sn",          // Фамилия
			"givenName",   // Имя
			"middleName",  // Отчество
			"mail",        // Email
			"department",  // Отдел
			"title",       // Должность
			"telephoneNumber",
			"mobile",
		},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Printf("search error for %s: %v", username, err)
		return nil, errs.ErrConnection
	}

	var info models.UserInfo
	if len(sr.Entries) > 0 {
		entry := sr.Entries[0]

		// Логируем все полученные атрибуты для отладки
		log.Printf("AD attributes for %s:", username)
		for _, attr := range entry.Attributes {
			log.Printf("  %s = %v", attr.Name, attr.Values)
		}

		// Приоритет: sn/givenName/middleName → иначе парсим displayName
		info.LastName = entry.GetAttributeValue("sn")
		info.FirstName = entry.GetAttributeValue("givenName")
		info.MiddleName = entry.GetAttributeValue("middleName")

		// Если отдельные поля пустые — берём displayName и разбиваем по пробелу
		if info.LastName == "" && info.FirstName == "" {
			displayName := entry.GetAttributeValue("displayName")
			parts := strings.Fields(displayName) // разбивает по любым пробелам
			switch len(parts) {
			case 1:
				info.LastName = parts[0]
			case 2:
				info.LastName, info.FirstName = parts[0], parts[1]
			case 3:
				info.LastName, info.FirstName, info.MiddleName = parts[0], parts[1], parts[2]
			default:
				if len(parts) > 3 {
					info.LastName = parts[0]
					info.FirstName = parts[1]
					info.MiddleName = strings.Join(parts[2:], " ")
				}
			}
		}

		// Дополнительные поля
		info.Email = entry.GetAttributeValue("mail")
		info.Department = entry.GetAttributeValue("department")
		info.Position = entry.GetAttributeValue("title")
		info.Phone = entry.GetAttributeValue("telephoneNumber")
		if info.Phone == "" {
			info.Phone = entry.GetAttributeValue("mobile")
		}
	}

	log.Printf("user %s authenticated: %s %s %s", username, info.LastName, info.FirstName, info.MiddleName)
	return &info, nil
}

func NewADConfig() ADConfig {
	return ADConfig{
		Domain:     os.Getenv("AD_DOMAIN"),
		Server:     os.Getenv("AD_SERVER"),
		SearchBase: os.Getenv("AD_SEARCH_BASE"),
	}
}
