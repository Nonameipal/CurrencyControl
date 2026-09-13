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
		INN:      inn,
		Phones:   make([]string, 0),
		Accounts: make([]string, 0),
	}

	reReportData := regexp.MustCompile(`(?s)<(?:[^:>]+:)?reportData[^>]*>(.*?)</(?:[^:>]+:)?reportData>`)
	match := reReportData.FindStringSubmatch(raw)
	if len(match) < 2 {
		logger.Error(nil, "ABS: reportData block not found in response for INN %s", inn)
		return info
	}
	inner := html.UnescapeString(match[1])

	getField := func(tagName string) string {
		re := regexp.MustCompile(`(?i)<` + tagName + `>([^<]*)</` + tagName + `>`)
		m := re.FindStringSubmatch(inner)
		if len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
		return ""
	}

	surname    := getField("S_CLI_SURNAME")
	firstName  := getField("S_CLI_NAME")
	patronymic := getField("S_CLI_PATRONYMIC")
	parts := []string{}
	for _, p := range []string{surname, firstName, patronymic} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	info.FullName = strings.Join(parts, " ")

	typeName := strings.ToLower(getField("S_CLI_TYPE_NAME"))
	switch {
	case strings.Contains(typeName, "физ"):
		info.ClientType = domain.ClientTypeIndividual
	case strings.Contains(typeName, "юр"):
		info.ClientType = domain.ClientTypeLegalEntity
	default:
		if len(strings.TrimSpace(inn)) == 14 {
			info.ClientType = domain.ClientTypeIndividual
		} else {
			info.ClientType = domain.ClientTypeLegalEntity
		}
	}

	phone1 := getField("S_CLI_PH1_NUM")
	if phone1 != "" {
		info.Phones = append(info.Phones, phone1)
	}
	phone2 := getField("S_CLI_PH2_NUM")
	if phone2 != "" {
		info.Phones = append(info.Phones, phone2)
	}

	reAcc := regexp.MustCompile(`(?i)<S_ACC_NUM>([0-9]{16,28})</S_ACC_NUM>`)
	for _, m := range reAcc.FindAllStringSubmatch(inner, -1) {
		if len(m) > 1 && m[1] != "" {
			info.Accounts = append(info.Accounts, m[1])
		}
	}

	return info
}
