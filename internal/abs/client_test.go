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

const sampleSoleProprietorXML = `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
   <soap:Body>
      <ns2:loadClientsListElemResponse xmlns:ns2="http://bus.colvir.com/service/clients/v1">
         <client>
            <type>individual</type>
            <taxIdentificationNumber>
               <code>045931660</code>
            </taxIdentificationNumber>
            <longName>СИ БУРЯК ИВАН</longName>
            <longName>СИ БУРЯК ИВАН РАДИОНОВИЧ</longName>
            <longName>SI BURYAK IVAN</longName>
            <contactData>
               <type>
                  <code>PHN</code>
               </type>
               <value>992918616796</value>
            </contactData>
         </client>
      </ns2:loadClientsListElemResponse>
   </soap:Body>
</soap:Envelope>`

func TestParseClientXML_SoleProprietor(t *testing.T) {
	res, err := ParseClientXML([]byte(sampleSoleProprietorXML), "045931660")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.INN != "045931660" {
		t.Errorf("INN: got %q, want %q", res.INN, "045931660")
	}
	if res.FullName != "БУРЯК ИВАН РАДИОНОВИЧ" {
		t.Errorf("FullName: got %q, want %q", res.FullName, "БУРЯК ИВАН РАДИОНОВИЧ")
	}
	if res.ClientType != "Индивидуальный предприниматель" {
		t.Errorf("ClientType: got %q, want %q", res.ClientType, "Индивидуальный предприниматель")
	}
	if len(res.Phones) == 0 || res.Phones[0] != "992918616796" {
		t.Errorf("Phones: got %v, want [992918616796]", res.Phones)
	}
}

func TestCleanSoleProprietorPrefix(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantIsSP    bool
	}{
		{"СИ БУРЯК ИВАН РАДИОНОВИЧ", "БУРЯК ИВАН РАДИОНОВИЧ", true},
		{"СИ. БУРЯК ИВАН РАДИОНОВИЧ", "БУРЯК ИВАН РАДИОНОВИЧ", true},
		{"СИ.БУРЯК ИВАН РАДИОНОВИЧ", "БУРЯК ИВАН РАДИОНОВИЧ", true},
		{"Си Буряк Иван Радионович", "Буряк Иван Радионович", true},
		{"си   буряк   иван", "буряк   иван", true},
		{"CI BURYAK IVAN", "BURYAK IVAN", true},
		{"SI BURYAK IVAN", "BURYAK IVAN", true},
		{"ИП Буряк Иван Радионович", "Буряк Иван Радионович", true},
		{"СИ- Буряк Иван Радионович", "Буряк Иван Радионович", true},
		// False positive checks: ordinary names starting with "СИ" must NOT be modified
		{"СИДОРОВ АЛЕКСЕЙ", "СИДОРОВ АЛЕКСЕЙ", false},
		{"СИМОНОВ СЕРГЕЙ", "СИМОНОВ СЕРГЕЙ", false},
		{"СИНОИ РАҲИМ", "СИНОИ РАҲИМ", false},
		{"СИТОРАИ ШАБ", "СИТОРАИ ШАБ", false},
		{"Рачабов Некрузчон Саймухторович", "Рачабов Некрузчон Саймухторович", false},
		{"", "", false},
		{"СИ", "СИ", false},
		{"   ", "", false},
	}

	for _, tt := range tests {
		gotName, gotIsSP := CleanSoleProprietorPrefix(tt.input)
		if gotName != tt.wantName || gotIsSP != tt.wantIsSP {
			t.Errorf("CleanSoleProprietorPrefix(%q) = (%q, %v), want (%q, %v)",
				tt.input, gotName, gotIsSP, tt.wantName, tt.wantIsSP)
		}
	}
}

