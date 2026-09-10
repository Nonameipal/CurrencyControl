package abs

import (
	"testing"

	"CurrencyControl/internal/domain"
)

func TestParseColvirResponse(t *testing.T) {
	rawXML := `<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
<soapenv:Body>
	<v1:loadColvirReportDataElemResponse xmlns:v1="http://bus.colvir.com/service/statement/v1">
		<v1:reportData>
			&lt;ROW&gt;
				&lt;CLI_NAME&gt;ООО "СОХИБКОР"&lt;/CLI_NAME&gt;
				&lt;PHONE&gt;+992901112233&lt;/PHONE&gt;
				&lt;ACCOUNT&gt;20202972000000000001&lt;/ACCOUNT&gt;
				&lt;OPERATOR&gt;Исмоилов А.М.&lt;/OPERATOR&gt;
			&lt;/ROW&gt;
		</v1:reportData>
	</v1:loadColvirReportDataElemResponse>
</soapenv:Body>
</soapenv:Envelope>`

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
	if res.Operator != "Исмоилов А.М." {
		t.Errorf("expected Исмоилов А.М., got %s", res.Operator)
	}

	// Test 14-digit PINFL individual
	resInd := parseColvirResponse(rawXML, "12345678901234")
	if resInd.ClientType != domain.ClientTypeIndividual {
		t.Errorf("expected %s for 14-digit INN, got %s", domain.ClientTypeIndividual, resInd.ClientType)
	}
}
