package abs

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"CurrencyControl/internal/configs"
)

type ABSClientInfo struct {
	INN        string   `json:"inn"`
	FullName   string   `json:"full_name"`
	ClientType string   `json:"client_type"`
	Phones     []string `json:"phones"`
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
		ep = "http://10.64.20.34:8181/cxf/clients/v1"
	}
	return &absClient{
		endpoint:   ep,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

const soapTemplate = `<soapenv:Envelope 
    xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:v1="http://bus.colvir.com/service/clients/v1"
    xmlns:sup="http://bus.colvir.com/common/support/v1"
    xmlns:q="http://bus.colvir.com/common/query/v1">
   <soapenv:Header/>
   <soapenv:Body>
      <v1:loadClientsListElem>
         <sup:head>
            <sup:requestId>%s</sup:requestId>
            <sup:params>
               <sup:clientType>CBS</sup:clientType>
               <sup:interfaceVersion>1.0</sup:interfaceVersion>
               <sup:language>ru</sup:language>
               <sup:operationalDate>%s</sup:operationalDate>
            </sup:params>
         </sup:head>
         <v1:taxIdentificationNumber>%s</v1:taxIdentificationNumber>
      </v1:loadClientsListElem>
   </soapenv:Body>
</soapenv:Envelope>`

type XMLNode struct {
	XMLName xml.Name
	Content string     `xml:",chardata"`
	Attrs   []xml.Attr `xml:",any,attr"`
	Nodes   []XMLNode  `xml:",any"`
}

func findFirst(n *XMLNode, name string) *XMLNode {
	if n.XMLName.Local == name {
		return n
	}

	for i := range n.Nodes {
		if found := findFirst(&n.Nodes[i], name); found != nil {
			return found
		}
	}

	return nil
}

func findAll(n *XMLNode, name string, out *[]*XMLNode) {
	if n.XMLName.Local == name {
		*out = append(*out, n)
	}

	for i := range n.Nodes {
		findAll(&n.Nodes[i], name, out)
	}
}

func (c *absClient) GetClientByINN(ctx context.Context, inn string) (*ABSClientInfo, error) {
	inn = strings.TrimSpace(inn)
	if inn == "" {
		return nil, fmt.Errorf("ИНН не может быть пустым")
	}

	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	opDate := time.Now().Format("2006-01-02T15:04:05")

	reqBody := fmt.Sprintf(
		soapTemplate,
		requestID,
		opDate,
		inn,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса к АБС: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("АБС недоступен: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа АБС: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("АБС вернул статус %d: %s", resp.StatusCode, string(respBody))
	}

	return ParseClientXML(respBody, inn)
}

func ParseClientXML(respBody []byte, targetINN string) (*ABSClientInfo, error) {
	var root XMLNode
	if err := xml.Unmarshal(respBody, &root); err != nil {
		return nil, fmt.Errorf("ошибка парсинга XML ответа АБС: %w", err)
	}

	var clientTypeRaw string
	var typeNodes []*XMLNode
	findAll(&root, "type", &typeNodes)

	for _, tn := range typeNodes {
		if len(tn.Nodes) == 0 && strings.TrimSpace(tn.Content) != "" {
			clientTypeRaw = strings.TrimSpace(tn.Content)
			break
		}
	}

	var clientType string
	switch clientTypeRaw {
	case "individual":
		clientType = "Физическое лицо"
	case "corporate", "legal":
		clientType = "Юридическое лицо"
	default:
		clientType = clientTypeRaw
	}

	var longName string
	var longNameNodes []*XMLNode
	findAll(&root, "longName", &longNameNodes)

	switch clientTypeRaw {
	case "individual":
		if len(longNameNodes) > 1 {
			longName = strings.TrimSpace(longNameNodes[1].Content)
		} else if len(longNameNodes) > 0 {
			longName = strings.TrimSpace(longNameNodes[0].Content)
		}
	case "corporate", "legal":
		if len(longNameNodes) > 0 {
			longName = strings.TrimSpace(longNameNodes[0].Content)
		}
	default:
		if len(longNameNodes) > 0 {
			longName = strings.TrimSpace(longNameNodes[0].Content)
		}
	}

	var parsedINN string
	if n := findFirst(&root, "taxIdentificationNumber"); n != nil {
		if code := findFirst(n, "code"); code != nil {
			parsedINN = strings.TrimSpace(code.Content)
		} else if strings.TrimSpace(n.Content) != "" {
			parsedINN = strings.TrimSpace(n.Content)
		}
	}
	if parsedINN == "" {
		parsedINN = targetINN
	}

	var phone string
	var contactNodes []*XMLNode
	findAll(&root, "contactData", &contactNodes)

	for _, cd := range contactNodes {
		typeN := findFirst(cd, "type")
		if typeN == nil {
			continue
		}
		codeN := findFirst(typeN, "code")
		if codeN != nil && strings.TrimSpace(codeN.Content) == "PHN" {
			if valN := findFirst(cd, "value"); valN != nil {
				phone = strings.TrimSpace(valN.Content)
				break
			}
		}
	}

	if longName == "" && phone == "" && len(typeNodes) == 0 {
		return nil, fmt.Errorf("клиент с ИНН '%s' не найден в АБС", targetINN)
	}

	var phones []string
	if phone != "" {
		phones = []string{phone}
	} else {
		phones = []string{}
	}

	return &ABSClientInfo{
		INN:        parsedINN,
		FullName:   longName,
		ClientType: clientType,
		Phones:     phones,
	}, nil
}

