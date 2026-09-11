package abs

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"CurrencyControl/internal/configs"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/logger"
)

type ABSClientInfo struct {
	FullName    string   `json:"full_name"`
	INN         string   `json:"inn"`
	ClientType  string   `json:"client_type"` 
	Phones      []string `json:"phones"`
	Accounts    []string `json:"accounts"`
	RawResponse string   `json:"raw_response,omitempty"`
}

type ABSClient interface {
	GetClientByINN(ctx context.Context, inn string) (*ABSClientInfo, error)
}

type absClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(cfg configs.ABSParams) ABSClient {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://10.64.20.34:8181/cxf/statement/v1"
	}
	return &absClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (c *absClient) GetClientByINN(ctx context.Context, inn string) (*ABSClientInfo, error) {
	inn = strings.TrimSpace(inn)
	if inn == "" {
		return nil, fmt.Errorf("ИНН не может быть пустым")
	}
	endpoint := c.endpoint
	if endpoint == "" {
		endpoint = "http://10.64.20.34:8181/cxf/statement/v1"
	}

	reqID := newUUID()
	sessID := newUUID()
	procID := newUUID()
	opDate := time.Now().Format("2006-01-02T15:04:05")

	soapReq := fmt.Sprintf(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:v1="http://bus.colvir.com/service/statement/v1" xmlns:v11="http://bus.colvir.com/common/support/v1">
<soapenv:Header/>
<soapenv:Body>
<v1:loadColvirReportDataElem>
<v11:head>
<v11:requestId>%s</v11:requestId>
<v11:sessionId>%s</v11:sessionId>
<v11:processId>%s</v11:processId>
<v11:params>
<v11:clientType>CBS</v11:clientType>
<v11:interfaceVersion>1.0</v11:interfaceVersion>
<v11:language>ru</v11:language>
<v11:operationalDate>%s</v11:operationalDate>
</v11:params>
</v11:head>
<v1:reportCode>Z_342_CLI_INFO_BYPH2</v1:reportCode>
<v1:reportParams>prmS_CLI_TAX_Code=>%s</v1:reportParams>
<v1:rawFormat>true</v1:rawFormat>
</v1:loadColvirReportDataElem>
</soapenv:Body>
</soapenv:Envelope>`, reqID, sessID, procID, opDate, inn)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(soapReq))
	if err != nil {
		return nil, fmt.Errorf("ошибка формирования HTTP-запроса в АБС: %w", err)
	}

	httpReq.Header.Set("Content-Type", "text/xml; charset=utf-8")
	httpReq.Header.Set("SOAPAction", "")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.Error(err, "ABS request failed for INN %s to endpoint %s", inn, endpoint)
		return nil, fmt.Errorf("АБС банк недоступен (%s): %w", endpoint, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа АБС: %w", err)
	}
	rawResp := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("АБС вернул статус %d: %s", resp.StatusCode, rawResp)
	}

	clientInfo := parseColvirResponse(rawResp, inn)
	clientInfo.RawResponse = rawResp
	return clientInfo, nil
}

func parseColvirResponse(raw string, inn string) *ABSClientInfo {
	info := &ABSClientInfo{
		INN:        inn,
		ClientType: domain.ClientTypeLegalEntity,
		Phones:     make([]string, 0),
		Accounts:   make([]string, 0),
	}

	cleanINN := strings.TrimSpace(inn)
	if len(cleanINN) == 14 {
		info.ClientType = domain.ClientTypeIndividual
	} else {
		info.ClientType = domain.ClientTypeLegalEntity
	}

	reportData := raw
	reData := regexp.MustCompile(`(?s)<(?:.*:)?(?:reportData|data|return)[^>]*>(.*?)</(?:.*:)?(?:reportData|data|return)>`)
	if match := reData.FindStringSubmatch(raw); len(match) > 1 {
		reportData = match[1]
	}
	reportData = html.UnescapeString(reportData)

	reName := regexp.MustCompile(`(?i)<(?:.*:)?(?:NAME|CLI_NAME|FULL_NAME|CLIENT_NAME)[^>]*>([^<]+)</`)
	if match := reName.FindStringSubmatch(reportData); len(match) > 1 {
		info.FullName = strings.TrimSpace(match[1])
	}

	rePhones := regexp.MustCompile(`(?i)<(?:.*:)?(?:PHONE|TEL|MOBILE|CLI_PHONE)[^>]*>([^<]+)</`)
	phoneMatches := rePhones.FindAllStringSubmatch(reportData, -1)
	for _, m := range phoneMatches {
		if len(m) > 1 {
			val := strings.TrimSpace(m[1])
			if val != "" {
				info.Phones = append(info.Phones, val)
			}
		}
	}

	reAcc := regexp.MustCompile(`(?i)<(?:.*:)?(?:ACC|ACCOUNT|ACCOUNT_NUMBER|CODE)[^>]*>([0-9]{16,28})</`)
	accMatches := reAcc.FindAllStringSubmatch(reportData, -1)
	for _, m := range accMatches {
		if len(m) > 1 {
			val := strings.TrimSpace(m[1])
			if val != "" {
				info.Accounts = append(info.Accounts, val)
			}
		}
	}

	if info.FullName == "" && strings.Contains(reportData, ";") {
		lines := strings.Split(reportData, "\n")
		for _, line := range lines {
			parts := strings.Split(line, ";")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if len(p) > 3 && info.FullName == "" && !strings.Contains(p, "=") {
					info.FullName = p
				}
				if regexp.MustCompile(`^[0-9]{16,28}$`).MatchString(p) {
					info.Accounts = append(info.Accounts, p)
				}
				if regexp.MustCompile(`^\+?[0-9]{7,15}$`).MatchString(p) {
					info.Phones = append(info.Phones, p)
				}
			}
		}
	}

	if len(cleanINN) != 9 && len(cleanINN) != 14 {
		upperName := strings.ToUpper(info.FullName)
		if strings.Contains(upperName, "ЧДММ") || strings.Contains(upperName, "ҶДММ") ||
			strings.Contains(upperName, "ООО") || strings.Contains(upperName, "ЗАО") ||
			strings.Contains(upperName, "ОАО") || strings.Contains(upperName, "СП") {
			info.ClientType = domain.ClientTypeLegalEntity
		} else if strings.Contains(upperName, "ИП") || strings.Contains(upperName, "СОХИБКОР") ||
			strings.Contains(upperName, "СОҲИБКОР") {
			info.ClientType = domain.ClientTypeIndividual
		}
	}

	return info
}
