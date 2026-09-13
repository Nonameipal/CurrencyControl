package abs

import (
	"testing"

	"CurrencyControl/internal/domain"
)

func TestParseColvirResponse(t *testing.T) {
	rawXML := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
<soap:Body>
	<rpt:loadColvirReportDataResponseElem xmlns:rpt="http://bus.colvir.com/service/statement/v1">
		<rpt:result>
			<rpt:cReportItem>
				<rpt:code>Z_342_CLI_INFO_BYPH2</rpt:code>
				<rpt:reportData>&lt;MT94x&gt;&lt;TITLE&gt;
					&lt;S_CLI_PH1_NUM&gt;+992901112233&lt;/S_CLI_PH1_NUM&gt;
					&lt;S_CLI_TYPE_NAME&gt;Юридические лица&lt;/S_CLI_TYPE_NAME&gt;
					&lt;S_CLI_SURNAME&gt;ООО "СОХИБКОР"&lt;/S_CLI_SURNAME&gt;
					&lt;S_ACC_NUM&gt;20202972000000000001&lt;/S_ACC_NUM&gt;
				&lt;/TITLE&gt;&lt;/MT94x&gt;</rpt:reportData>
			</rpt:cReportItem>
		</rpt:result>
	</rpt:loadColvirReportDataResponseElem>
</soap:Body>
</soap:Envelope>`

	res := parseColvirResponse(rawXML, "010001234")
	if res.ClientType != domain.ClientTypeLegalEntity {
		t.Errorf("expected %s, got %s", domain.ClientTypeLegalEntity, res.ClientType)
	}
	if res.FullName != `ООО "СОХИБКОР"` {
		t.Errorf("expected ООО \"СОХИБКОР\", got %s", res.FullName)
	}
	if len(res.Phones) != 1 || res.Phones[0] != "+992901112233" {
		t.Errorf("expected [+992901112233], got %v", res.Phones)
	}
	if len(res.Accounts) != 1 || res.Accounts[0] != "20202972000000000001" {
		t.Errorf("expected [20202972000000000001], got %v", res.Accounts)
	}

	// Test individual
	rawXMLInd := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
<soap:Body>
	<rpt:loadColvirReportDataResponseElem xmlns:rpt="http://bus.colvir.com/service/statement/v1">
		<rpt:result>
			<rpt:cReportItem>
				<rpt:reportData>&lt;MT94x&gt;&lt;TITLE&gt;
					&lt;S_CLI_TYPE_NAME&gt;Физические лица&lt;/S_CLI_TYPE_NAME&gt;
				&lt;/TITLE&gt;&lt;/MT94x&gt;</rpt:reportData>
			</rpt:cReportItem>
		</rpt:result>
	</rpt:loadColvirReportDataResponseElem>
</soap:Body>
</soap:Envelope>`
	resInd := parseColvirResponse(rawXMLInd, "12345678901234")
	if resInd.ClientType != domain.ClientTypeIndividual {
		t.Errorf("expected %s for individual, got %s", domain.ClientTypeIndividual, resInd.ClientType)
	}
}
