package abs

import (
	"testing"
)

const sampleCorporateXML = `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
   <soap:Body>
      <ns2:loadClientsListElemResponse xmlns:ns2="http://bus.colvir.com/service/clients/v1">
         <client>
            <type>corporate</type>
            <taxIdentificationNumber>
               <code>040008291</code>
            </taxIdentificationNumber>
            <longName>ҶДММ "АКТИВ БАНК"</longName>
            <longName>JSC "ACTIVE BANK"</longName>
            <longName>АКТИВ БАНК</longName>
            <contactData>
               <type>
                  <code>PHN</code>
               </type>
               <value>+992446000000</value>
            </contactData>
         </client>
      </ns2:loadClientsListElemResponse>
   </soap:Body>
</soap:Envelope>`

const sampleIndividualXML = `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
   <soap:Body>
      <ns2:loadClientsListElemResponse xmlns:ns2="http://bus.colvir.com/service/clients/v1">
         <client>
            <type>individual</type>
            <taxIdentificationNumber>
               <code>987654321</code>
            </taxIdentificationNumber>
            <longName>Рачабов Некрузчон</longName>
            <longName>Рачабов Некрузчон Саймухторович</longName>
            <longName>Rajabov Nekruzjon</longName>
            <contactData>
               <type>
                  <code>PHN</code>
               </type>
               <value>992714475050</value>
            </contactData>
         </client>
      </ns2:loadClientsListElemResponse>
   </soap:Body>
</soap:Envelope>`

func TestParseClientXML_Corporate(t *testing.T) {
	res, err := ParseClientXML([]byte(sampleCorporateXML), "040008291")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.INN != "040008291" {
		t.Errorf("INN: got %q, want %q", res.INN, "040008291")
	}
	// For corporate: index 0 (1st longName)
	if res.FullName != `ҶДММ "АКТИВ БАНК"` {
		t.Errorf("FullName: got %q, want %q", res.FullName, `ҶДММ "АКТИВ БАНК"`)
	}
	if res.ClientType != "Юридическое лицо" {
		t.Errorf("ClientType: got %q, want %q", res.ClientType, "Юридическое лицо")
	}
	if len(res.Phones) == 0 || res.Phones[0] != "+992446000000" {
		t.Errorf("Phones: got %v, want [+992446000000]", res.Phones)
	}
}

func TestParseClientXML_Individual(t *testing.T) {
	res, err := ParseClientXML([]byte(sampleIndividualXML), "987654321")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.INN != "987654321" {
		t.Errorf("INN: got %q, want %q", res.INN, "987654321")
	}
	// For individual: index 1 (2nd longName)
	if res.FullName != "Рачабов Некрузчон Саймухторович" {
		t.Errorf("FullName: got %q, want %q", res.FullName, "Рачабов Некрузчон Саймухторович")
	}
	if res.ClientType != "Физическое лицо" {
		t.Errorf("ClientType: got %q, want %q", res.ClientType, "Физическое лицо")
	}
	if len(res.Phones) == 0 || res.Phones[0] != "992714475050" {
		t.Errorf("Phones: got %v, want [992714475050]", res.Phones)
	}
}

func TestParseClientXML_NotFound(t *testing.T) {
	emptyXML := `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body/></soap:Envelope>`
	_, err := ParseClientXML([]byte(emptyXML), "123")
	if err == nil {
		t.Fatal("expected error for empty XML, got nil")
	}
}
