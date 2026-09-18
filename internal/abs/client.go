package abs

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"CurrencyControl/internal/configs"
	"CurrencyControl/internal/domain"
)


type ABSClientInfo struct {
	INN        string   `json:"inn"`
	FullName   string   `json:"full_name"`
	ClientType string   `json:"client_type"`
	Phones     []string `json:"phones"`
	Accounts   []string `json:"accounts"`
}

type ABSClient interface {
	GetClientByINN(ctx context.Context, inn string) (*ABSClientInfo, error)
}

type absClient struct {
	endpoint   string
	httpClient *http.Client
}

func NewClient(cfg configs.ABSParams) ABSClient {
	ep := cfg.Endpoint
	if ep == "" {
		ep = "http://10.64.20.34:8181/cxf/statement/v1"
	}
	return &absClient{
		endpoint:   ep,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
type soapEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    soapBody
}
type soapBody struct {
	XMLName  xml.Name    `xml:"Body"`
	Response *reportResp `xml:"loadColvirReportDataResponseElem"`
	Fault    *soapFault  `xml:"Fault"`
}
type soapFault struct {
	FaultString string `xml:"faultstring"`
}

type reportResp struct {
	ReportData string `xml:"result>cReportItem>reportData"`
}
func (c *absClient) GetClientByINN(ctx context.Context, inn string) (*ABSClientInfo, error) {
	inn = strings.TrimSpace(inn)
	if inn == "" {
		return nil, fmt.Errorf("ИНН не может быть пустым")
	}
	reqBody := fmt.Sprintf(`<soapenv:Envelope
    xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:v1="http://bus.colvir.com/service/statement/v1"
    xmlns:v11="http://bus.colvir.com/common/support/v1">
  <soapenv:Header/>
  <soapenv:Body>
    <v1:loadColvirReportDataElem>
      <v11:head>
        <v11:requestId>%s</v11:requestId>
        <v11:sessionId>session-%s</v11:sessionId>
        <v11:processId>process-%s</v11:processId>
        <v11:params>
          <v11:clientType>CBS</v11:clientType>
          <v11:interfaceVersion>1.0</v11:interfaceVersion>
          <v11:language>ru</v11:language>
          <v11:operationalDate>%s</v11:operationalDate>
        </v11:params>
      </v11:head>
      <v1:reportCode>Z_342_CLI_INFO_BYPH2</v1:reportCode>
      <v1:reportParams>prmS_CLI_TAX_Code=&gt;%s</v1:reportParams>
      <v1:rawFormat>true</v1:rawFormat>
    </v1:loadColvirReportDataElem>
  </soapenv:Body>
</soapenv:Envelope>`,
		newUUID(), newUUID(), newUUID(),
		time.Now().Format("2006-01-02T15:04:05"),
		inn,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString(reqBody))
	if err != nil {
		return nil, fmt.Errorf("формирование запроса к CBS: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Accept", "text/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CBS недоступен: %w", err)
	}
	defer resp.Body.Close()

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("чтение ответа CBS: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CBS вернул статус %d", resp.StatusCode)
	}

	var envelope soapEnvelope
	if err := xml.Unmarshal(rawBytes, &envelope); err != nil {
		return nil, fmt.Errorf("разбор SOAP-ответа: %w", err)
	}

	if f := envelope.Body.Fault; f != nil {
		return nil, fmt.Errorf("CBS ошибка: %s", f.FaultString)
	}

	if envelope.Body.Response == nil || envelope.Body.Response.ReportData == "" {
		return nil, fmt.Errorf("клиент с ИНН '%s' не найден в CBS", inn)
	}

	return parseReportData(envelope.Body.Response.ReportData, inn)
}
func parseReportData(data, inn string) (*ABSClientInfo, error) {
	type reportXML struct {
		XMLName    xml.Name `xml:"MT94x"`
		Surname    string   `xml:"TITLE>S_CLI_SURNAME"`
		Name       string   `xml:"TITLE>S_CLI_NAME"`
		Patronymic string   `xml:"TITLE>S_CLI_PATRONYMIC"`
		TaxCode    string   `xml:"TITLE>S_CLI_TAX_CODE"`
		TypeName   string   `xml:"TITLE>S_CLI_TYPE_NAME"`
		Phone1     string   `xml:"TITLE>S_CLI_PH1_NUM"`
		Phone2     string   `xml:"TITLE>S_CLI_PH2_NUM"`
	}

	var r reportXML
	if err := xml.Unmarshal([]byte(data), &r); err != nil {
		return nil, fmt.Errorf("разбор reportData: %w", err)
	}

	fullName := strings.TrimSpace(strings.Join(filterEmpty(r.Surname, r.Name, r.Patronymic), " "))
	if fullName == "" {
		return nil, fmt.Errorf("клиент с ИНН '%s' не найден в CBS", inn)
	}

	parsedINN := strings.TrimSpace(r.TaxCode)
	if parsedINN == "" {
		parsedINN = inn
	}

	clientType := domain.ClientTypeLegalEntity
	if strings.Contains(strings.ToLower(r.TypeName), "физ") {
		clientType = domain.ClientTypeIndividual
	}

	return &ABSClientInfo{
		INN:        parsedINN,
		FullName:   fullName,
		ClientType: clientType,
		Phones:     unique(r.Phone1, r.Phone2),
		Accounts:   []string{},
	}, nil
}

func filterEmpty(parts ...string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func unique(parts ...string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
